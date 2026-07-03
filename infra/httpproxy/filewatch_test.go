package httpproxy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFile(t *testing.T) {
	tests := []struct {
		name    string
		missing bool   // no file on disk at all
		content string // file body when missing == false
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name:    "file overrides env, absent key falls back, empty key disables",
			content: "http_proxy: \"http://file-proxy:3128\"\nno_proxy: \"\"\n",
			env: map[string]string{
				"HTTPS_PROXY": "http://env-proxy:3128",
				"NO_PROXY":    "env.host",
			},
			want: Config{
				HTTPProxy:  "http://file-proxy:3128",
				HTTPSProxy: "http://env-proxy:3128",
				NoProxy:    "",
			},
		},
		{
			name:    "json accepted",
			content: `{"http_proxy":"http://json-proxy:3128"}`,
			want:    Config{HTTPProxy: "http://json-proxy:3128"},
		},
		{
			name:    "comments allowed",
			content: "# corp proxy\nhttp_proxy: \"http://p:3128\"\n",
			want:    Config{HTTPProxy: "http://p:3128"},
		},
		{
			name:    "missing file falls back to environment",
			missing: true,
			env:     map[string]string{"HTTP_PROXY": "http://env-proxy:3128"},
			want:    Config{HTTPProxy: "http://env-proxy:3128"},
		},
		{
			name:    "empty file falls back to environment",
			content: "",
			env:     map[string]string{"HTTP_PROXY": "http://env-proxy:3128"},
			want:    Config{HTTPProxy: "http://env-proxy:3128"},
		},
		{
			name:    "misspelled key rejected",
			content: "htttp_proxy: \"http://p:3128\"\n",
			wantErr: true,
		},
		{
			name:    "malformed yaml rejected",
			content: "http_proxy: [broken\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearProxyEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			path := filepath.Join(t.TempDir(), "proxy.yml")
			if !tt.missing {
				writeFile(t, path, tt.content)
			}
			got, err := loadFile(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("loadFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("loadFile() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestWatchFileReloads(t *testing.T) {
	clearProxyEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "proxy.yml")
	writeFile(t, path, "http_proxy: \"http://proxy-a:3128\"\n")

	m := NewManager(WithDebounce(10 * time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- m.WatchFile(ctx, path) }()

	waitFor(t, "initial load", func() bool {
		return m.Current().HTTPProxy == "http://proxy-a:3128"
	})

	// In-place edit.
	writeFile(t, path, "http_proxy: \"http://proxy-b:3128\"\n")
	waitFor(t, "reload after edit", func() bool {
		return m.Current().HTTPProxy == "http://proxy-b:3128"
	})

	// Atomic rename-replace, the way ConfigMaps and editors update files.
	tmp := filepath.Join(dir, "proxy.yml.tmp")
	writeFile(t, tmp, "http_proxy: \"http://proxy-c:3128\"\n")
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "reload after rename-replace", func() bool {
		return m.Current().HTTPProxy == "http://proxy-c:3128"
	})

	// Malformed file keeps the last good settings.
	writeFile(t, path, "http_proxy: [broken\n")
	time.Sleep(100 * time.Millisecond)
	if got := m.Current().HTTPProxy; got != "http://proxy-c:3128" {
		t.Fatalf("malformed file replaced settings: %q", got)
	}

	// Deleting the file reverts to environment settings.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "revert to environment after delete", func() bool {
		return m.Current().HTTPProxy == ""
	})

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WatchFile returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WatchFile did not stop on context cancel")
	}
}

func TestWatchFileSurvivesDirectoryReplace(t *testing.T) {
	clearProxyEnv(t)
	root := t.TempDir()
	dir := filepath.Join(root, "conf")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "proxy.yml")
	writeFile(t, path, "http_proxy: \"http://proxy-a:3128\"\n")

	m := NewManager(WithDebounce(10 * time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.WatchFile(ctx, path)

	waitFor(t, "initial load", func() bool {
		return m.Current().HTTPProxy == "http://proxy-a:3128"
	})

	// Replace the whole directory, the way a config dir may be redeployed.
	if err := os.Rename(dir, dir+".bak"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "revert to environment after dir rename", func() bool {
		return m.Current().HTTPProxy == ""
	})
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, "http_proxy: \"http://proxy-b:3128\"\n")
	waitFor(t, "reload from recreated directory", func() bool {
		return m.Current().HTTPProxy == "http://proxy-b:3128"
	})
}

func TestWatchFileEmptyPathIsNoop(t *testing.T) {
	clearProxyEnv(t)
	m := NewManager()
	if err := m.WatchFile(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
}

func TestWatchFileMissingDirFails(t *testing.T) {
	clearProxyEnv(t)
	m := NewManager()
	err := m.WatchFile(context.Background(), filepath.Join(t.TempDir(), "nosuchdir", "proxy.yml"))
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}
