package healthhttp

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// fakeAddr lets a test present any address while the socket stays on loopback.
type fakeAddr string

func (a fakeAddr) Network() string { return "tcp" }
func (a fakeAddr) String() string  { return string(a) }

// lyingListener is a real loopback listener that reports someone else's
// address, so the non-loopback path is exercised without binding a public port.
type lyingListener struct {
	net.Listener
	addr net.Addr
}

func (l lyingListener) Addr() net.Addr { return l.addr }

func stopServerAtCleanup(t *testing.T, srv *Server) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if err := srv.Stop(ctx); err != nil {
			t.Errorf("Stop: %v", err)
		}
	})
}

func startServer(t *testing.T, reg *health.Registry) *Server {
	t.Helper()

	srv := NewServer(reg, "127.0.0.1:0")
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	stopServerAtCleanup(t, srv)

	return srv
}

// fetch does a real HTTP GET and returns the status, body and headers.
func fetch(t *testing.T, url string) (int, string, http.Header) {
	t.Helper()

	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading %s: %v", url, err)
	}

	return res.StatusCode, string(body), res.Header
}

func TestNilServer(t *testing.T) {
	srv := NewServer(okRegistry(t), "")
	if srv != nil {
		t.Fatal("NewServer with an empty address is not nil")
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start on a nil Server: %v", err)
	}
	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop on a nil Server: %v", err)
	}
	if got := srv.Addr(); got != "" {
		t.Fatalf("Addr on a nil Server = %q, want empty", got)
	}
}

func TestServerServesProbes(t *testing.T) {
	reg := okRegistry(t)
	srv := startServer(t, reg)

	if _, _, err := net.SplitHostPort(srv.Addr()); err != nil {
		t.Fatalf("Addr = %q, want a bound host:port: %v", srv.Addr(), err)
	}

	code, body, _ := fetch(t, "http://"+srv.Addr()+"/readyz")
	want := get(Handler(reg), "/readyz")
	if code != want.Code || body != want.Body.String() {
		t.Fatalf("server answered %d %q, mounted handler answered %d %q",
			code, body, want.Code, want.Body.String())
	}
}

func TestStartBindError(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer held.Close()

	srv := NewServer(okRegistry(t), held.Addr().String())

	err = srv.Start()
	if err == nil {
		t.Fatal("Start on a busy port returned nil")
	}
	if !strings.Contains(err.Error(), held.Addr().String()) {
		t.Fatalf("error %q does not mention the address %q", err, held.Addr())
	}
}

func TestStopIdempotent(t *testing.T) {
	srv := NewServer(okRegistry(t), "127.0.0.1:0")

	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop before Start: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
}

func TestVerboseOnLoopback(t *testing.T) {
	srv := startServer(t, okRegistry(t))

	code, body, header := fetch(t, "http://"+srv.Addr()+"/readyz?verbose")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if got := header.Get("Content-Type"); got != jsonCT {
		t.Fatalf("Content-Type = %q, want %q", got, jsonCT)
	}
	if !strings.HasPrefix(body, `{"status":"`+health.NameReady+`"`) {
		t.Fatalf("body = %q, want the D1 JSON", body)
	}
}

func TestVerboseRefusedOffLoopback(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	off := fakeAddr("192.168.1.10:8080")
	srv := NewServer(failingRegistry(t), off.String())
	// serve() bypasses Start(), so the Serve goroutine is ours to stop —
	// otherwise -count=3 accumulates three of them.
	stopServerAtCleanup(t, srv)
	if err := srv.serve(lyingListener{Listener: l, addr: off}); err != nil {
		t.Fatalf("serve: %v", err)
	}

	_, body, header := fetch(t, "http://"+l.Addr().String()+"/readyz?verbose")
	if got := header.Get("Content-Type"); got != textCT {
		t.Fatalf("Content-Type = %q, want %q", got, textCT)
	}
	if body != health.NameNotReady+"\n" {
		t.Fatalf("body = %q, want the one-word body", body)
	}
	assertNoSentinel(t, "off-loopback /readyz?verbose", body, header)
}

func TestLoopbackOnly(t *testing.T) {
	bound, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer bound.Close()

	for _, tt := range []struct {
		addr net.Addr
		want bool
	}{
		{fakeAddr(":8080"), false},
		{fakeAddr("0.0.0.0:8080"), false},
		{fakeAddr("[::]:8080"), false},
		{fakeAddr("127.0.0.1:8080"), true},
		{fakeAddr("127.5.5.5:8080"), true}, // all of 127/8
		{fakeAddr("[::1]:8080"), true},
		{fakeAddr("[::ffff:127.0.0.1]:8080"), true},
		{fakeAddr("localhost:8080"), false}, // a listener never reports a name
		{fakeAddr("192.168.1.10:8080"), false},
		{fakeAddr("example.com:80"), false}, // no DNS lookup is attempted, deliberately
		{fakeAddr("localhost"), false},      // no port: SplitHostPort errors
		{bound.Addr(), true},
	} {
		t.Run(tt.addr.String(), func(t *testing.T) {
			if got := loopbackOnly(tt.addr); got != tt.want {
				t.Fatalf("loopbackOnly(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}
