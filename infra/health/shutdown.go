package health

import (
	"context"
	"errors"
	"reflect"
)

// Stopper is a transport that has to be stopped between Drain and Stop, such
// as the sd_notify notifier or an HTTP probe server.
type Stopper interface {
	Stop(context.Context) error
}

// isNil reports whether t holds nothing, including a typed nil. A nil pointer
// inside an interface is not == nil, and callers routinely pass one:
// sdnotify.New returns a nil *Notifier when NOTIFY_SOCKET is unset. That one is
// nil-safe, but a Stopper from elsewhere need not be.
func isNil(t Stopper) bool {
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

// Shutdown runs the drain sequence in the order the registry requires: Drain
// first so the verdict flips before anything is torn down, then each transport,
// then the registry itself.
//
// ctx must carry at least DrainHold of budget: Drain returns immediately and
// the hold is absorbed inside Stop. A shorter context cuts the hold short and
// service discovery may never observe the node leaving rotation.
//
// Every transport is stopped even if an earlier one fails; the errors are
// joined.
func Shutdown(ctx context.Context, r *Registry, transports ...Stopper) error {
	r.Drain()

	var errs []error

	for _, t := range transports {
		if isNil(t) {
			continue
		}

		if err := t.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if err := r.Stop(ctx); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
