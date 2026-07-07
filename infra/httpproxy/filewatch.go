package httpproxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.yaml.in/yaml/v3"
)

// fileConfig distinguishes keys absent from the file (fall back to the
// process environment) from keys explicitly set to an empty value (disable
// the proxy for that class of requests).
type fileConfig struct {
	HTTPProxy  *string `yaml:"http_proxy"`
	HTTPSProxy *string `yaml:"https_proxy"`
	NoProxy    *string `yaml:"no_proxy"`
}

// WatchFile applies the settings file at path and keeps watching it for
// changes until ctx is done; run it in its own goroutine — setup failures
// are logged before being returned, so a fire-and-forget
// `go mgr.WatchFile(...)` never fails silently. An empty path disables
// watching and keeps the environment-based settings.
//
// The file is YAML (JSON is accepted too):
//
//	http_proxy: "http://user:pass@10.0.1.1:3128"
//	https_proxy: "http://10.0.1.1:3128"
//	no_proxy: "localhost,127.0.0.1,.svc,10.0.0.0/8"
//
// A missing file, or a key missing from it, falls back to the corresponding
// environment variable; deleting the file reverts to environment settings.
// Unknown keys, files that fail to parse and settings that fail validation
// are logged and ignored, keeping the last good settings.
//
// The watch is placed on the parent directory and any event in it triggers a
// debounced reload (unchanged settings are a no-op), so rename-replace
// updates (mv tmp file) and even
// replacement of the directory itself are all picked up.
func (m *Manager) WatchFile(ctx context.Context, path string) error {
	err := m.watchFile(ctx, path)
	if err != nil {
		m.log.Error("httpproxy: settings file watch failed; hot reload disabled",
			"file", path, "error", err)
	}
	return err
}

func (m *Manager) watchFile(ctx context.Context, path string) error {
	if path == "" {
		m.log.Info("httpproxy: no settings file configured; using environment settings")
		return nil
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("httpproxy: resolve %q: %w", path, err)
	}
	dir := filepath.Dir(path)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("httpproxy: start watcher: %w", err)
	}
	defer watcher.Close()
	if err = watcher.Add(dir); err != nil {
		return fmt.Errorf("httpproxy: watch %q: %w", dir, err)
	}
	m.log.Info("httpproxy: watching settings file", "file", path)
	// Log each distinct problem once, not on every event: repeated logging
	// would spam on a persistently broken file, and if the service logs
	// into the watched directory it would even re-trigger itself.
	lastErr := ""
	apply := func() {
		err := m.applyFile(path)
		switch {
		case err == nil:
			lastErr = ""
		case err.Error() != lastErr:
			lastErr = err.Error()
			m.log.Error("httpproxy: keeping previous settings", "file", path, "error", err)
		}
	}
	apply()

	reload := time.NewTimer(m.debounce)
	reload.Stop()
	// dirStale is set when the watched directory itself is removed or
	// renamed: the OS keeps the watch bound to the old inode (or drops it),
	// so it must be re-established on the current path.
	dirStale := false
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// Attribute-only events are noise; worse, reading the file
			// updates atime on some platforms (macOS kqueue reports it as
			// Chmod), so reacting to them would re-trigger reload forever.
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 && filepath.Clean(event.Name) == dir {
				dirStale = true
			}
			reload.Reset(m.debounce)
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			m.log.Error("httpproxy: watch error", "file", path, "error", err)
		case <-reload.C:
			apply()
			if dirStale || !slices.Contains(watcher.WatchList(), dir) {
				_ = watcher.Remove(dir)
				if err = watcher.Add(dir); err != nil {
					m.log.Error("httpproxy: re-watch failed", "dir", dir, "error", err)
					reload.Reset(time.Second)
				} else {
					dirStale = false
					// The file may have changed before the watch was
					// re-established; confirm once more.
					reload.Reset(time.Second)
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (m *Manager) applyFile(path string) error {
	cfg, err := loadFile(path)
	if err == nil {
		err = m.Update(cfg)
	}
	return err
}

// loadFile reads the settings file and merges it over the process
// environment. A missing or empty file yields plain environment settings.
func loadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return FromEnvironment(), nil
	}
	if err != nil {
		return Config{}, err
	}
	var file fileConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	// Reject misspelled keys: a typo like "htttp_proxy" must be an error,
	// not a silently ignored setting.
	dec.KnownFields(true)
	if err = dec.Decode(&file); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg := FromEnvironment()
	if file.HTTPProxy != nil {
		cfg.HTTPProxy = *file.HTTPProxy
	}
	if file.HTTPSProxy != nil {
		cfg.HTTPSProxy = *file.HTTPSProxy
	}
	if file.NoProxy != nil {
		cfg.NoProxy = *file.NoProxy
	}
	return cfg, nil
}
