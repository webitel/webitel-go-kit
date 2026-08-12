// Package sdnotify reports the health registry to systemd over NOTIFY_SOCKET.
// Without that variable New returns nil and every method is a no-op.
package sdnotify
