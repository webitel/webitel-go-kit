// Package httpproxy provides hot-reloadable forward-proxy settings
// (http_proxy / https_proxy / no_proxy) for outbound HTTP clients.
//
// A Manager holds the current settings in an atomic snapshot and exposes a
// Proxy method assignable to http.Transport.Proxy. The transport consults
// that function on every request, so settings changed at runtime — via
// Update or a watched config file (WatchFile) — take effect immediately,
// without rebuilding clients or restarting the service. After every applied
// change the Manager closes idle connections on all registered transports,
// so pooled keep-alive connections do not keep using the previous route.
//
// Typical wiring at service startup:
//
//	mgr := httpproxy.NewManager(httpproxy.WithLogger(log))
//	if err := mgr.HookDefaultTransport(); err != nil { ... }
//	go mgr.WatchFile(ctx, os.Getenv("PROXY_CONFIG_FILE"))
//
// and for custom transports:
//
//	t := mgr.Transport()            // clone of http.DefaultTransport
//	t2 := mgr.WrapTransport(custom) // caller-built *http.Transport
package httpproxy

import (
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

const defaultDebounce = 250 * time.Millisecond

// IdleCloser is the subset of *http.Transport used to invalidate pooled
// connections after a settings change.
type IdleCloser interface {
	CloseIdleConnections()
}

// Manager holds the current proxy settings and the set of transports that
// depend on them. The zero value is not usable; use NewManager.
type Manager struct {
	log      *slog.Logger
	debounce time.Duration

	// updateMu serializes Update calls; snap stays atomic for lock-free reads.
	updateMu sync.Mutex
	snap     atomic.Pointer[snapshot]

	mu         sync.Mutex
	transports []IdleCloser

	// shared is the lazily-built transport behind Client, so per-request
	// Client calls reuse one connection pool and one registration.
	sharedOnce sync.Once
	shared     *http.Transport
}

type snapshot struct {
	cfg   Config
	proxy func(*url.URL) (*url.URL, error)
}

// Option configures a Manager.
type Option func(*Manager)

// WithLogger sets the logger used for settings changes and watch errors.
// Defaults to slog.Default().
func WithLogger(log *slog.Logger) Option {
	return func(m *Manager) {
		if log != nil {
			m.log = log
		}
	}
}

// WithDebounce sets how long WatchFile coalesces bursts of file events
// before reloading. Defaults to 250ms.
func WithDebounce(d time.Duration) Option {
	return func(m *Manager) {
		if d > 0 {
			m.debounce = d
		}
	}
}

// NewManager returns a Manager initialized from the process environment
// (HTTP_PROXY, HTTPS_PROXY, NO_PROXY).
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		log:      slog.Default(),
		debounce: defaultDebounce,
	}
	for _, opt := range opts {
		opt(m)
	}
	m.snap.Store(newSnapshot(FromEnvironment()))
	return m
}

func newSnapshot(c Config) *snapshot {
	return &snapshot{cfg: c, proxy: c.proxyFunc()}
}

// Proxy resolves the proxy URL for a single request against the current
// settings. Assign it to http.Transport.Proxy: because the transport calls
// it on every request, long-lived clients pick up settings changes without
// being rebuilt.
func (m *Manager) Proxy(req *http.Request) (*url.URL, error) {
	return m.snap.Load().proxy(req.URL)
}

// Current returns the settings in effect.
func (m *Manager) Current() Config {
	return m.snap.Load().cfg
}

// Update validates the settings and atomically makes them current, then
// closes idle connections on all registered transports so that new requests
// re-dial via the new route. On validation error the previous settings stay
// in effect. Unchanged settings are a no-op.
func (m *Manager) Update(c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	m.updateMu.Lock()
	defer m.updateMu.Unlock()
	prev := m.snap.Load().cfg
	if prev == c {
		return nil
	}
	m.snap.Store(newSnapshot(c))
	m.closeIdleConnections()
	m.log.Info("httpproxy: settings changed",
		slog.Group("old",
			"http_proxy", maskProxyURL(prev.HTTPProxy),
			"https_proxy", maskProxyURL(prev.HTTPSProxy),
			"no_proxy", prev.NoProxy),
		slog.Group("new",
			"http_proxy", maskProxyURL(c.HTTPProxy),
			"https_proxy", maskProxyURL(c.HTTPSProxy),
			"no_proxy", c.NoProxy),
	)
	return nil
}

// Register adds a long-lived transport to the set whose idle connections are
// closed after every settings change. Transports obtained from Transport,
// WrapTransport and HookDefaultTransport are registered automatically.
// Do not register short-lived per-request transports: the Manager keeps the
// reference forever, so they would leak — set Proxy on them directly instead.
func (m *Manager) Register(t IdleCloser) {
	if t == nil {
		return
	}
	m.mu.Lock()
	m.transports = append(m.transports, t)
	m.mu.Unlock()
}

func (m *Manager) closeIdleConnections() {
	m.mu.Lock()
	transports := slices.Clone(m.transports)
	m.mu.Unlock()
	for _, t := range transports {
		t.CloseIdleConnections()
	}
}
