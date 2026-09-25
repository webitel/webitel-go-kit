package health

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/webitel/webitel-go-kit/infra/health"
	"github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.2.0/healthconv"
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

	status, err := healthconv.NewStatusObservable(meter)
	if err != nil {
		return nil, err
	}
	
	checkStatus, err := healthconv.NewCheckStatusObservable(meter)
	if err != nil {
		return nil, err
	}

	transitions, err := healthconv.NewCheckTransitionsObservable(meter)
	if err != nil {
		return nil, err
	}

	duration, err := healthconv.NewCheckDurationObservable(meter)
	if err != nil {
		return nil, err
	}

	return meter.RegisterCallback(
		collect(snap, status, checkStatus, transitions, duration),
		status.Inst(), checkStatus.Inst(), transitions.Inst(), duration.Inst(),
	)
}

var (
	nodeStates  = []healthconv.StateAttr{healthconv.StateReady, healthconv.StateDegraded, healthconv.StateNotReady}
	checkStates = []healthconv.CheckStateAttr{healthconv.CheckStateOk, healthconv.CheckStateFail, healthconv.CheckStateUnknown}
)

func nodeState(s health.State) healthconv.StateAttr {
	switch s {
	case health.StateReady:
		return healthconv.StateReady
	case health.StateDegraded:
		return healthconv.StateDegraded
	default:
		return healthconv.StateNotReady
	}
}

func checkState(s health.Status) healthconv.CheckStateAttr {
	switch s {
	case health.StatusOK:
		return healthconv.CheckStateOk
	case health.StatusFail:
		return healthconv.CheckStateFail
	default:
		return healthconv.CheckStateUnknown
	}
}

func boolValue(b bool) int64 {
	if b {
		return 1
	}

	return 0
}

func collect(
	snap Snapshotter,
	status healthconv.StatusObservable,
	checkStatus healthconv.CheckStatusObservable,
	transitions healthconv.CheckTransitionsObservable,
	duration healthconv.CheckDurationObservable,
) metric.Callback {
	return func(_ context.Context, o metric.Observer) error {
		snapshot := snap.Snapshot()

		current := nodeState(snapshot.State)
		for _, s := range nodeStates {
			o.ObserveInt64(status.Inst(), boolValue(s == current), metric.WithAttributes(status.AttrState(s)))
		}

		for _, c := range snapshot.Checks {
			group := healthconv.CheckGroupAttr(c.Group.String())

			cur := checkState(c.Status)
			for _, s := range checkStates {
				o.ObserveInt64(checkStatus.Inst(), boolValue(s == cur), metric.WithAttributes(
					checkStatus.AttrCheckName(c.Name),
					checkStatus.AttrCheckGroup(group),
					checkStatus.AttrCheckState(s),
				))
			}

			o.ObserveInt64(transitions.Inst(), int64(c.Transitions.OK), metric.WithAttributes(
				transitions.AttrCheckName(c.Name),
				transitions.AttrCheckGroup(group),
				transitions.AttrCheckState(healthconv.CheckStateOk),
			))

			o.ObserveInt64(transitions.Inst(), int64(c.Transitions.Fail), metric.WithAttributes(
				transitions.AttrCheckName(c.Name),
				transitions.AttrCheckGroup(group),
				transitions.AttrCheckState(healthconv.CheckStateFail),
			))

			if !c.LastRun.IsZero() {
				o.ObserveFloat64(duration.Inst(), c.LastDuration.Seconds(), metric.WithAttributes(
					duration.AttrCheckName(c.Name),
					duration.AttrCheckGroup(group),
				))
			}
		}

		return nil
	}
}
