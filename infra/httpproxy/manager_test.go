package httpproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// clearProxyEnv isolates tests from the host's proxy environment.
func clearProxyEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"HTTP_PROXY", "http_proxy",
		"HTTPS_PROXY", "https_proxy",
		"NO_PROXY", "no_proxy",
		"REQUEST_METHOD",
	} {
		t.Setenv(key, "")
	}
}

func mustRequest(t *testing.T, target string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func proxyFor(t *testing.T, m *Manager, target string) *url.URL {
	t.Helper()
	proxyURL, err := m.Proxy(mustRequest(t, target))
	if err != nil {
		t.Fatalf("Proxy(%s): %v", target, err)
	}
	return proxyURL
}

func TestUpdateSwapsProxyResolution(t *testing.T) {
	clearProxyEnv(t)
	m := NewManager()

	if got := proxyFor(t, m, "http://example.invalid/"); got != nil {
		t.Fatalf("expected direct connection initially, got %v", got)
	}

	if err := m.Update(Config{HTTPProxy: "http://proxy-a:3128"}); err != nil {
		t.Fatal(err)
	}
	if got := proxyFor(t, m, "http://example.invalid/"); got == nil || got.Host != "proxy-a:3128" {
		t.Fatalf("expected proxy-a:3128, got %v", got)
	}

	if err := m.Update(Config{HTTPProxy: "http://proxy-b:3128"}); err != nil {
		t.Fatal(err)
	}
	if got := proxyFor(t, m, "http://example.invalid/"); got == nil || got.Host != "proxy-b:3128" {
		t.Fatalf("expected proxy-b:3128, got %v", got)
	}
}

func TestProxyResolution(t *testing.T) {
	noProxyCfg := Config{
		HTTPProxy: "http://proxy:3128",
		NoProxy:   "exact.host, .corp.example.com, 10.0.0.0/8",
	}
	tests := []struct {
		name   string
		cfg    Config
		target string
		want   string // resolved proxy URL; "" means direct
	}{
		{
			name:   "empty settings mean direct",
			cfg:    Config{},
			target: "http://example.invalid/",
			want:   "",
		},
		{
			name:   "bare host:port gets http scheme",
			cfg:    Config{HTTPProxy: "proxy-a:3128"},
			target: "http://example.invalid/",
			want:   "http://proxy-a:3128",
		},
		{
			name:   "https requests use https_proxy",
			cfg:    Config{HTTPProxy: "proxy-a:3128", HTTPSProxy: "https://proxy-b:3129"},
			target: "https://example.invalid/",
			want:   "https://proxy-b:3129",
		},
		{
			name:   "http requests ignore https_proxy",
			cfg:    Config{HTTPSProxy: "https://proxy-b:3129"},
			target: "http://example.invalid/",
			want:   "",
		},
		{
			name:   "no_proxy exact host",
			cfg:    noProxyCfg,
			target: "http://exact.host/",
			want:   "",
		},
		{
			name:   "no_proxy domain suffix",
			cfg:    noProxyCfg,
			target: "http://svc.corp.example.com/",
			want:   "",
		},
		{
			name:   "no_proxy CIDR",
			cfg:    noProxyCfg,
			target: "http://10.1.2.3/",
			want:   "",
		},
		{
			name:   "host outside no_proxy is proxied",
			cfg:    noProxyCfg,
			target: "http://proxied.invalid/",
			want:   "http://proxy:3128",
		},
		{
			name:   "suffix does not match unrelated domain",
			cfg:    noProxyCfg,
			target: "http://other.example.com:443/",
			want:   "http://proxy:3128",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearProxyEnv(t)
			m := NewManager()
			if err := m.Update(tt.cfg); err != nil {
				t.Fatal(err)
			}
			got := ""
			if proxyURL := proxyFor(t, m, tt.target); proxyURL != nil {
				got = proxyURL.String()
			}
			if got != tt.want {
				t.Errorf("Proxy(%s) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestUpdateInvalidKeepsPrevious(t *testing.T) {
	clearProxyEnv(t)
	m := NewManager()
	if err := m.Update(Config{HTTPProxy: "http://good:3128"}); err != nil {
		t.Fatal(err)
	}
	if err := m.Update(Config{HTTPProxy: "http://bad host:3128"}); err == nil {
		t.Fatal("expected validation error")
	}
	if got := m.Current().HTTPProxy; got != "http://good:3128" {
		t.Fatalf("previous settings lost: %q", got)
	}
}

type idleSpy struct{ calls atomic.Int32 }

func (s *idleSpy) CloseIdleConnections() { s.calls.Add(1) }

func TestUpdateClosesIdleConnectionsOnceOnChange(t *testing.T) {
	clearProxyEnv(t)
	m := NewManager()
	spy := &idleSpy{}
	m.Register(spy)

	if err := m.Update(Config{HTTPProxy: "http://proxy:3128"}); err != nil {
		t.Fatal(err)
	}
	if got := spy.calls.Load(); got != 1 {
		t.Fatalf("expected 1 CloseIdleConnections call, got %d", got)
	}
	// Unchanged settings must be a no-op.
	if err := m.Update(Config{HTTPProxy: "http://proxy:3128"}); err != nil {
		t.Fatal(err)
	}
	if got := spy.calls.Load(); got != 1 {
		t.Fatalf("no-op update closed connections: %d calls", got)
	}
}

// startProxy returns an httptest server that acts as a plain-HTTP forward
// proxy replying with its own id, so the test can tell which proxy served
// the request.
func startProxy(t *testing.T, id string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !r.URL.IsAbs() {
			http.Error(w, "not a proxy request", http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, id)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLongLivedTransportSeesSwap(t *testing.T) {
	clearProxyEnv(t)
	proxyA := startProxy(t, "A")
	proxyB := startProxy(t, "B")

	m := NewManager()
	client := m.Client(5 * time.Second)

	fetch := func() string {
		t.Helper()
		resp, err := client.Get("http://target.invalid/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}

	if err := m.Update(Config{HTTPProxy: proxyA.URL}); err != nil {
		t.Fatal(err)
	}
	if got := fetch(); got != "A" {
		t.Fatalf("expected response from proxy A, got %q", got)
	}
	// Same client, warm connection pool: the swap must still take effect.
	if err := m.Update(Config{HTTPProxy: proxyB.URL}); err != nil {
		t.Fatal(err)
	}
	if got := fetch(); got != "B" {
		t.Fatalf("expected response from proxy B after swap, got %q", got)
	}
}

func TestHookDefaultTransport(t *testing.T) {
	clearProxyEnv(t)
	def, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Skip("http.DefaultTransport is not *http.Transport")
	}
	oldProxy := def.Proxy
	defer func() {
		def.Proxy = oldProxy
		def.CloseIdleConnections()
	}()

	proxy := startProxy(t, "hooked")
	m := NewManager()
	if err := m.HookDefaultTransport(); err != nil {
		t.Fatal(err)
	}
	if err := m.Update(Config{HTTPProxy: proxy.URL}); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get("http://target.invalid/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hooked" {
		t.Fatalf("http.Get bypassed the hooked proxy: %q", body)
	}
}

func TestWrapTransportEnablesProxyOnTLSTransport(t *testing.T) {
	clearProxyEnv(t)
	m := NewManager()
	if err := m.Update(Config{HTTPProxy: "http://proxy:3128"}); err != nil {
		t.Fatal(err)
	}
	custom := m.WrapTransport(&http.Transport{})
	proxyURL, err := custom.Proxy(mustRequest(t, "http://example.invalid/"))
	if err != nil {
		t.Fatal(err)
	}
	if proxyURL == nil || proxyURL.Host != "proxy:3128" {
		t.Fatalf("wrapped transport does not resolve proxy: %v", proxyURL)
	}
}

func TestMaskProxyURL(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{
			name: "credentials replaced",
			addr: "http://user:secret@proxy:3128",
			want: "http://***@proxy:3128",
		},
		{
			name: "no credentials untouched",
			addr: "http://proxy:3128",
			want: "http://proxy:3128",
		},
		{
			name: "empty stays empty",
			addr: "",
			want: "",
		},
		{
			name: "unparsable input still redacted",
			addr: "http://user:secret@bad host:3128",
			want: "http://***@bad host:3128",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskProxyURL(tt.addr); got != tt.want {
				t.Errorf("maskProxyURL(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		wantErr     bool
		errContains string // must appear in the error text
		errOmits    string // must NOT appear in the error text
	}{
		{
			name: "empty config",
			cfg:  Config{},
		},
		{
			name: "valid urls and no_proxy list",
			cfg: Config{
				HTTPProxy:  "http://proxy:3128",
				HTTPSProxy: "socks5://proxy:1080",
				NoProxy:    "localhost, .corp.example.com, 10.0.0.0/8, 192.168.1.5, host:8080, *, ::1",
			},
		},
		{
			name: "bare host:port accepted",
			cfg:  Config{HTTPProxy: "proxy:3128"},
		},
		{
			name: "bare host accepted",
			cfg:  Config{HTTPProxy: "proxy"},
		},
		{
			name:    "port without host rejected",
			cfg:     Config{HTTPProxy: ":3128"},
			wantErr: true,
		},
		{
			name:    "scheme without host rejected",
			cfg:     Config{HTTPProxy: "http://"},
			wantErr: true,
		},
		{
			name:        "error text redacts credentials",
			cfg:         Config{HTTPSProxy: "http://user:secret@bad host:3128"},
			wantErr:     true,
			errContains: "***",
			errOmits:    "secret",
		},
		{
			name:    "no_proxy invalid CIDR rejected",
			cfg:     Config{NoProxy: "10.0.0.0/8x"},
			wantErr: true,
		},
		{
			name:    "no_proxy invalid IP rejected",
			cfg:     Config{NoProxy: "10.0.0.256"},
			wantErr: true,
		},
		{
			name:    "no_proxy short CIDR rejected",
			cfg:     Config{NoProxy: "1.2.3/24"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				return
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Validate() error %q does not contain %q", err, tt.errContains)
			}
			if tt.errOmits != "" && strings.Contains(err.Error(), tt.errOmits) {
				t.Errorf("Validate() error %q leaks %q", err, tt.errOmits)
			}
		})
	}
}

func TestCGIGuardPreserved(t *testing.T) {
	clearProxyEnv(t)
	t.Setenv("REQUEST_METHOD", "GET")
	m := NewManager()
	if err := m.Update(Config{HTTPProxy: "http://proxy:3128"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Proxy(mustRequest(t, "http://example.invalid/")); err == nil {
		t.Fatal("expected httpoxy guard error in CGI environment")
	}
}

func TestClientReusesSharedTransport(t *testing.T) {
	clearProxyEnv(t)
	m := NewManager()
	c1, c2 := m.Client(time.Second), m.Client(2*time.Second)
	if c1.Transport != c2.Transport {
		t.Fatal("Client calls must share one transport")
	}
	m.mu.Lock()
	registered := len(m.transports)
	m.mu.Unlock()
	if registered != 1 {
		t.Fatalf("expected 1 registered transport, got %d", registered)
	}
}
