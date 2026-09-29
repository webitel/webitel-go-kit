package health

import (
	"context"
	"reflect"
)

// Transport exposes the registry's verdict, such as the sd_notify notifier or
// an HTTP probe server. The registry starts it after the checks and stops it
// before the DrainHold wait. Stop on a transport that never started must be
// safe: it must not block or fail, though it may still send a shutdown signal.
type Transport interface {
	Start(ctx context.Context, r *Registry) error
	Stop(ctx context.Context) error
}

// Option configures a Registry.
type Option func(*Registry)

// WithTransport adds a transport to start and stop with the registry. Nil and
// typed-nil transports are ignored.
func WithTransport(t Transport) Option {
	return func(r *Registry) {
		if isNil(t) {
			return
		}

		r.transports = append(r.transports, t)
	}
}

// isNil reports whether t holds nothing, including a typed nil. A nil pointer
// inside an interface is not == nil, and callers routinely pass one:
// sdnotify.New returns a nil *Notifier when NOTIFY_SOCKET is unset. That one is
// nil-safe, but a Transport from elsewhere need not be.
func isNil(t Transport) bool {
	if t == nil {
		return true
	}

	switch v := reflect.ValueOf(t); v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
