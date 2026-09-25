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
// semantic conventions. It represents the group the check belongs to, which
// decides how a failure affects readiness.
type CheckGroupAttr string

var (
	// CheckGroupLiveness is the liveness probe; a failure also fails readiness.
	CheckGroupLiveness CheckGroupAttr = "liveness"
	// CheckGroupCritical is the node-local dependency; a failure takes the node out
	// of rotation.
	CheckGroupCritical CheckGroupAttr = "critical"
	// CheckGroupInformational is the dependency whose failure only degrades; the
	// node stays in rotation.
	CheckGroupInformational CheckGroupAttr = "informational"
)

// CheckStatusAttr is an attribute conforming to the webitel.health.check.status
// semantic conventions. It represents the status a check transitioned into.
type CheckStatusAttr string

var (
	// CheckStatusOk is the passing check.
	CheckStatusOk CheckStatusAttr = "ok"
	// CheckStatusFail is the check that failed past its threshold.
	CheckStatusFail CheckStatusAttr = "fail"
)

// CheckDuration is an instrument used to record metric values conforming to the
// "webitel.health.check.duration" semantic conventions. It represents the
// elapsed time of a check's last completed run; absent until the first run
// lands.
type CheckDuration struct {
	metric.Float64Gauge
}

var newCheckDurationOpts = []metric.Float64GaugeOption{
	metric.WithDescription("Elapsed time of a check's last completed run; absent until the first run lands."),
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
	return "Elapsed time of a check's last completed run; absent until the first run lands."
}

// Record records val to the current distribution for attrs.
//
// The checkGroup is the the group the check belongs to, which decides how a
// failure affects readiness.
//
// The checkName is the the name the check was registered under.
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
// represents the elapsed time of a check's last completed run; absent until the
// first run lands.
type CheckDurationObservable struct {
	metric.Float64ObservableGauge
}

var newCheckDurationObservableOpts = []metric.Float64ObservableGaugeOption{
	metric.WithDescription("Elapsed time of a check's last completed run; absent until the first run lands."),
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
	return "Elapsed time of a check's last completed run; absent until the first run lands."
}

// AttrCheckGroup returns a required attribute for the
// "webitel.health.check.group" semantic convention. It represents the group the
// check belongs to, which decides how a failure affects readiness.
func (CheckDurationObservable) AttrCheckGroup(val CheckGroupAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.group", string(val))
}

// AttrCheckName returns a required attribute for the "webitel.health.check.name"
// semantic convention. It represents the name the check was registered under.
func (CheckDurationObservable) AttrCheckName(val string) attribute.KeyValue {
	return attribute.String("webitel.health.check.name", val)
}

// CheckState is an instrument used to record metric values conforming to the
// "webitel.health.check.state" semantic conventions. It represents the 1 when a
// check's current status is ok, 0 otherwise; a stale result is not healthy.
type CheckState struct {
	metric.Int64Gauge
}

var newCheckStateOpts = []metric.Int64GaugeOption{
	metric.WithDescription("1 when a check's current status is ok, 0 otherwise; a stale result is not healthy."),
	metric.WithUnit("{check}"),
}

// NewCheckState returns a new CheckState instrument.
func NewCheckState(
	m metric.Meter,
	opt ...metric.Int64GaugeOption,
) (CheckState, error) {
	// Check if the meter is nil.
	if m == nil {
		return CheckState{noop.Int64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newCheckStateOpts
	} else {
		opt = append(opt, newCheckStateOpts...)
	}

	i, err := m.Int64Gauge(
		"webitel.health.check.state",
		opt...,
	)
	if err != nil {
		return CheckState{noop.Int64Gauge{}}, err
	}
	return CheckState{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CheckState) Inst() metric.Int64Gauge {
	return m.Int64Gauge
}

// Name returns the semantic convention name of the instrument.
func (CheckState) Name() string {
	return "webitel.health.check.state"
}

// Unit returns the semantic convention unit of the instrument
func (CheckState) Unit() string {
	return "{check}"
}

// Description returns the semantic convention description of the instrument
func (CheckState) Description() string {
	return "1 when a check's current status is ok, 0 otherwise; a stale result is not healthy."
}

// Record records val to the current distribution for attrs.
//
// The checkGroup is the the group the check belongs to, which decides how a
// failure affects readiness.
//
// The checkName is the the name the check was registered under.
func (m CheckState) Record(
	ctx context.Context,
	val int64,
	checkGroup CheckGroupAttr,
	checkName string,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Gauge.Record(ctx, val, metric.WithAttributes(
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

	m.Int64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
func (m CheckState) RecordSet(ctx context.Context, val int64, set attribute.Set) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Int64Gauge.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Int64Gauge.Record(ctx, val, *o...)
}

// CheckStateObservable is an instrument used to record metric values conforming
// to the "webitel.health.check.state" semantic conventions. It represents the 1
// when a check's current status is ok, 0 otherwise; a stale result is not
// healthy.
type CheckStateObservable struct {
	metric.Int64ObservableGauge
}

var newCheckStateObservableOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("1 when a check's current status is ok, 0 otherwise; a stale result is not healthy."),
	metric.WithUnit("{check}"),
}

// NewCheckStateObservable returns a new CheckStateObservable instrument.
func NewCheckStateObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (CheckStateObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return CheckStateObservable{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newCheckStateObservableOpts
	} else {
		opt = append(opt, newCheckStateObservableOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.health.check.state",
		opt...,
	)
	if err != nil {
		return CheckStateObservable{noop.Int64ObservableGauge{}}, err
	}
	return CheckStateObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CheckStateObservable) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (CheckStateObservable) Name() string {
	return "webitel.health.check.state"
}

// Unit returns the semantic convention unit of the instrument
func (CheckStateObservable) Unit() string {
	return "{check}"
}

// Description returns the semantic convention description of the instrument
func (CheckStateObservable) Description() string {
	return "1 when a check's current status is ok, 0 otherwise; a stale result is not healthy."
}

// AttrCheckGroup returns a required attribute for the
// "webitel.health.check.group" semantic convention. It represents the group the
// check belongs to, which decides how a failure affects readiness.
func (CheckStateObservable) AttrCheckGroup(val CheckGroupAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.group", string(val))
}

// AttrCheckName returns a required attribute for the "webitel.health.check.name"
// semantic convention. It represents the name the check was registered under.
func (CheckStateObservable) AttrCheckName(val string) attribute.KeyValue {
	return attribute.String("webitel.health.check.name", val)
}

// CheckTransitions is an instrument used to record metric values conforming to
// the "webitel.health.check.transitions" semantic conventions. It represents the
// cumulative count of a check's status transitions, by the status transitioned
// into.
type CheckTransitions struct {
	metric.Int64Counter
}

var newCheckTransitionsOpts = []metric.Int64CounterOption{
	metric.WithDescription("Cumulative count of a check's status transitions, by the status transitioned into."),
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
	return "Cumulative count of a check's status transitions, by the status transitioned into."
}

// Add adds incr to the existing count for attrs.
//
// The checkGroup is the the group the check belongs to, which decides how a
// failure affects readiness.
//
// The checkName is the the name the check was registered under.
//
// The checkStatus is the the status a check transitioned into.
func (m CheckTransitions) Add(
	ctx context.Context,
	incr int64,
	checkGroup CheckGroupAttr,
	checkName string,
	checkStatus CheckStatusAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Counter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Counter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.health.check.group", string(checkGroup)),
			attribute.String("webitel.health.check.name", checkName),
			attribute.String("webitel.health.check.status", string(checkStatus)),
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
				attribute.String("webitel.health.check.status", string(checkStatus)),
			)...,
		),
	)

	m.Int64Counter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
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
// represents the cumulative count of a check's status transitions, by the status
// transitioned into.
type CheckTransitionsObservable struct {
	metric.Int64ObservableCounter
}

var newCheckTransitionsObservableOpts = []metric.Int64ObservableCounterOption{
	metric.WithDescription("Cumulative count of a check's status transitions, by the status transitioned into."),
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
	return "Cumulative count of a check's status transitions, by the status transitioned into."
}

// AttrCheckGroup returns a required attribute for the
// "webitel.health.check.group" semantic convention. It represents the group the
// check belongs to, which decides how a failure affects readiness.
func (CheckTransitionsObservable) AttrCheckGroup(val CheckGroupAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.group", string(val))
}

// AttrCheckName returns a required attribute for the "webitel.health.check.name"
// semantic convention. It represents the name the check was registered under.
func (CheckTransitionsObservable) AttrCheckName(val string) attribute.KeyValue {
	return attribute.String("webitel.health.check.name", val)
}

// AttrCheckStatus returns a required attribute for the
// "webitel.health.check.status" semantic convention. It represents the status a
// check transitioned into.
func (CheckTransitionsObservable) AttrCheckStatus(val CheckStatusAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.status", string(val))
}

// Ready is an instrument used to record metric values conforming to the
// "webitel.health.ready" semantic conventions. It represents the 1 when the
// registry reports the node ready to serve traffic, 0 otherwise.
type Ready struct {
	metric.Int64Gauge
}

var newReadyOpts = []metric.Int64GaugeOption{
	metric.WithDescription("1 when the registry reports the node ready to serve traffic, 0 otherwise."),
	metric.WithUnit("{node}"),
}

// NewReady returns a new Ready instrument.
func NewReady(
	m metric.Meter,
	opt ...metric.Int64GaugeOption,
) (Ready, error) {
	// Check if the meter is nil.
	if m == nil {
		return Ready{noop.Int64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newReadyOpts
	} else {
		opt = append(opt, newReadyOpts...)
	}

	i, err := m.Int64Gauge(
		"webitel.health.ready",
		opt...,
	)
	if err != nil {
		return Ready{noop.Int64Gauge{}}, err
	}
	return Ready{i}, nil
}

// Inst returns the underlying metric instrument.
func (m Ready) Inst() metric.Int64Gauge {
	return m.Int64Gauge
}

// Name returns the semantic convention name of the instrument.
func (Ready) Name() string {
	return "webitel.health.ready"
}

// Unit returns the semantic convention unit of the instrument
func (Ready) Unit() string {
	return "{node}"
}

// Description returns the semantic convention description of the instrument
func (Ready) Description() string {
	return "1 when the registry reports the node ready to serve traffic, 0 otherwise."
}

// Record records val to the current distribution for attrs.
func (m Ready) Record(ctx context.Context, val int64, attrs ...attribute.KeyValue) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Gauge.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributes(attrs...))
	m.Int64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
func (m Ready) RecordSet(ctx context.Context, val int64, set attribute.Set) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Int64Gauge.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Int64Gauge.Record(ctx, val, *o...)
}

// ReadyObservable is an instrument used to record metric values conforming to
// the "webitel.health.ready" semantic conventions. It represents the 1 when the
// registry reports the node ready to serve traffic, 0 otherwise.
type ReadyObservable struct {
	metric.Int64ObservableGauge
}

var newReadyObservableOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("1 when the registry reports the node ready to serve traffic, 0 otherwise."),
	metric.WithUnit("{node}"),
}

// NewReadyObservable returns a new ReadyObservable instrument.
func NewReadyObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (ReadyObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return ReadyObservable{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newReadyObservableOpts
	} else {
		opt = append(opt, newReadyObservableOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.health.ready",
		opt...,
	)
	if err != nil {
		return ReadyObservable{noop.Int64ObservableGauge{}}, err
	}
	return ReadyObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ReadyObservable) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (ReadyObservable) Name() string {
	return "webitel.health.ready"
}

// Unit returns the semantic convention unit of the instrument
func (ReadyObservable) Unit() string {
	return "{node}"
}

// Description returns the semantic convention description of the instrument
func (ReadyObservable) Description() string {
	return "1 when the registry reports the node ready to serve traffic, 0 otherwise."
}
