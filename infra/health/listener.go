package health

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

// ListenerCheck reports whether ln still accepts connections. It proves the
// socket is live and says nothing about whether a request would succeed.
//
// It dials the listener's own address, not the address the service advertises
// to discovery. Those differ whenever a wildcard bind is resolved to a public
// address, and a host frequently cannot reach itself that way — the dial would
// then test the network rather than the listener. A wildcard ("[::]",
// "0.0.0.0") is not dialable as written, so loopback is substituted; it
// reaches the same socket.
func ListenerCheck(ln net.Listener) Check {
	return func(ctx context.Context) error {
		if ln == nil {
			return errors.New("health: nil listener")
		}

		addr := ln.Addr()
		if addr == nil {
			return errors.New("health: listener has no address")
		}

		network, target := addr.Network(), addr.String()

		// Only host:port networks need the wildcard rewrite; a unix socket
		// address is a path and is dialable as-is.
		if strings.HasPrefix(network, "tcp") {
			host, port, err := net.SplitHostPort(target)
			if err != nil {
				return fmt.Errorf("health: listener address %q: %w", target, err)
			}

			if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
				host = "127.0.0.1"
			}

			target = net.JoinHostPort(host, port)
		}

		var d net.Dialer

		conn, err := d.DialContext(ctx, network, target)
		if err != nil {
			return fmt.Errorf("health: listener not accepting on %s: %w", target, err)
		}

		return conn.Close()
	}
}
