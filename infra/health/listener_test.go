package health

import (
	"context"
	"net"
	"path/filepath"
	"strings"
	"testing"
)

func acceptLoop(ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}

		c.Close()
	}
}

func TestListenerCheckAcceptsAndFails(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go acceptLoop(ln)

	check := ListenerCheck(ln)
	if err := check(context.Background()); err != nil {
		t.Fatalf("live listener: %v", err)
	}

	// A check that cannot fail proves nothing.
	ln.Close()

	if err := check(context.Background()); err == nil {
		t.Error("closed listener reported healthy")
	}
}

// A wildcard bind is not dialable as written. Substituting loopback is what
// keeps the check about the listener rather than about the network.
func TestListenerCheckWildcardBindUsesLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	defer ln.Close()

	go acceptLoop(ln)

	host, _, _ := net.SplitHostPort(ln.Addr().String())
	if ip := net.ParseIP(host); host != "" && ip != nil && !ip.IsUnspecified() {
		t.Skipf("listener did not bind wildcard (got %q)", host)
	}

	if err := ListenerCheck(ln)(context.Background()); err != nil {
		t.Errorf("wildcard bind should dial loopback: %v", err)
	}
}

func TestListenerCheckUnixSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.sock")

	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Skipf("unix sockets unavailable: %v", err)
	}

	defer ln.Close()

	go acceptLoop(ln)

	// A unix address is a path, so it must skip the host:port rewrite.
	if err := ListenerCheck(ln)(context.Background()); err != nil {
		t.Errorf("unix listener: %v", err)
	}
}

func TestListenerCheckNilListener(t *testing.T) {
	err := ListenerCheck(nil)(context.Background())
	if err == nil {
		t.Fatal("nil listener reported healthy")
	}

	if !strings.Contains(err.Error(), "nil listener") {
		t.Errorf("unhelpful error: %v", err)
	}
}

func TestListenerCheckHonoursContext(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	defer ln.Close()

	go acceptLoop(ln)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := ListenerCheck(ln)(ctx); err == nil {
		t.Error("a cancelled context should abort the dial")
	}
}

// A tcp6 listener is IPv6-only: rewriting its wildcard to 127.0.0.1 would
// report a healthy listener as dead.
func TestListenerCheckIPv6WildcardUsesIPv6Loopback(t *testing.T) {
	ln, err := net.Listen("tcp6", "[::]:0")
	if err != nil {
		t.Skipf("no IPv6 on this host: %v", err)
	}

	defer ln.Close()

	go acceptLoop(ln)

	if err := ListenerCheck(ln)(context.Background()); err != nil {
		t.Errorf("tcp6 wildcard should dial [::1]: %v", err)
	}
}
