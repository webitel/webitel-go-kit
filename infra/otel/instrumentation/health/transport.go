package health

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel/metric"

	"github.com/webitel/webitel-go-kit/infra/health"
)

var _ health.Transport = (*Transport)(nil)

// Transport binds the metrics to a registry's lifetime: they are registered
// when the registry starts and unregistered when it stops. A nil *Transport is
// a no-op.
type Transport struct {
	opts []Option

	// mu guards reg against a Stop racing the registry's Start.
	mu  sync.Mutex
	reg metric.Registration
}

// NewTransport builds a Transport for health.WithTransport.
func NewTransport(opts ...Option) *Transport {
	return &Transport{opts: opts}
}

// Start registers the metrics against r. Calling it twice returns an error.
func (t *Transport) Start(_ context.Context, r *health.Registry) error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.reg != nil {
		return errors.New("otel/instrumentation/health: transport is already started")
	}

	reg, err := register(r, t.opts...)
	if err != nil {
		return err
	}

	t.reg = reg

	return nil
}

// Stop unregisters the metrics. Never started is a no-op.
func (t *Transport) Stop(context.Context) error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.reg == nil {
		return nil
	}

	err := t.reg.Unregister()
	t.reg = nil

	return err
}
