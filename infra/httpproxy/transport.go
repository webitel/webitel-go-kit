package httpproxy

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// Transport returns a clone of http.DefaultTransport with Proxy bound to the
// Manager and registered for idle-connection invalidation. Create it once
// and share it: every call permanently registers a new clone with its own
// connection pool, so calling it per request leaks registrations and defeats
// keep-alive reuse — for a ready-made shared client use Client instead.
func (m *Manager) Transport() *http.Transport {
	var t *http.Transport
	if def, ok := http.DefaultTransport.(*http.Transport); ok {
		t = def.Clone()
	} else {
		// Keep in sync with net/http.DefaultTransport defaults.
		t = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}
	t.Proxy = m.Proxy
	m.Register(t)
	return t
}

// WrapTransport binds a caller-built long-lived transport to the Manager:
// sets its Proxy and registers it for idle-connection invalidation.
// A nil argument is equivalent to Transport(). Like Transport, it registers
// the transport for the Manager's lifetime — do not call it per request;
// for short-lived transports set Proxy to Manager.Proxy directly.
func (m *Manager) WrapTransport(t *http.Transport) *http.Transport {
	if t == nil {
		return m.Transport()
	}
	t.Proxy = m.Proxy
	m.Register(t)
	return t
}

// Client returns an *http.Client backed by the Manager's single shared
// transport, so it is cheap and safe to call anywhere, including per
// request: all returned clients reuse one connection pool. A zero timeout
// means no client-level timeout.
func (m *Manager) Client(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: m.sharedTransport(),
		Timeout:   timeout,
	}
}

func (m *Manager) sharedTransport() *http.Transport {
	m.sharedOnce.Do(func() {
		m.shared = m.Transport()
	})
	return m.shared
}

// HookDefaultTransport binds http.DefaultTransport to the Manager, making
// every default-transport call site (http.Get, http.DefaultClient,
// zero-value http.Client, SDKs building on the default transport) honor the
// current settings. Call it once, as early as possible in main(), before any
// outbound request is made and before any code clones the default transport:
// mutating the transport is not synchronized with in-flight requests, and
// clones taken earlier keep the previous Proxy function.
func (m *Manager) HookDefaultTransport() error {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return fmt.Errorf("httpproxy: http.DefaultTransport is %T, not *http.Transport", http.DefaultTransport)
	}
	t.Proxy = m.Proxy
	m.Register(t)
	return nil
}
