// Code generated from semantic convention specification. DO NOT EDIT.

// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

// Package healthconv provides types and functionality for OpenTelemetry semantic
// conventions in the "webitel.health" namespace.
package healthconv

import (
	"context"

	"github.com/webitel/webitel-go-kit/infra/otel/semconv/internal/metricpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// CheckGroupAttr is an attribute conforming to the webitel.health.check.group
// semantic conventions. It represents the group of the health check, which
// determines how a failure of the check affects the readiness of the node.
type CheckGroupAttr string

var (
	// CheckGroupLiveness is the check tests whether the process is operational. A
	// failure takes the node out of rotation and fails its liveness probe.
	CheckGroupLiveness CheckGroupAttr = "liveness"
	// CheckGroupCritical is the check tests a dependency local to the node. A
	// failure takes the node out of rotation.
	CheckGroupCritical CheckGroupAttr = "critical"
	// CheckGroupInformational is the check tests a shared dependency. A failure
	// marks the node degraded and leaves it in rotation.
	CheckGroupInformational CheckGroupAttr = "informational"
)

// CheckStateAttr is an attribute conforming to the webitel.health.check.state
// semantic conventions. It represents the state of the health check.
type CheckStateAttr string

var (
	// CheckStateOk is the check passes.
	CheckStateOk CheckStateAttr = "ok"
	// CheckStateFail is the check fails.
	CheckStateFail CheckStateAttr = "fail"
	// CheckStateUnknown is the check has not run yet, or its result is stale.
	CheckStateUnknown CheckStateAttr = "unknown"
)

// StateAttr is an attribute conforming to the webitel.health.state semantic
// conventions. It represents the readiness state of the node.
type StateAttr string

var (
	// StateReady is the node is in rotation and all its checks pass.
	StateReady StateAttr = "ready"
	// StateDegraded is the node is in rotation and at least one `informational`
	// check fails.
	StateDegraded StateAttr = "degraded"
	// StateNotReady is the node is out of rotation.
	StateNotReady StateAttr = "not_ready"
)

// CheckDuration is an instrument used to record metric values conforming to the
// "webitel.health.check.duration" semantic conventions. It represents the
// duration of the last completed run of a health check.
type CheckDuration struct {
	metric.Float64Gauge
}

var newCheckDurationOpts = []metric.Float64GaugeOption{
	metric.WithDescription("Duration of the last completed run of a health check."),
	metric.WithUnit("s"),
}

// NewCheckDuration returns a new CheckDuration instrument.
func NewCheckDuration(
	m metric.Meter,
	opt ...metric.Float64GaugeOption,
) (CheckDuration, error) {
	// Check if the meter is nil.
	if m == nil {
		return CheckDuration{noop.Float64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newCheckDurationOpts
	} else {
		opt = append(opt, newCheckDurationOpts...)
	}

	i, err := m.Float64Gauge(
		"webitel.health.check.duration",
		opt...,
	)
	if err != nil {
		return CheckDuration{noop.Float64Gauge{}}, err
	}
	return CheckDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CheckDuration) Inst() metric.Float64Gauge {
	return m.Float64Gauge
}

// Name returns the semantic convention name of the instrument.
func (CheckDuration) Name() string {
	return "webitel.health.check.duration"
}

// Unit returns the semantic convention unit of the instrument
func (CheckDuration) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (CheckDuration) Description() string {
	return "Duration of the last completed run of a health check."
}

// Record records val to the current distribution for attrs.
//
// The checkGroup is the the group of the health check, which determines how a
// failure of the check affects the readiness of the node.
//
// The checkName is the the name of the health check.
//
// This metric MUST NOT be reported before the first run of the check completes.
// A run in progress MUST NOT update the value.
func (m CheckDuration) Record(
	ctx context.Context,
	val float64,
	checkGroup CheckGroupAttr,
	checkName string,
	attrs ...attribute.KeyValue,
) {
	if !m.Float64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Float64Gauge.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.health.check.group", string(checkGroup)),
			attribute.String("webitel.health.check.name", checkName),
		))
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(
		*o,
		metric.WithAttributes(
			append(
				attrs[:len(attrs):len(attrs)],
				attribute.String("webitel.health.check.group", string(checkGroup)),
				attribute.String("webitel.health.check.name", checkName),
			)...,
		),
	)

	m.Float64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// This metric MUST NOT be reported before the first run of the check completes.
// A run in progress MUST NOT update the value.
func (m CheckDuration) RecordSet(ctx context.Context, val float64, set attribute.Set) {
	if !m.Float64Gauge.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Float64Gauge.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Float64Gauge.Record(ctx, val, *o...)
}

// CheckDurationObservable is an instrument used to record metric values
// conforming to the "webitel.health.check.duration" semantic conventions. It
// represents the duration of the last completed run of a health check.
type CheckDurationObservable struct {
	metric.Float64ObservableGauge
}

var newCheckDurationObservableOpts = []metric.Float64ObservableGaugeOption{
	metric.WithDescription("Duration of the last completed run of a health check."),
	metric.WithUnit("s"),
}

// NewCheckDurationObservable returns a new CheckDurationObservable instrument.
func NewCheckDurationObservable(
	m metric.Meter,
	opt ...metric.Float64ObservableGaugeOption,
) (CheckDurationObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return CheckDurationObservable{noop.Float64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newCheckDurationObservableOpts
	} else {
		opt = append(opt, newCheckDurationObservableOpts...)
	}

	i, err := m.Float64ObservableGauge(
		"webitel.health.check.duration",
		opt...,
	)
	if err != nil {
		return CheckDurationObservable{noop.Float64ObservableGauge{}}, err
	}
	return CheckDurationObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CheckDurationObservable) Inst() metric.Float64ObservableGauge {
	return m.Float64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (CheckDurationObservable) Name() string {
	return "webitel.health.check.duration"
}

// Unit returns the semantic convention unit of the instrument
func (CheckDurationObservable) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (CheckDurationObservable) Description() string {
	return "Duration of the last completed run of a health check."
}

// AttrCheckGroup returns a required attribute for the
// "webitel.health.check.group" semantic convention. It represents the group of
// the health check, which determines how a failure of the check affects the
// readiness of the node.
func (CheckDurationObservable) AttrCheckGroup(val CheckGroupAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.group", string(val))
}

// AttrCheckName returns a required attribute for the "webitel.health.check.name"
// semantic convention. It represents the name of the health check.
func (CheckDurationObservable) AttrCheckName(val string) attribute.KeyValue {
	return attribute.String("webitel.health.check.name", val)
}

// CheckStatus is an instrument used to record metric values conforming to the
// "webitel.health.check.status" semantic conventions. It represents the current
// status of the health check.
type CheckStatus struct {
	metric.Int64UpDownCounter
}

var newCheckStatusOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("The current status of the health check."),
	metric.WithUnit("1"),
}

// NewCheckStatus returns a new CheckStatus instrument.
func NewCheckStatus(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (CheckStatus, error) {
	// Check if the meter is nil.
	if m == nil {
		return CheckStatus{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newCheckStatusOpts
	} else {
		opt = append(opt, newCheckStatusOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.health.check.status",
		opt...,
	)
	if err != nil {
		return CheckStatus{noop.Int64UpDownCounter{}}, err
	}
	return CheckStatus{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CheckStatus) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (CheckStatus) Name() string {
	return "webitel.health.check.status"
}

// Unit returns the semantic convention unit of the instrument
func (CheckStatus) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (CheckStatus) Description() string {
	return "The current status of the health check."
}

// Add adds incr to the existing count for attrs.
//
// The checkGroup is the the group of the health check, which determines how a
// failure of the check affects the readiness of the node.
//
// The checkName is the the name of the health check.
//
// The checkState is the the state of the health check.
//
// A timeseries is produced for every possible value of
// `webitel.health.check.state`. The value of this metric is 1 for the current
// state of the check, and 0 for the other states.
func (m CheckStatus) Add(
	ctx context.Context,
	incr int64,
	checkGroup CheckGroupAttr,
	checkName string,
	checkState CheckStateAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.health.check.group", string(checkGroup)),
			attribute.String("webitel.health.check.name", checkName),
			attribute.String("webitel.health.check.state", string(checkState)),
		))
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(
		*o,
		metric.WithAttributes(
			append(
				attrs[:len(attrs):len(attrs)],
				attribute.String("webitel.health.check.group", string(checkGroup)),
				attribute.String("webitel.health.check.name", checkName),
				attribute.String("webitel.health.check.state", string(checkState)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// A timeseries is produced for every possible value of
// `webitel.health.check.state`. The value of this metric is 1 for the current
// state of the check, and 0 for the other states.
func (m CheckStatus) AddSet(ctx context.Context, incr int64, set attribute.Set) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Int64UpDownCounter.Add(ctx, incr)
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// CheckStatusObservable is an instrument used to record metric values conforming
// to the "webitel.health.check.status" semantic conventions. It represents the
// current status of the health check.
type CheckStatusObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newCheckStatusObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("The current status of the health check."),
	metric.WithUnit("1"),
}

// NewCheckStatusObservable returns a new CheckStatusObservable instrument.
func NewCheckStatusObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (CheckStatusObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return CheckStatusObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newCheckStatusObservableOpts
	} else {
		opt = append(opt, newCheckStatusObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.health.check.status",
		opt...,
	)
	if err != nil {
		return CheckStatusObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return CheckStatusObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CheckStatusObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (CheckStatusObservable) Name() string {
	return "webitel.health.check.status"
}

// Unit returns the semantic convention unit of the instrument
func (CheckStatusObservable) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (CheckStatusObservable) Description() string {
	return "The current status of the health check."
}

// AttrCheckGroup returns a required attribute for the
// "webitel.health.check.group" semantic convention. It represents the group of
// the health check, which determines how a failure of the check affects the
// readiness of the node.
func (CheckStatusObservable) AttrCheckGroup(val CheckGroupAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.group", string(val))
}

// AttrCheckName returns a required attribute for the "webitel.health.check.name"
// semantic convention. It represents the name of the health check.
func (CheckStatusObservable) AttrCheckName(val string) attribute.KeyValue {
	return attribute.String("webitel.health.check.name", val)
}

// AttrCheckState returns a required attribute for the
// "webitel.health.check.state" semantic convention. It represents the state of
// the health check.
func (CheckStatusObservable) AttrCheckState(val CheckStateAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.state", string(val))
}

// CheckTransitions is an instrument used to record metric values conforming to
// the "webitel.health.check.transitions" semantic conventions. It represents the
// number of transitions of a health check into the state described by the
// `webitel.health.check.state` attribute.
type CheckTransitions struct {
	metric.Int64Counter
}

var newCheckTransitionsOpts = []metric.Int64CounterOption{
	metric.WithDescription("Number of transitions of a health check into the state described by the `webitel.health.check.state` attribute."),
	metric.WithUnit("{transition}"),
}

// NewCheckTransitions returns a new CheckTransitions instrument.
func NewCheckTransitions(
	m metric.Meter,
	opt ...metric.Int64CounterOption,
) (CheckTransitions, error) {
	// Check if the meter is nil.
	if m == nil {
		return CheckTransitions{noop.Int64Counter{}}, nil
	}

	if len(opt) == 0 {
		opt = newCheckTransitionsOpts
	} else {
		opt = append(opt, newCheckTransitionsOpts...)
	}

	i, err := m.Int64Counter(
		"webitel.health.check.transitions",
		opt...,
	)
	if err != nil {
		return CheckTransitions{noop.Int64Counter{}}, err
	}
	return CheckTransitions{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CheckTransitions) Inst() metric.Int64Counter {
	return m.Int64Counter
}

// Name returns the semantic convention name of the instrument.
func (CheckTransitions) Name() string {
	return "webitel.health.check.transitions"
}

// Unit returns the semantic convention unit of the instrument
func (CheckTransitions) Unit() string {
	return "{transition}"
}

// Description returns the semantic convention description of the instrument
func (CheckTransitions) Description() string {
	return "Number of transitions of a health check into the state described by the `webitel.health.check.state` attribute."
}

// Add adds incr to the existing count for attrs.
//
// The checkGroup is the the group of the health check, which determines how a
// failure of the check affects the readiness of the node.
//
// The checkName is the the name of the health check.
//
// The checkState is the the state of the health check.
//
// The first completed run of a check MUST be counted as a transition. A result
// becoming stale MUST NOT be counted as a transition.
func (m CheckTransitions) Add(
	ctx context.Context,
	incr int64,
	checkGroup CheckGroupAttr,
	checkName string,
	checkState CheckStateAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Counter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Counter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.health.check.group", string(checkGroup)),
			attribute.String("webitel.health.check.name", checkName),
			attribute.String("webitel.health.check.state", string(checkState)),
		))
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(
		*o,
		metric.WithAttributes(
			append(
				attrs[:len(attrs):len(attrs)],
				attribute.String("webitel.health.check.group", string(checkGroup)),
				attribute.String("webitel.health.check.name", checkName),
				attribute.String("webitel.health.check.state", string(checkState)),
			)...,
		),
	)

	m.Int64Counter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// The first completed run of a check MUST be counted as a transition. A result
// becoming stale MUST NOT be counted as a transition.
func (m CheckTransitions) AddSet(ctx context.Context, incr int64, set attribute.Set) {
	if !m.Int64Counter.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Int64Counter.Add(ctx, incr)
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Int64Counter.Add(ctx, incr, *o...)
}

// CheckTransitionsObservable is an instrument used to record metric values
// conforming to the "webitel.health.check.transitions" semantic conventions. It
// represents the number of transitions of a health check into the state
// described by the `webitel.health.check.state` attribute.
type CheckTransitionsObservable struct {
	metric.Int64ObservableCounter
}

var newCheckTransitionsObservableOpts = []metric.Int64ObservableCounterOption{
	metric.WithDescription("Number of transitions of a health check into the state described by the `webitel.health.check.state` attribute."),
	metric.WithUnit("{transition}"),
}

// NewCheckTransitionsObservable returns a new CheckTransitionsObservable
// instrument.
func NewCheckTransitionsObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableCounterOption,
) (CheckTransitionsObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return CheckTransitionsObservable{noop.Int64ObservableCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newCheckTransitionsObservableOpts
	} else {
		opt = append(opt, newCheckTransitionsObservableOpts...)
	}

	i, err := m.Int64ObservableCounter(
		"webitel.health.check.transitions",
		opt...,
	)
	if err != nil {
		return CheckTransitionsObservable{noop.Int64ObservableCounter{}}, err
	}
	return CheckTransitionsObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CheckTransitionsObservable) Inst() metric.Int64ObservableCounter {
	return m.Int64ObservableCounter
}

// Name returns the semantic convention name of the instrument.
func (CheckTransitionsObservable) Name() string {
	return "webitel.health.check.transitions"
}

// Unit returns the semantic convention unit of the instrument
func (CheckTransitionsObservable) Unit() string {
	return "{transition}"
}

// Description returns the semantic convention description of the instrument
func (CheckTransitionsObservable) Description() string {
	return "Number of transitions of a health check into the state described by the `webitel.health.check.state` attribute."
}

// AttrCheckGroup returns a required attribute for the
// "webitel.health.check.group" semantic convention. It represents the group of
// the health check, which determines how a failure of the check affects the
// readiness of the node.
func (CheckTransitionsObservable) AttrCheckGroup(val CheckGroupAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.group", string(val))
}

// AttrCheckName returns a required attribute for the "webitel.health.check.name"
// semantic convention. It represents the name of the health check.
func (CheckTransitionsObservable) AttrCheckName(val string) attribute.KeyValue {
	return attribute.String("webitel.health.check.name", val)
}

// AttrCheckState returns a required attribute for the
// "webitel.health.check.state" semantic convention. It represents the state of
// the health check.
func (CheckTransitionsObservable) AttrCheckState(val CheckStateAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.state", string(val))
}

// Status is an instrument used to record metric values conforming to the
// "webitel.health.status" semantic conventions. It represents the current
// readiness status of the node.
type Status struct {
	metric.Int64UpDownCounter
}

var newStatusOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("The current readiness status of the node."),
	metric.WithUnit("1"),
}

// NewStatus returns a new Status instrument.
func NewStatus(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (Status, error) {
	// Check if the meter is nil.
	if m == nil {
		return Status{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newStatusOpts
	} else {
		opt = append(opt, newStatusOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.health.status",
		opt...,
	)
	if err != nil {
		return Status{noop.Int64UpDownCounter{}}, err
	}
	return Status{i}, nil
}

// Inst returns the underlying metric instrument.
func (m Status) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (Status) Name() string {
	return "webitel.health.status"
}

// Unit returns the semantic convention unit of the instrument
func (Status) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (Status) Description() string {
	return "The current readiness status of the node."
}

// Add adds incr to the existing count for attrs.
//
// The state is the the readiness state of the node.
//
// A timeseries is produced for every possible value of `webitel.health.state`.
// The value of this metric is 1 for the current state of the node, and 0 for the
// other states.
func (m Status) Add(
	ctx context.Context,
	incr int64,
	state StateAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.health.state", string(state)),
		))
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(
		*o,
		metric.WithAttributes(
			append(
				attrs[:len(attrs):len(attrs)],
				attribute.String("webitel.health.state", string(state)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// A timeseries is produced for every possible value of `webitel.health.state`.
// The value of this metric is 1 for the current state of the node, and 0 for the
// other states.
func (m Status) AddSet(ctx context.Context, incr int64, set attribute.Set) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Int64UpDownCounter.Add(ctx, incr)
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// StatusObservable is an instrument used to record metric values conforming to
// the "webitel.health.status" semantic conventions. It represents the current
// readiness status of the node.
type StatusObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newStatusObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("The current readiness status of the node."),
	metric.WithUnit("1"),
}

// NewStatusObservable returns a new StatusObservable instrument.
func NewStatusObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (StatusObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return StatusObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newStatusObservableOpts
	} else {
		opt = append(opt, newStatusObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.health.status",
		opt...,
	)
	if err != nil {
		return StatusObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return StatusObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m StatusObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (StatusObservable) Name() string {
	return "webitel.health.status"
}

// Unit returns the semantic convention unit of the instrument
func (StatusObservable) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (StatusObservable) Description() string {
	return "The current readiness status of the node."
}

// AttrState returns a required attribute for the "webitel.health.state" semantic
// convention. It represents the readiness state of the node.
func (StatusObservable) AttrState(val StateAttr) attribute.KeyValue {
	return attribute.String("webitel.health.state", string(val))
}
