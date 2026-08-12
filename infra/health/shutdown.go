package health

import (
	"context"
	"errors"
)

// Stopper is a transport that has to be stopped between Drain and Stop, such
// as the sd_notify notifier or an HTTP probe server.
type Stopper interface {
	Stop(context.Context) error
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
		if t == nil {
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
