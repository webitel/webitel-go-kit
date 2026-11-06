// Code generated from semantic convention specification. DO NOT EDIT.

// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

// Package outboxconv provides types and functionality for OpenTelemetry semantic
// conventions in the "webitel.outbox" namespace.
package outboxconv

import (
	"context"

	"github.com/webitel/webitel-go-kit/infra/otel/semconv/internal/metricpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// RelayStateAttr is an attribute conforming to the webitel.outbox.relay.state
// semantic conventions. It represents the state of the node in the outbox relay.
type RelayStateAttr string

var (
	// RelayStateLeader is the node runs the outbox relay.
	RelayStateLeader RelayStateAttr = "leader"
	// RelayStateFollower is the node does not run the outbox relay.
	RelayStateFollower RelayStateAttr = "follower"
)

// EventAge is an instrument used to record metric values conforming to the
// "webitel.outbox.event.age" semantic conventions. It represents the age of the
// oldest outbox event waiting to be published to the broker.
type EventAge struct {
	metric.Float64Gauge
}

var newEventAgeOpts = []metric.Float64GaugeOption{
	metric.WithDescription("Age of the oldest outbox event waiting to be published to the broker."),
	metric.WithUnit("s"),
}

// NewEventAge returns a new EventAge instrument.
func NewEventAge(
	m metric.Meter,
	opt ...metric.Float64GaugeOption,
) (EventAge, error) {
	// Check if the meter is nil.
	if m == nil {
		return EventAge{noop.Float64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newEventAgeOpts
	} else {
		opt = append(opt, newEventAgeOpts...)
	}

	i, err := m.Float64Gauge(
		"webitel.outbox.event.age",
		opt...,
	)
	if err != nil {
		return EventAge{noop.Float64Gauge{}}, err
	}
	return EventAge{i}, nil
}

// Inst returns the underlying metric instrument.
func (m EventAge) Inst() metric.Float64Gauge {
	return m.Float64Gauge
}

// Name returns the semantic convention name of the instrument.
func (EventAge) Name() string {
	return "webitel.outbox.event.age"
}

// Unit returns the semantic convention unit of the instrument
func (EventAge) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (EventAge) Description() string {
	return "Age of the oldest outbox event waiting to be published to the broker."
}

// Record records val to the current distribution for attrs.
//
// The value MUST be 0 when no event is waiting. This metric MUST be reported
// only by the node that runs the relay.
func (m EventAge) Record(ctx context.Context, val float64, attrs ...attribute.KeyValue) {
	if !m.Float64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Float64Gauge.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributes(attrs...))
	m.Float64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// The value MUST be 0 when no event is waiting. This metric MUST be reported
// only by the node that runs the relay.
func (m EventAge) RecordSet(ctx context.Context, val float64, set attribute.Set) {
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

// EventAgeObservable is an instrument used to record metric values conforming to
// the "webitel.outbox.event.age" semantic conventions. It represents the age of
// the oldest outbox event waiting to be published to the broker.
type EventAgeObservable struct {
	metric.Float64ObservableGauge
}

var newEventAgeObservableOpts = []metric.Float64ObservableGaugeOption{
	metric.WithDescription("Age of the oldest outbox event waiting to be published to the broker."),
	metric.WithUnit("s"),
}

// NewEventAgeObservable returns a new EventAgeObservable instrument.
func NewEventAgeObservable(
	m metric.Meter,
	opt ...metric.Float64ObservableGaugeOption,
) (EventAgeObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return EventAgeObservable{noop.Float64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newEventAgeObservableOpts
	} else {
		opt = append(opt, newEventAgeObservableOpts...)
	}

	i, err := m.Float64ObservableGauge(
		"webitel.outbox.event.age",
		opt...,
	)
	if err != nil {
		return EventAgeObservable{noop.Float64ObservableGauge{}}, err
	}
	return EventAgeObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m EventAgeObservable) Inst() metric.Float64ObservableGauge {
	return m.Float64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (EventAgeObservable) Name() string {
	return "webitel.outbox.event.age"
}

// Unit returns the semantic convention unit of the instrument
func (EventAgeObservable) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (EventAgeObservable) Description() string {
	return "Age of the oldest outbox event waiting to be published to the broker."
}

// EventCount is an instrument used to record metric values conforming to the
// "webitel.outbox.event.count" semantic conventions. It represents the number of
// outbox events waiting to be published to the broker.
type EventCount struct {
	metric.Int64UpDownCounter
}

var newEventCountOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("Number of outbox events waiting to be published to the broker."),
	metric.WithUnit("{event}"),
}

// NewEventCount returns a new EventCount instrument.
func NewEventCount(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (EventCount, error) {
	// Check if the meter is nil.
	if m == nil {
		return EventCount{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newEventCountOpts
	} else {
		opt = append(opt, newEventCountOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.outbox.event.count",
		opt...,
	)
	if err != nil {
		return EventCount{noop.Int64UpDownCounter{}}, err
	}
	return EventCount{i}, nil
}

// Inst returns the underlying metric instrument.
func (m EventCount) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (EventCount) Name() string {
	return "webitel.outbox.event.count"
}

// Unit returns the semantic convention unit of the instrument
func (EventCount) Unit() string {
	return "{event}"
}

// Description returns the semantic convention description of the instrument
func (EventCount) Description() string {
	return "Number of outbox events waiting to be published to the broker."
}

// Add adds incr to the existing count for attrs.
//
// This metric MUST be reported only by the node that runs the relay.
func (m EventCount) Add(ctx context.Context, incr int64, attrs ...attribute.KeyValue) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr)
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(*o, metric.WithAttributes(attrs...))
	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// This metric MUST be reported only by the node that runs the relay.
func (m EventCount) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// EventCountObservable is an instrument used to record metric values conforming
// to the "webitel.outbox.event.count" semantic conventions. It represents the
// number of outbox events waiting to be published to the broker.
type EventCountObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newEventCountObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("Number of outbox events waiting to be published to the broker."),
	metric.WithUnit("{event}"),
}

// NewEventCountObservable returns a new EventCountObservable instrument.
func NewEventCountObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (EventCountObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return EventCountObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newEventCountObservableOpts
	} else {
		opt = append(opt, newEventCountObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.outbox.event.count",
		opt...,
	)
	if err != nil {
		return EventCountObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return EventCountObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m EventCountObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (EventCountObservable) Name() string {
	return "webitel.outbox.event.count"
}

// Unit returns the semantic convention unit of the instrument
func (EventCountObservable) Unit() string {
	return "{event}"
}

// Description returns the semantic convention description of the instrument
func (EventCountObservable) Description() string {
	return "Number of outbox events waiting to be published to the broker."
}

// RelayPoisoned is an instrument used to record metric values conforming to the
// "webitel.outbox.relay.poisoned" semantic conventions. It represents the number
// of outbox events moved to the poison queue after the relay ran out of retries.
type RelayPoisoned struct {
	metric.Int64Counter
}

var newRelayPoisonedOpts = []metric.Int64CounterOption{
	metric.WithDescription("Number of outbox events moved to the poison queue after the relay ran out of retries."),
	metric.WithUnit("{event}"),
}

// NewRelayPoisoned returns a new RelayPoisoned instrument.
func NewRelayPoisoned(
	m metric.Meter,
	opt ...metric.Int64CounterOption,
) (RelayPoisoned, error) {
	// Check if the meter is nil.
	if m == nil {
		return RelayPoisoned{noop.Int64Counter{}}, nil
	}

	if len(opt) == 0 {
		opt = newRelayPoisonedOpts
	} else {
		opt = append(opt, newRelayPoisonedOpts...)
	}

	i, err := m.Int64Counter(
		"webitel.outbox.relay.poisoned",
		opt...,
	)
	if err != nil {
		return RelayPoisoned{noop.Int64Counter{}}, err
	}
	return RelayPoisoned{i}, nil
}

// Inst returns the underlying metric instrument.
func (m RelayPoisoned) Inst() metric.Int64Counter {
	return m.Int64Counter
}

// Name returns the semantic convention name of the instrument.
func (RelayPoisoned) Name() string {
	return "webitel.outbox.relay.poisoned"
}

// Unit returns the semantic convention unit of the instrument
func (RelayPoisoned) Unit() string {
	return "{event}"
}

// Description returns the semantic convention description of the instrument
func (RelayPoisoned) Description() string {
	return "Number of outbox events moved to the poison queue after the relay ran out of retries."
}

// Add adds incr to the existing count for attrs.
//
// An event MUST be counted once, when it is moved to the poison queue. An event
// in the poison queue is never published.
func (m RelayPoisoned) Add(ctx context.Context, incr int64, attrs ...attribute.KeyValue) {
	if !m.Int64Counter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Counter.Add(ctx, incr)
		return
	}

	o := metricpool.AddOptions()
	defer metricpool.PutAddOptions(o)

	*o = append(*o, metric.WithAttributes(attrs...))
	m.Int64Counter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// An event MUST be counted once, when it is moved to the poison queue. An event
// in the poison queue is never published.
func (m RelayPoisoned) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// RelayPoisonedObservable is an instrument used to record metric values
// conforming to the "webitel.outbox.relay.poisoned" semantic conventions. It
// represents the number of outbox events moved to the poison queue after the
// relay ran out of retries.
type RelayPoisonedObservable struct {
	metric.Int64ObservableCounter
}

var newRelayPoisonedObservableOpts = []metric.Int64ObservableCounterOption{
	metric.WithDescription("Number of outbox events moved to the poison queue after the relay ran out of retries."),
	metric.WithUnit("{event}"),
}

// NewRelayPoisonedObservable returns a new RelayPoisonedObservable instrument.
func NewRelayPoisonedObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableCounterOption,
) (RelayPoisonedObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return RelayPoisonedObservable{noop.Int64ObservableCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newRelayPoisonedObservableOpts
	} else {
		opt = append(opt, newRelayPoisonedObservableOpts...)
	}

	i, err := m.Int64ObservableCounter(
		"webitel.outbox.relay.poisoned",
		opt...,
	)
	if err != nil {
		return RelayPoisonedObservable{noop.Int64ObservableCounter{}}, err
	}
	return RelayPoisonedObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m RelayPoisonedObservable) Inst() metric.Int64ObservableCounter {
	return m.Int64ObservableCounter
}

// Name returns the semantic convention name of the instrument.
func (RelayPoisonedObservable) Name() string {
	return "webitel.outbox.relay.poisoned"
}

// Unit returns the semantic convention unit of the instrument
func (RelayPoisonedObservable) Unit() string {
	return "{event}"
}

// Description returns the semantic convention description of the instrument
func (RelayPoisonedObservable) Description() string {
	return "Number of outbox events moved to the poison queue after the relay ran out of retries."
}

// RelayStatus is an instrument used to record metric values conforming to the
// "webitel.outbox.relay.status" semantic conventions. It represents the current
// status of the node in the outbox relay.
type RelayStatus struct {
	metric.Int64UpDownCounter
}

var newRelayStatusOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("The current status of the node in the outbox relay."),
	metric.WithUnit("1"),
}

// NewRelayStatus returns a new RelayStatus instrument.
func NewRelayStatus(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (RelayStatus, error) {
	// Check if the meter is nil.
	if m == nil {
		return RelayStatus{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newRelayStatusOpts
	} else {
		opt = append(opt, newRelayStatusOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.outbox.relay.status",
		opt...,
	)
	if err != nil {
		return RelayStatus{noop.Int64UpDownCounter{}}, err
	}
	return RelayStatus{i}, nil
}

// Inst returns the underlying metric instrument.
func (m RelayStatus) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (RelayStatus) Name() string {
	return "webitel.outbox.relay.status"
}

// Unit returns the semantic convention unit of the instrument
func (RelayStatus) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (RelayStatus) Description() string {
	return "The current status of the node in the outbox relay."
}

// Add adds incr to the existing count for attrs.
//
// The relayState is the the state of the node in the outbox relay.
//
// A timeseries is produced for every possible value of
// `webitel.outbox.relay.state`. The value of this metric is 1 for the current
// state of the node, and 0 for the other states.
func (m RelayStatus) Add(
	ctx context.Context,
	incr int64,
	relayState RelayStateAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.outbox.relay.state", string(relayState)),
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
				attribute.String("webitel.outbox.relay.state", string(relayState)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// A timeseries is produced for every possible value of
// `webitel.outbox.relay.state`. The value of this metric is 1 for the current
// state of the node, and 0 for the other states.
func (m RelayStatus) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// RelayStatusObservable is an instrument used to record metric values conforming
// to the "webitel.outbox.relay.status" semantic conventions. It represents the
// current status of the node in the outbox relay.
type RelayStatusObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newRelayStatusObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("The current status of the node in the outbox relay."),
	metric.WithUnit("1"),
}

// NewRelayStatusObservable returns a new RelayStatusObservable instrument.
func NewRelayStatusObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (RelayStatusObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return RelayStatusObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newRelayStatusObservableOpts
	} else {
		opt = append(opt, newRelayStatusObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.outbox.relay.status",
		opt...,
	)
	if err != nil {
		return RelayStatusObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return RelayStatusObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m RelayStatusObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (RelayStatusObservable) Name() string {
	return "webitel.outbox.relay.status"
}

// Unit returns the semantic convention unit of the instrument
func (RelayStatusObservable) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (RelayStatusObservable) Description() string {
	return "The current status of the node in the outbox relay."
}

// AttrRelayState returns a required attribute for the
// "webitel.outbox.relay.state" semantic convention. It represents the state of
// the node in the outbox relay.
func (RelayStatusObservable) AttrRelayState(val RelayStateAttr) attribute.KeyValue {
	return attribute.String("webitel.outbox.relay.state", string(val))
}
