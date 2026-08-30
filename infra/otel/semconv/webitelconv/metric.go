// Code generated from semantic convention specification. DO NOT EDIT.

// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

// Package webitelconv provides types and functionality for OpenTelemetry semantic
// conventions in the "webitel" namespace.
package webitelconv

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// HealthCheckGroupAttr is an attribute conforming to the
// webitel.health.check.group semantic conventions. It represents the group the
// check belongs to, which decides how a failure affects readiness.
type HealthCheckGroupAttr string

var (
	// HealthCheckGroupLiveness is the is this process wedged; counts towards
	// readiness.
	HealthCheckGroupLiveness HealthCheckGroupAttr = "liveness"
	// HealthCheckGroupCritical is a node-local failure that takes the node out of
	// rotation.
	HealthCheckGroupCritical HealthCheckGroupAttr = "critical"
	// HealthCheckGroupInformational is the only degrades; the node stays in
	// rotation.
	HealthCheckGroupInformational HealthCheckGroupAttr = "informational"
)

// HealthCheckStatusAttr is an attribute conforming to the
// webitel.health.check.status semantic conventions. It represents the status a
// check transitioned into.
type HealthCheckStatusAttr string

var (
	// HealthCheckStatusOk is the check is passing.
	HealthCheckStatusOk HealthCheckStatusAttr = "ok"
	// HealthCheckStatusFail is the check has failed its threshold.
	HealthCheckStatusFail HealthCheckStatusAttr = "fail"
)

// HealthCheckDuration is an instrument used to record metric values conforming
// to the "webitel.health.check.duration" semantic conventions. It represents the
// elapsed time of a check's last completed run; absent until the first run
// lands.
type HealthCheckDuration struct {
	metric.Float64ObservableGauge
}

var newHealthCheckDurationOpts = []metric.Float64ObservableGaugeOption{
	metric.WithDescription("Elapsed time of a check's last completed run; absent until the first run lands."),
	metric.WithUnit("s"),
}

// NewHealthCheckDuration returns a new HealthCheckDuration instrument.
func NewHealthCheckDuration(
	m metric.Meter,
	opt ...metric.Float64ObservableGaugeOption,
) (HealthCheckDuration, error) {
	// Check if the meter is nil.
	if m == nil {
		return HealthCheckDuration{noop.Float64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newHealthCheckDurationOpts
	} else {
		opt = append(opt, newHealthCheckDurationOpts...)
	}

	i, err := m.Float64ObservableGauge(
		"webitel.health.check.duration",
		opt...,
	)
	if err != nil {
		return HealthCheckDuration{noop.Float64ObservableGauge{}}, err
	}
	return HealthCheckDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m HealthCheckDuration) Inst() metric.Float64ObservableGauge {
	return m.Float64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (HealthCheckDuration) Name() string {
	return "webitel.health.check.duration"
}

// Unit returns the semantic convention unit of the instrument
func (HealthCheckDuration) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (HealthCheckDuration) Description() string {
	return "Elapsed time of a check's last completed run; absent until the first run lands."
}

// AttrHealthCheckGroup returns a required attribute for the
// "webitel.health.check.group" semantic convention. It represents the group the
// check belongs to, which decides how a failure affects readiness.
func (HealthCheckDuration) AttrHealthCheckGroup(val HealthCheckGroupAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.group", string(val))
}

// AttrHealthCheckName returns a required attribute for the
// "webitel.health.check.name" semantic convention. It represents the name the
// check was registered under.
func (HealthCheckDuration) AttrHealthCheckName(val string) attribute.KeyValue {
	return attribute.String("webitel.health.check.name", val)
}

// HealthCheckState is an instrument used to record metric values conforming to
// the "webitel.health.check.state" semantic conventions. It represents the 1
// when a check's current status is ok, 0 otherwise; a stale result is not
// healthy.
type HealthCheckState struct {
	metric.Int64ObservableGauge
}

var newHealthCheckStateOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("1 when a check's current status is ok, 0 otherwise; a stale result is not healthy."),
	metric.WithUnit("1"),
}

// NewHealthCheckState returns a new HealthCheckState instrument.
func NewHealthCheckState(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (HealthCheckState, error) {
	// Check if the meter is nil.
	if m == nil {
		return HealthCheckState{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newHealthCheckStateOpts
	} else {
		opt = append(opt, newHealthCheckStateOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.health.check.state",
		opt...,
	)
	if err != nil {
		return HealthCheckState{noop.Int64ObservableGauge{}}, err
	}
	return HealthCheckState{i}, nil
}

// Inst returns the underlying metric instrument.
func (m HealthCheckState) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (HealthCheckState) Name() string {
	return "webitel.health.check.state"
}

// Unit returns the semantic convention unit of the instrument
func (HealthCheckState) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (HealthCheckState) Description() string {
	return "1 when a check's current status is ok, 0 otherwise; a stale result is not healthy."
}

// AttrHealthCheckGroup returns a required attribute for the
// "webitel.health.check.group" semantic convention. It represents the group the
// check belongs to, which decides how a failure affects readiness.
func (HealthCheckState) AttrHealthCheckGroup(val HealthCheckGroupAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.group", string(val))
}

// AttrHealthCheckName returns a required attribute for the
// "webitel.health.check.name" semantic convention. It represents the name the
// check was registered under.
func (HealthCheckState) AttrHealthCheckName(val string) attribute.KeyValue {
	return attribute.String("webitel.health.check.name", val)
}

// HealthCheckTransitions is an instrument used to record metric values
// conforming to the "webitel.health.check.transitions" semantic conventions. It
// represents the cumulative count of a check's status transitions, by the status
// transitioned into.
type HealthCheckTransitions struct {
	metric.Int64ObservableCounter
}

var newHealthCheckTransitionsOpts = []metric.Int64ObservableCounterOption{
	metric.WithDescription("Cumulative count of a check's status transitions, by the status transitioned into."),
	metric.WithUnit("{transition}"),
}

// NewHealthCheckTransitions returns a new HealthCheckTransitions instrument.
func NewHealthCheckTransitions(
	m metric.Meter,
	opt ...metric.Int64ObservableCounterOption,
) (HealthCheckTransitions, error) {
	// Check if the meter is nil.
	if m == nil {
		return HealthCheckTransitions{noop.Int64ObservableCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newHealthCheckTransitionsOpts
	} else {
		opt = append(opt, newHealthCheckTransitionsOpts...)
	}

	i, err := m.Int64ObservableCounter(
		"webitel.health.check.transitions",
		opt...,
	)
	if err != nil {
		return HealthCheckTransitions{noop.Int64ObservableCounter{}}, err
	}
	return HealthCheckTransitions{i}, nil
}

// Inst returns the underlying metric instrument.
func (m HealthCheckTransitions) Inst() metric.Int64ObservableCounter {
	return m.Int64ObservableCounter
}

// Name returns the semantic convention name of the instrument.
func (HealthCheckTransitions) Name() string {
	return "webitel.health.check.transitions"
}

// Unit returns the semantic convention unit of the instrument
func (HealthCheckTransitions) Unit() string {
	return "{transition}"
}

// Description returns the semantic convention description of the instrument
func (HealthCheckTransitions) Description() string {
	return "Cumulative count of a check's status transitions, by the status transitioned into."
}

// AttrHealthCheckGroup returns a required attribute for the
// "webitel.health.check.group" semantic convention. It represents the group the
// check belongs to, which decides how a failure affects readiness.
func (HealthCheckTransitions) AttrHealthCheckGroup(val HealthCheckGroupAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.group", string(val))
}

// AttrHealthCheckName returns a required attribute for the
// "webitel.health.check.name" semantic convention. It represents the name the
// check was registered under.
func (HealthCheckTransitions) AttrHealthCheckName(val string) attribute.KeyValue {
	return attribute.String("webitel.health.check.name", val)
}

// AttrHealthCheckStatus returns a required attribute for the
// "webitel.health.check.status" semantic convention. It represents the status a
// check transitioned into.
func (HealthCheckTransitions) AttrHealthCheckStatus(val HealthCheckStatusAttr) attribute.KeyValue {
	return attribute.String("webitel.health.check.status", string(val))
}

// HealthReady is an instrument used to record metric values conforming to the
// "webitel.health.ready" semantic conventions. It represents the 1 when the
// registry reports the node ready to serve traffic, 0 otherwise.
type HealthReady struct {
	metric.Int64ObservableGauge
}

var newHealthReadyOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("1 when the registry reports the node ready to serve traffic, 0 otherwise."),
	metric.WithUnit("1"),
}

// NewHealthReady returns a new HealthReady instrument.
func NewHealthReady(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (HealthReady, error) {
	// Check if the meter is nil.
	if m == nil {
		return HealthReady{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newHealthReadyOpts
	} else {
		opt = append(opt, newHealthReadyOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.health.ready",
		opt...,
	)
	if err != nil {
		return HealthReady{noop.Int64ObservableGauge{}}, err
	}
	return HealthReady{i}, nil
}

// Inst returns the underlying metric instrument.
func (m HealthReady) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (HealthReady) Name() string {
	return "webitel.health.ready"
}

// Unit returns the semantic convention unit of the instrument
func (HealthReady) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (HealthReady) Description() string {
	return "1 when the registry reports the node ready to serve traffic, 0 otherwise."
}
