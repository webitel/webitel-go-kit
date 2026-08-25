// Package health exports infra/health's Registry as OpenTelemetry metrics:
// webitel.health.ready, webitel.health.check.state,
// webitel.health.check.transitions and webitel.health.check.duration. One
// callback feeds all four from a single Snapshot() per collection, so a
// scrape runs no checks.
//
// Call Start once per (registry, provider) pair.
package health

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/webitel/webitel-go-kit/infra/health"
	"github.com/webitel/webitel-go-kit/infra/otel/semconv/webitelconv"
)

const scopeName = "github.com/webitel/webitel-go-kit/infra/otel/instrumentation/health"

// Version is the current release version of this instrumentation.
func Version() string {
	return "0.0.0"
}

// Snapshotter is what the binding reads; *health.Registry satisfies it.
type Snapshotter interface {
	Snapshot() health.Snapshot
}

// Start registers the metrics against snap and returns the registration as
// the teardown handle.
func Start(snap Snapshotter, opts ...Option) (metric.Registration, error) {
	if snap == nil {
		return nil, errors.New("otel/instrumentation/health: nil Snapshotter")
	}

	cfg := &bindingConfig{mp: otel.GetMeterProvider()}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	meter := cfg.mp.Meter(scopeName, metric.WithInstrumentationVersion(Version()))

	ready, err := webitelconv.NewHealthReady(meter)
	if err != nil {
		return nil, err
	}
	state, err := webitelconv.NewHealthCheckState(meter)
	if err != nil {
		return nil, err
	}
	transitions, err := webitelconv.NewHealthCheckTransitions(meter)
	if err != nil {
		return nil, err
	}
	duration, err := webitelconv.NewHealthCheckDuration(meter)
	if err != nil {
		return nil, err
	}

	return meter.RegisterCallback(
		collect(snap, ready, state, transitions, duration),
		ready.Inst(), state.Inst(), transitions.Inst(), duration.Inst(),
	)
}

func collect(
	snap Snapshotter,
	ready webitelconv.HealthReady,
	state webitelconv.HealthCheckState,
	transitions webitelconv.HealthCheckTransitions,
	duration webitelconv.HealthCheckDuration,
) metric.Callback {
	return func(_ context.Context, o metric.Observer) error {
		snapshot := snap.Snapshot()

		readyVal := int64(0)
		if snapshot.State.Ready() {
			readyVal = 1
		}
		o.ObserveInt64(ready.Inst(), readyVal)

		for _, c := range snapshot.Checks {
			group := webitelconv.HealthCheckGroupAttr(c.Group.String())
			checkAttrs := metric.WithAttributes(
				state.AttrHealthCheckName(c.Name),
				state.AttrHealthCheckGroup(group),
			)

			stateVal := int64(0)
			if c.Status == health.StatusOK {
				stateVal = 1
			}
			o.ObserveInt64(state.Inst(), stateVal, checkAttrs)

			// Zero counts are observed too, so the first flip shows as an increase().
			o.ObserveInt64(transitions.Inst(), int64(c.Transitions.OK), metric.WithAttributes(
				transitions.AttrHealthCheckName(c.Name),
				transitions.AttrHealthCheckGroup(group),
				transitions.AttrHealthCheckStatus(webitelconv.HealthCheckStatusOk),
			))
			o.ObserveInt64(transitions.Inst(), int64(c.Transitions.Fail), metric.WithAttributes(
				transitions.AttrHealthCheckName(c.Name),
				transitions.AttrHealthCheckGroup(group),
				transitions.AttrHealthCheckStatus(webitelconv.HealthCheckStatusFail),
			))

			if !c.LastRun.IsZero() {
				o.ObserveFloat64(duration.Inst(), c.LastDuration.Seconds(), checkAttrs)
			}
		}

		return nil
	}
}
