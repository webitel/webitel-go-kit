// Code generated from semantic convention specification. DO NOT EDIT.

// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

// Package dcctlconv provides types and functionality for OpenTelemetry semantic
// conventions in the "webitel.dcctl" namespace.
package dcctlconv

import (
	"context"

	"github.com/webitel/webitel-go-kit/infra/otel/semconv/internal/metricpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// ErrorTypeAttr is an attribute conforming to the error.type semantic
// conventions. It represents the describes a class of error the operation ended
// with.
type ErrorTypeAttr string

var (
	// ErrorTypeOther is a fallback error value to be used when the instrumentation
	// doesn't define a custom value.
	ErrorTypeOther ErrorTypeAttr = "_OTHER"
)

// AlarmNameAttr is an attribute conforming to the webitel.dcctl.alarm.name
// semantic conventions. It represents the identifier of the alarm.
type AlarmNameAttr string

var (
	// AlarmNameConflict is the two valid records have the same generation and
	// different active datacenters.
	AlarmNameConflict AlarmNameAttr = "conflict"
	// AlarmNameUncommitted is a voter holds a record with a generation above the
	// effective one, without a majority and without `forced`.
	AlarmNameUncommitted AlarmNameAttr = "uncommitted"
	// AlarmNameStale is the voter of a datacenter holds a generation lower than the
	// effective one.
	AlarmNameStale AlarmNameAttr = "stale"
	// AlarmNameSplitBrain is a managed job of a datacenter that is not active is
	// not stopped.
	AlarmNameSplitBrain AlarmNameAttr = "split_brain"
	// AlarmNameDBSplit is the Patroni leader of a datacenter that is not active is
	// a primary.
	AlarmNameDBSplit AlarmNameAttr = "db_split"
	// AlarmNameMultiplePrimaries is the more than one primary is reachable across
	// all datacenters.
	AlarmNameMultiplePrimaries AlarmNameAttr = "multiple_primaries"
	// AlarmNameDBPrimaryMissing is the active datacenter has no primary.
	AlarmNameDBPrimaryMissing AlarmNameAttr = "db_primary_missing"
	// AlarmNameDatacenterUnreachable is the datacenter has been unreachable longer
	// than the degraded threshold.
	AlarmNameDatacenterUnreachable AlarmNameAttr = "datacenter_unreachable"
	// AlarmNameDBLeaderMismatch is the datacenter of the Patroni leader is not the
	// active datacenter.
	AlarmNameDBLeaderMismatch AlarmNameAttr = "db_leader_mismatch"
	// AlarmNameLeaseLost is the lease is not renewed, or does not match the
	// effective state.
	AlarmNameLeaseLost AlarmNameAttr = "lease_lost"
	// AlarmNameAppsDegraded is a managed job of the active datacenter is not
	// healthy.
	AlarmNameAppsDegraded AlarmNameAttr = "apps_degraded"
	// AlarmNameDBDegraded is a Patroni member of the datacenter is unreachable or
	// unusable, or Patroni is paused.
	AlarmNameDBDegraded AlarmNameAttr = "db_degraded"
	// AlarmNameStandbyNotFollowing is the Patroni leader of a datacenter that is
	// not active does not follow the active datacenter.
	AlarmNameStandbyNotFollowing AlarmNameAttr = "standby_not_following"
	// AlarmNameVoterUnreachable is the voter has been unreachable longer than the
	// degraded threshold.
	AlarmNameVoterUnreachable AlarmNameAttr = "voter_unreachable"
	// AlarmNameQuorumLost is the fewer voters than a majority are reachable.
	AlarmNameQuorumLost AlarmNameAttr = "quorum_lost"
	// AlarmNameArbiterUninitialized is the arbiter is reachable and its storage is
	// empty.
	AlarmNameArbiterUninitialized AlarmNameAttr = "arbiter_uninitialized"
	// AlarmNameOperationInterrupted is an operation is in the `interrupted` state.
	AlarmNameOperationInterrupted AlarmNameAttr = "operation_interrupted"
)

// DatacenterStateAttr is an attribute conforming to the
// webitel.dcctl.datacenter.state semantic conventions. It represents the state
// of the datacenter.
type DatacenterStateAttr string

var (
	// DatacenterStateUnreachable is the no source of the datacenter is reachable.
	DatacenterStateUnreachable DatacenterStateAttr = "unreachable"
	// DatacenterStateUninitialized is the no voter holds the coordination state.
	DatacenterStateUninitialized DatacenterStateAttr = "uninitialized"
	// DatacenterStateUndetermined is the role of the datacenter cannot be told,
	// because the voters give no decided effective state.
	DatacenterStateUndetermined DatacenterStateAttr = "undetermined"
	// DatacenterStateActive is the datacenter is the active datacenter of the
	// effective state.
	DatacenterStateActive DatacenterStateAttr = "active"
	// DatacenterStateSplitBrain is the datacenter is not active, and a managed job
	// of it is not stopped.
	DatacenterStateSplitBrain DatacenterStateAttr = "split_brain"
	// DatacenterStateDBSplit is the datacenter is not active, and its Patroni
	// leader is a primary.
	DatacenterStateDBSplit DatacenterStateAttr = "db_split"
	// DatacenterStateStale is the voter of the datacenter holds a generation lower
	// than the effective one.
	DatacenterStateStale DatacenterStateAttr = "stale"
	// DatacenterStateNeedsRecovery is the datacenter is marked as needing recovery
	// in the effective state.
	DatacenterStateNeedsRecovery DatacenterStateAttr = "needs_recovery"
	// DatacenterStateStandby is the datacenter is not active, and none of the other
	// states apply.
	DatacenterStateStandby DatacenterStateAttr = "standby"
)

// JobStateAttr is an attribute conforming to the webitel.dcctl.job.state
// semantic conventions. It represents the state of a job managed by dcctl.
type JobStateAttr string

var (
	// JobStateHealthy is the job is running and healthy.
	JobStateHealthy JobStateAttr = "healthy"
	// JobStateUnhealthy is the job is not stopped and is not healthy, including
	// while it starts.
	JobStateUnhealthy JobStateAttr = "unhealthy"
	// JobStateStopping is the job is stopped and has allocations that are still
	// pending, running or unknown.
	JobStateStopping JobStateAttr = "stopping"
	// JobStateStopped is the job is stopped and has no allocations left.
	JobStateStopped JobStateAttr = "stopped"
)

// OperationModeAttr is an attribute conforming to the
// webitel.dcctl.operation.mode semantic conventions. It represents the mode in
// which the operation changes the coordination state.
type OperationModeAttr string

var (
	// OperationModeQuorum is the operation changes the state with a majority of
	// voters.
	OperationModeQuorum OperationModeAttr = "quorum"
	// OperationModeForced is the operation changes the state without a majority of
	// voters.
	OperationModeForced OperationModeAttr = "forced"
)

// OperationStateAttr is an attribute conforming to the
// webitel.dcctl.operation.state semantic conventions. It represents the state of
// the operation.
type OperationStateAttr string

var (
	// OperationStateRunning is the operation is in progress.
	OperationStateRunning OperationStateAttr = "running"
	// OperationStateSucceeded is the operation completed.
	OperationStateSucceeded OperationStateAttr = "succeeded"
	// OperationStateFailed is the operation failed.
	OperationStateFailed OperationStateAttr = "failed"
	// OperationStateInterrupted is the operation was interrupted and waits to be
	// resumed or abandoned.
	OperationStateInterrupted OperationStateAttr = "interrupted"
	// OperationStateAbandoned is the operation was abandoned.
	OperationStateAbandoned OperationStateAttr = "abandoned"
)

// OperationTypeAttr is an attribute conforming to the
// webitel.dcctl.operation.type semantic conventions. It represents the type of
// the operation.
type OperationTypeAttr string

var (
	// OperationTypeSwitch is the planned switch of the active datacenter.
	OperationTypeSwitch OperationTypeAttr = "switch"
	// OperationTypeFailover is the forced switch of the active datacenter.
	OperationTypeFailover OperationTypeAttr = "failover"
	// OperationTypeDemote is the demotion of the database of a datacenter to a
	// standby.
	OperationTypeDemote OperationTypeAttr = "demote"
	// OperationTypeRepoint is the repointing of the standby database of a
	// datacenter to the active datacenter.
	OperationTypeRepoint OperationTypeAttr = "repoint"
	// OperationTypeMarkRecovered is the removal of a datacenter from the
	// datacenters that need recovery.
	OperationTypeMarkRecovered OperationTypeAttr = "mark_recovered"
	// OperationTypeResolveConflict is the resolution of a conflict between records
	// of the coordination state.
	OperationTypeResolveConflict OperationTypeAttr = "resolve_conflict"
	// OperationTypeInitialize is the initialization of the coordination state.
	OperationTypeInitialize OperationTypeAttr = "initialize"
	// OperationTypeInitializeArbiter is the initialization of the arbiter.
	OperationTypeInitializeArbiter OperationTypeAttr = "initialize_arbiter"
	// OperationTypeSyncApps is the synchronization of managed jobs with the active
	// datacenter.
	OperationTypeSyncApps OperationTypeAttr = "sync_apps"
)

// PatroniRoleAttr is an attribute conforming to the webitel.dcctl.patroni.role
// semantic conventions. It represents the Patroni role of a cluster member.
type PatroniRoleAttr string

var (
	// PatroniRolePrimary is the member is a primary.
	PatroniRolePrimary PatroniRoleAttr = "primary"
	// PatroniRoleStandbyLeader is the member is the leader of a standby cluster.
	PatroniRoleStandbyLeader PatroniRoleAttr = "standby_leader"
	// PatroniRoleReplica is the member is a replica.
	PatroniRoleReplica PatroniRoleAttr = "replica"
	// PatroniRoleOther is the member has a role other than the listed ones.
	PatroniRoleOther PatroniRoleAttr = "other"
	// PatroniRoleUnknown is the role is not known because the member is not
	// reachable.
	PatroniRoleUnknown PatroniRoleAttr = "unknown"
)

// QuorumModeAttr is an attribute conforming to the webitel.dcctl.quorum.mode
// semantic conventions. It represents the mode in which the coordination state
// can be changed.
type QuorumModeAttr string

var (
	// QuorumModeQuorum is a majority of voters is reachable.
	QuorumModeQuorum QuorumModeAttr = "quorum"
	// QuorumModeForcedOnly is a majority of voters is not reachable, and only a
	// forced operation can change the state.
	QuorumModeForcedOnly QuorumModeAttr = "forced_only"
)

// SourceStateAttr is an attribute conforming to the webitel.dcctl.source.state
// semantic conventions. It represents the state of a source of dcctl.
type SourceStateAttr string

var (
	// SourceStateOk is the all the queries to the source succeed.
	SourceStateOk SourceStateAttr = "ok"
	// SourceStatePartial is the some of the queries to the source fail.
	SourceStatePartial SourceStateAttr = "partial"
	// SourceStateUnreachable is the source is not reachable.
	SourceStateUnreachable SourceStateAttr = "unreachable"
)

// SourceSystemAttr is an attribute conforming to the webitel.dcctl.source.system
// semantic conventions. It represents the system dcctl reads the state from.
type SourceSystemAttr string

var (
	// SourceSystemConsul is the [Consul].
	//
	// [Consul]: https://www.consul.io/
	SourceSystemConsul SourceSystemAttr = "consul"
	// SourceSystemNomad is the [Nomad].
	//
	// [Nomad]: https://www.nomadproject.io/
	SourceSystemNomad SourceSystemAttr = "nomad"
	// SourceSystemPatroni is the [Patroni].
	//
	// [Patroni]: https://patroni.readthedocs.io/
	SourceSystemPatroni SourceSystemAttr = "patroni"
	// SourceSystemArbiter is the dcctl arbiter.
	SourceSystemArbiter SourceSystemAttr = "arbiter"
)

// StepStateAttr is an attribute conforming to the webitel.dcctl.step.state
// semantic conventions. It represents the state of the step of an operation.
type StepStateAttr string

var (
	// StepStatePending is the step has not started.
	StepStatePending StepStateAttr = "pending"
	// StepStateRunning is the step is in progress.
	StepStateRunning StepStateAttr = "running"
	// StepStateSucceeded is the step completed.
	StepStateSucceeded StepStateAttr = "succeeded"
	// StepStateFailed is the step failed.
	StepStateFailed StepStateAttr = "failed"
	// StepStateSkipped is the step was skipped.
	StepStateSkipped StepStateAttr = "skipped"
)

// StepTypeAttr is an attribute conforming to the webitel.dcctl.step.type
// semantic conventions. It represents the type of the step of an operation.
type StepTypeAttr string

var (
	// StepTypeAppsPreflight is the managed jobs of the datacenter are checked
	// before the operation changes them.
	StepTypeAppsPreflight StepTypeAttr = "apps_preflight"
	// StepTypeAppsStop is the managed jobs of a stage are stopped.
	StepTypeAppsStop StepTypeAttr = "apps_stop"
	// StepTypeAppsStart is the managed jobs of a stage are started.
	StepTypeAppsStart StepTypeAttr = "apps_start"
	// StepTypeGeneration is the generation of the coordination state is changed.
	StepTypeGeneration StepTypeAttr = "generation"
	// StepTypeVerify is the result of the operation is verified.
	StepTypeVerify StepTypeAttr = "verify"
	// StepTypePatroniDemote is the Patroni cluster of the datacenter is demoted to
	// a standby.
	StepTypePatroniDemote StepTypeAttr = "patroni_demote"
	// StepTypePatroniCatchup is the standby waits to catch up with the primary.
	StepTypePatroniCatchup StepTypeAttr = "patroni_catchup"
	// StepTypePatroniNoPrimary is the step waits until the datacenter has no
	// primary.
	StepTypePatroniNoPrimary StepTypeAttr = "patroni_no_primary"
	// StepTypePatroniPromote is the Patroni cluster of the datacenter is promoted.
	StepTypePatroniPromote StepTypeAttr = "patroni_promote"
	// StepTypePatroniRepoint is the standby cluster is pointed to the active
	// datacenter.
	StepTypePatroniRepoint StepTypeAttr = "patroni_repoint"
	// StepTypePatroniFollow is the standby cluster starts to follow the active
	// datacenter.
	StepTypePatroniFollow StepTypeAttr = "patroni_follow"
	// StepTypePatroniSwitchover is the patroni switches the leader over to the
	// target datacenter.
	StepTypePatroniSwitchover StepTypeAttr = "patroni_switchover"
	// StepTypePatroniFailover is the patroni fails over inside one cluster that has
	// no leader (stretch profile).
	StepTypePatroniFailover StepTypeAttr = "patroni_failover"
)

// VoterStateAttr is an attribute conforming to the webitel.dcctl.voter.state
// semantic conventions. It represents the state of the voter.
type VoterStateAttr string

var (
	// VoterStateOk is the voter is reachable and holds the coordination state.
	VoterStateOk VoterStateAttr = "ok"
	// VoterStateUnreachable is the voter is not reachable.
	VoterStateUnreachable VoterStateAttr = "unreachable"
	// VoterStateUninitialized is the voter is reachable and holds no coordination
	// state.
	VoterStateUninitialized VoterStateAttr = "uninitialized"
	// VoterStateInvalid is the voter answered, but its `dcctl/state` record cannot
	// be decoded.
	VoterStateInvalid VoterStateAttr = "invalid"
)

// AlarmCount is an instrument used to record metric values conforming to the
// "webitel.dcctl.alarm.count" semantic conventions. It represents the number of
// alarms that are currently raised.
type AlarmCount struct {
	metric.Int64UpDownCounter
}

var newAlarmCountOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("Number of alarms that are currently raised."),
	metric.WithUnit("{alarm}"),
}

// NewAlarmCount returns a new AlarmCount instrument.
func NewAlarmCount(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (AlarmCount, error) {
	// Check if the meter is nil.
	if m == nil {
		return AlarmCount{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newAlarmCountOpts
	} else {
		opt = append(opt, newAlarmCountOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.dcctl.alarm.count",
		opt...,
	)
	if err != nil {
		return AlarmCount{noop.Int64UpDownCounter{}}, err
	}
	return AlarmCount{i}, nil
}

// Inst returns the underlying metric instrument.
func (m AlarmCount) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (AlarmCount) Name() string {
	return "webitel.dcctl.alarm.count"
}

// Unit returns the semantic convention unit of the instrument
func (AlarmCount) Unit() string {
	return "{alarm}"
}

// Description returns the semantic convention description of the instrument
func (AlarmCount) Description() string {
	return "Number of alarms that are currently raised."
}

// Add adds incr to the existing count for attrs.
//
// The alarmName is the the identifier of the alarm.
//
// All additional attrs passed are included in the recorded value.
func (m AlarmCount) Add(
	ctx context.Context,
	incr int64,
	alarmName AlarmNameAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.dcctl.alarm.name", string(alarmName)),
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
				attribute.String("webitel.dcctl.alarm.name", string(alarmName)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
func (m AlarmCount) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// AttrWebitelDatacenterName returns an optional attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (AlarmCount) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// AttrVoterName returns an optional attribute for the "webitel.dcctl.voter.name"
// semantic convention. It represents the name of the voter.
func (AlarmCount) AttrVoterName(val string) attribute.KeyValue {
	return attribute.String("webitel.dcctl.voter.name", val)
}

// AlarmCountObservable is an instrument used to record metric values conforming
// to the "webitel.dcctl.alarm.count" semantic conventions. It represents the
// number of alarms that are currently raised.
type AlarmCountObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newAlarmCountObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("Number of alarms that are currently raised."),
	metric.WithUnit("{alarm}"),
}

// NewAlarmCountObservable returns a new AlarmCountObservable instrument.
func NewAlarmCountObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (AlarmCountObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return AlarmCountObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newAlarmCountObservableOpts
	} else {
		opt = append(opt, newAlarmCountObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.dcctl.alarm.count",
		opt...,
	)
	if err != nil {
		return AlarmCountObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return AlarmCountObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m AlarmCountObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (AlarmCountObservable) Name() string {
	return "webitel.dcctl.alarm.count"
}

// Unit returns the semantic convention unit of the instrument
func (AlarmCountObservable) Unit() string {
	return "{alarm}"
}

// Description returns the semantic convention description of the instrument
func (AlarmCountObservable) Description() string {
	return "Number of alarms that are currently raised."
}

// AttrAlarmName returns a required attribute for the "webitel.dcctl.alarm.name"
// semantic convention. It represents the identifier of the alarm.
func (AlarmCountObservable) AttrAlarmName(val AlarmNameAttr) attribute.KeyValue {
	return attribute.String("webitel.dcctl.alarm.name", string(val))
}

// AttrWebitelDatacenterName returns an optional attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (AlarmCountObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// AttrVoterName returns an optional attribute for the "webitel.dcctl.voter.name"
// semantic convention. It represents the name of the voter.
func (AlarmCountObservable) AttrVoterName(val string) attribute.KeyValue {
	return attribute.String("webitel.dcctl.voter.name", val)
}

// CollectDuration is an instrument used to record metric values conforming to
// the "webitel.dcctl.collect.duration" semantic conventions. It represents the
// duration of the last completed collection cycle.
type CollectDuration struct {
	metric.Float64Gauge
}

var newCollectDurationOpts = []metric.Float64GaugeOption{
	metric.WithDescription("Duration of the last completed collection cycle."),
	metric.WithUnit("s"),
}

// NewCollectDuration returns a new CollectDuration instrument.
func NewCollectDuration(
	m metric.Meter,
	opt ...metric.Float64GaugeOption,
) (CollectDuration, error) {
	// Check if the meter is nil.
	if m == nil {
		return CollectDuration{noop.Float64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newCollectDurationOpts
	} else {
		opt = append(opt, newCollectDurationOpts...)
	}

	i, err := m.Float64Gauge(
		"webitel.dcctl.collect.duration",
		opt...,
	)
	if err != nil {
		return CollectDuration{noop.Float64Gauge{}}, err
	}
	return CollectDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CollectDuration) Inst() metric.Float64Gauge {
	return m.Float64Gauge
}

// Name returns the semantic convention name of the instrument.
func (CollectDuration) Name() string {
	return "webitel.dcctl.collect.duration"
}

// Unit returns the semantic convention unit of the instrument
func (CollectDuration) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (CollectDuration) Description() string {
	return "Duration of the last completed collection cycle."
}

// Record records val to the current distribution for attrs.
//
// This metric MUST NOT be reported before the first cycle completes. A cycle in
// progress MUST NOT update the value.
func (m CollectDuration) Record(ctx context.Context, val float64, attrs ...attribute.KeyValue) {
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
// This metric MUST NOT be reported before the first cycle completes. A cycle in
// progress MUST NOT update the value.
func (m CollectDuration) RecordSet(ctx context.Context, val float64, set attribute.Set) {
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

// CollectDurationObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.collect.duration" semantic conventions. It
// represents the duration of the last completed collection cycle.
type CollectDurationObservable struct {
	metric.Float64ObservableGauge
}

var newCollectDurationObservableOpts = []metric.Float64ObservableGaugeOption{
	metric.WithDescription("Duration of the last completed collection cycle."),
	metric.WithUnit("s"),
}

// NewCollectDurationObservable returns a new CollectDurationObservable
// instrument.
func NewCollectDurationObservable(
	m metric.Meter,
	opt ...metric.Float64ObservableGaugeOption,
) (CollectDurationObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return CollectDurationObservable{noop.Float64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newCollectDurationObservableOpts
	} else {
		opt = append(opt, newCollectDurationObservableOpts...)
	}

	i, err := m.Float64ObservableGauge(
		"webitel.dcctl.collect.duration",
		opt...,
	)
	if err != nil {
		return CollectDurationObservable{noop.Float64ObservableGauge{}}, err
	}
	return CollectDurationObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m CollectDurationObservable) Inst() metric.Float64ObservableGauge {
	return m.Float64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (CollectDurationObservable) Name() string {
	return "webitel.dcctl.collect.duration"
}

// Unit returns the semantic convention unit of the instrument
func (CollectDurationObservable) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (CollectDurationObservable) Description() string {
	return "Duration of the last completed collection cycle."
}

// DatacenterStatus is an instrument used to record metric values conforming to
// the "webitel.dcctl.datacenter.status" semantic conventions. It represents the
// current status of the datacenter.
type DatacenterStatus struct {
	metric.Int64UpDownCounter
}

var newDatacenterStatusOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("The current status of the datacenter."),
	metric.WithUnit("1"),
}

// NewDatacenterStatus returns a new DatacenterStatus instrument.
func NewDatacenterStatus(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (DatacenterStatus, error) {
	// Check if the meter is nil.
	if m == nil {
		return DatacenterStatus{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newDatacenterStatusOpts
	} else {
		opt = append(opt, newDatacenterStatusOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.dcctl.datacenter.status",
		opt...,
	)
	if err != nil {
		return DatacenterStatus{noop.Int64UpDownCounter{}}, err
	}
	return DatacenterStatus{i}, nil
}

// Inst returns the underlying metric instrument.
func (m DatacenterStatus) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (DatacenterStatus) Name() string {
	return "webitel.dcctl.datacenter.status"
}

// Unit returns the semantic convention unit of the instrument
func (DatacenterStatus) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (DatacenterStatus) Description() string {
	return "The current status of the datacenter."
}

// Add adds incr to the existing count for attrs.
//
// The webitelDatacenterName is the the name of the observed datacenter.
//
// The datacenterState is the the state of the datacenter.
//
// A timeseries is produced for every possible value of
// `webitel.dcctl.datacenter.state`. The value of this metric is 1 for the
// current state of the datacenter, and 0 for the other states.
func (m DatacenterStatus) Add(
	ctx context.Context,
	incr int64,
	webitelDatacenterName string,
	datacenterState DatacenterStateAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.datacenter.name", webitelDatacenterName),
			attribute.String("webitel.dcctl.datacenter.state", string(datacenterState)),
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
				attribute.String("webitel.datacenter.name", webitelDatacenterName),
				attribute.String("webitel.dcctl.datacenter.state", string(datacenterState)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// A timeseries is produced for every possible value of
// `webitel.dcctl.datacenter.state`. The value of this metric is 1 for the
// current state of the datacenter, and 0 for the other states.
func (m DatacenterStatus) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// DatacenterStatusObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.datacenter.status" semantic conventions. It
// represents the current status of the datacenter.
type DatacenterStatusObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newDatacenterStatusObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("The current status of the datacenter."),
	metric.WithUnit("1"),
}

// NewDatacenterStatusObservable returns a new DatacenterStatusObservable
// instrument.
func NewDatacenterStatusObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (DatacenterStatusObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return DatacenterStatusObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newDatacenterStatusObservableOpts
	} else {
		opt = append(opt, newDatacenterStatusObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.dcctl.datacenter.status",
		opt...,
	)
	if err != nil {
		return DatacenterStatusObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return DatacenterStatusObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m DatacenterStatusObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (DatacenterStatusObservable) Name() string {
	return "webitel.dcctl.datacenter.status"
}

// Unit returns the semantic convention unit of the instrument
func (DatacenterStatusObservable) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (DatacenterStatusObservable) Description() string {
	return "The current status of the datacenter."
}

// AttrWebitelDatacenterName returns a required attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (DatacenterStatusObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// AttrDatacenterState returns a required attribute for the
// "webitel.dcctl.datacenter.state" semantic convention. It represents the state
// of the datacenter.
func (DatacenterStatusObservable) AttrDatacenterState(val DatacenterStateAttr) attribute.KeyValue {
	return attribute.String("webitel.dcctl.datacenter.state", string(val))
}

// FencingStops is an instrument used to record metric values conforming to the
// "webitel.dcctl.fencing.stops" semantic conventions. It represents the number
// of fencing stops of the datacenter of the instance.
type FencingStops struct {
	metric.Int64Counter
}

var newFencingStopsOpts = []metric.Int64CounterOption{
	metric.WithDescription("Number of fencing stops of the datacenter of the instance."),
	metric.WithUnit("{stop}"),
}

// NewFencingStops returns a new FencingStops instrument.
func NewFencingStops(
	m metric.Meter,
	opt ...metric.Int64CounterOption,
) (FencingStops, error) {
	// Check if the meter is nil.
	if m == nil {
		return FencingStops{noop.Int64Counter{}}, nil
	}

	if len(opt) == 0 {
		opt = newFencingStopsOpts
	} else {
		opt = append(opt, newFencingStopsOpts...)
	}

	i, err := m.Int64Counter(
		"webitel.dcctl.fencing.stops",
		opt...,
	)
	if err != nil {
		return FencingStops{noop.Int64Counter{}}, err
	}
	return FencingStops{i}, nil
}

// Inst returns the underlying metric instrument.
func (m FencingStops) Inst() metric.Int64Counter {
	return m.Int64Counter
}

// Name returns the semantic convention name of the instrument.
func (FencingStops) Name() string {
	return "webitel.dcctl.fencing.stops"
}

// Unit returns the semantic convention unit of the instrument
func (FencingStops) Unit() string {
	return "{stop}"
}

// Description returns the semantic convention description of the instrument
func (FencingStops) Description() string {
	return "Number of fencing stops of the datacenter of the instance."
}

// Add adds incr to the existing count for attrs.
//
// The webitelDatacenterName is the the name of the datacenter of the instance.
//
// All additional attrs passed are included in the recorded value.
//
// This metric MUST be recorded once per stop, when the stop completes or fails.
func (m FencingStops) Add(
	ctx context.Context,
	incr int64,
	webitelDatacenterName string,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Counter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Counter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.datacenter.name", webitelDatacenterName),
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
				attribute.String("webitel.datacenter.name", webitelDatacenterName),
			)...,
		),
	)

	m.Int64Counter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// This metric MUST be recorded once per stop, when the stop completes or fails.
func (m FencingStops) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// AttrErrorType returns an optional attribute for the "error.type" semantic
// convention. It represents the describes a class of error the operation ended
// with.
func (FencingStops) AttrErrorType(val ErrorTypeAttr) attribute.KeyValue {
	return attribute.String("error.type", string(val))
}

// FencingStopsObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.fencing.stops" semantic conventions. It
// represents the number of fencing stops of the datacenter of the instance.
type FencingStopsObservable struct {
	metric.Int64ObservableCounter
}

var newFencingStopsObservableOpts = []metric.Int64ObservableCounterOption{
	metric.WithDescription("Number of fencing stops of the datacenter of the instance."),
	metric.WithUnit("{stop}"),
}

// NewFencingStopsObservable returns a new FencingStopsObservable instrument.
func NewFencingStopsObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableCounterOption,
) (FencingStopsObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return FencingStopsObservable{noop.Int64ObservableCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newFencingStopsObservableOpts
	} else {
		opt = append(opt, newFencingStopsObservableOpts...)
	}

	i, err := m.Int64ObservableCounter(
		"webitel.dcctl.fencing.stops",
		opt...,
	)
	if err != nil {
		return FencingStopsObservable{noop.Int64ObservableCounter{}}, err
	}
	return FencingStopsObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m FencingStopsObservable) Inst() metric.Int64ObservableCounter {
	return m.Int64ObservableCounter
}

// Name returns the semantic convention name of the instrument.
func (FencingStopsObservable) Name() string {
	return "webitel.dcctl.fencing.stops"
}

// Unit returns the semantic convention unit of the instrument
func (FencingStopsObservable) Unit() string {
	return "{stop}"
}

// Description returns the semantic convention description of the instrument
func (FencingStopsObservable) Description() string {
	return "Number of fencing stops of the datacenter of the instance."
}

// AttrWebitelDatacenterName returns a required attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// datacenter of the instance.
func (FencingStopsObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// AttrErrorType returns an optional attribute for the "error.type" semantic
// convention. It represents the describes a class of error the operation ended
// with.
func (FencingStopsObservable) AttrErrorType(val ErrorTypeAttr) attribute.KeyValue {
	return attribute.String("error.type", string(val))
}

// Generation is an instrument used to record metric values conforming to the
// "webitel.dcctl.generation" semantic conventions. It represents the effective
// generation of the coordination state.
type Generation struct {
	metric.Int64Gauge
}

var newGenerationOpts = []metric.Int64GaugeOption{
	metric.WithDescription("The effective generation of the coordination state."),
	metric.WithUnit("{generation}"),
}

// NewGeneration returns a new Generation instrument.
func NewGeneration(
	m metric.Meter,
	opt ...metric.Int64GaugeOption,
) (Generation, error) {
	// Check if the meter is nil.
	if m == nil {
		return Generation{noop.Int64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newGenerationOpts
	} else {
		opt = append(opt, newGenerationOpts...)
	}

	i, err := m.Int64Gauge(
		"webitel.dcctl.generation",
		opt...,
	)
	if err != nil {
		return Generation{noop.Int64Gauge{}}, err
	}
	return Generation{i}, nil
}

// Inst returns the underlying metric instrument.
func (m Generation) Inst() metric.Int64Gauge {
	return m.Int64Gauge
}

// Name returns the semantic convention name of the instrument.
func (Generation) Name() string {
	return "webitel.dcctl.generation"
}

// Unit returns the semantic convention unit of the instrument
func (Generation) Unit() string {
	return "{generation}"
}

// Description returns the semantic convention description of the instrument
func (Generation) Description() string {
	return "The effective generation of the coordination state."
}

// Record records val to the current distribution for attrs.
func (m Generation) Record(ctx context.Context, val int64, attrs ...attribute.KeyValue) {
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
func (m Generation) RecordSet(ctx context.Context, val int64, set attribute.Set) {
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

// GenerationObservable is an instrument used to record metric values conforming
// to the "webitel.dcctl.generation" semantic conventions. It represents the
// effective generation of the coordination state.
type GenerationObservable struct {
	metric.Int64ObservableGauge
}

var newGenerationObservableOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("The effective generation of the coordination state."),
	metric.WithUnit("{generation}"),
}

// NewGenerationObservable returns a new GenerationObservable instrument.
func NewGenerationObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (GenerationObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return GenerationObservable{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newGenerationObservableOpts
	} else {
		opt = append(opt, newGenerationObservableOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.dcctl.generation",
		opt...,
	)
	if err != nil {
		return GenerationObservable{noop.Int64ObservableGauge{}}, err
	}
	return GenerationObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m GenerationObservable) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (GenerationObservable) Name() string {
	return "webitel.dcctl.generation"
}

// Unit returns the semantic convention unit of the instrument
func (GenerationObservable) Unit() string {
	return "{generation}"
}

// Description returns the semantic convention description of the instrument
func (GenerationObservable) Description() string {
	return "The effective generation of the coordination state."
}

// JobCount is an instrument used to record metric values conforming to the
// "webitel.dcctl.job.count" semantic conventions. It represents the number of
// managed jobs that are currently in the state described by the
// `webitel.dcctl.job.state` attribute.
type JobCount struct {
	metric.Int64UpDownCounter
}

var newJobCountOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("Number of managed jobs that are currently in the state described by the `webitel.dcctl.job.state` attribute."),
	metric.WithUnit("{job}"),
}

// NewJobCount returns a new JobCount instrument.
func NewJobCount(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (JobCount, error) {
	// Check if the meter is nil.
	if m == nil {
		return JobCount{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newJobCountOpts
	} else {
		opt = append(opt, newJobCountOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.dcctl.job.count",
		opt...,
	)
	if err != nil {
		return JobCount{noop.Int64UpDownCounter{}}, err
	}
	return JobCount{i}, nil
}

// Inst returns the underlying metric instrument.
func (m JobCount) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (JobCount) Name() string {
	return "webitel.dcctl.job.count"
}

// Unit returns the semantic convention unit of the instrument
func (JobCount) Unit() string {
	return "{job}"
}

// Description returns the semantic convention description of the instrument
func (JobCount) Description() string {
	return "Number of managed jobs that are currently in the state described by the `webitel.dcctl.job.state` attribute."
}

// Add adds incr to the existing count for attrs.
//
// The webitelDatacenterName is the the name of the observed datacenter.
//
// The jobStage is the the start stage of a job managed by dcctl.
//
// The jobState is the the state of a job managed by dcctl.
func (m JobCount) Add(
	ctx context.Context,
	incr int64,
	webitelDatacenterName string,
	jobStage int,
	jobState JobStateAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.datacenter.name", webitelDatacenterName),
			attribute.Int("webitel.dcctl.job.stage", jobStage),
			attribute.String("webitel.dcctl.job.state", string(jobState)),
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
				attribute.String("webitel.datacenter.name", webitelDatacenterName),
				attribute.Int("webitel.dcctl.job.stage", jobStage),
				attribute.String("webitel.dcctl.job.state", string(jobState)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
func (m JobCount) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// JobCountObservable is an instrument used to record metric values conforming to
// the "webitel.dcctl.job.count" semantic conventions. It represents the number
// of managed jobs that are currently in the state described by the
// `webitel.dcctl.job.state` attribute.
type JobCountObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newJobCountObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("Number of managed jobs that are currently in the state described by the `webitel.dcctl.job.state` attribute."),
	metric.WithUnit("{job}"),
}

// NewJobCountObservable returns a new JobCountObservable instrument.
func NewJobCountObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (JobCountObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return JobCountObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newJobCountObservableOpts
	} else {
		opt = append(opt, newJobCountObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.dcctl.job.count",
		opt...,
	)
	if err != nil {
		return JobCountObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return JobCountObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m JobCountObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (JobCountObservable) Name() string {
	return "webitel.dcctl.job.count"
}

// Unit returns the semantic convention unit of the instrument
func (JobCountObservable) Unit() string {
	return "{job}"
}

// Description returns the semantic convention description of the instrument
func (JobCountObservable) Description() string {
	return "Number of managed jobs that are currently in the state described by the `webitel.dcctl.job.state` attribute."
}

// AttrWebitelDatacenterName returns a required attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (JobCountObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// AttrJobStage returns a required attribute for the "webitel.dcctl.job.stage"
// semantic convention. It represents the start stage of a job managed by dcctl.
func (JobCountObservable) AttrJobStage(val int) attribute.KeyValue {
	return attribute.Int("webitel.dcctl.job.stage", val)
}

// AttrJobState returns a required attribute for the "webitel.dcctl.job.state"
// semantic convention. It represents the state of a job managed by dcctl.
func (JobCountObservable) AttrJobState(val JobStateAttr) attribute.KeyValue {
	return attribute.String("webitel.dcctl.job.state", string(val))
}

// LeaseAge is an instrument used to record metric values conforming to the
// "webitel.dcctl.lease.age" semantic conventions. It represents the time since
// the counter of the lease last changed.
type LeaseAge struct {
	metric.Float64Gauge
}

var newLeaseAgeOpts = []metric.Float64GaugeOption{
	metric.WithDescription("Time since the counter of the lease last changed."),
	metric.WithUnit("s"),
}

// NewLeaseAge returns a new LeaseAge instrument.
func NewLeaseAge(
	m metric.Meter,
	opt ...metric.Float64GaugeOption,
) (LeaseAge, error) {
	// Check if the meter is nil.
	if m == nil {
		return LeaseAge{noop.Float64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newLeaseAgeOpts
	} else {
		opt = append(opt, newLeaseAgeOpts...)
	}

	i, err := m.Float64Gauge(
		"webitel.dcctl.lease.age",
		opt...,
	)
	if err != nil {
		return LeaseAge{noop.Float64Gauge{}}, err
	}
	return LeaseAge{i}, nil
}

// Inst returns the underlying metric instrument.
func (m LeaseAge) Inst() metric.Float64Gauge {
	return m.Float64Gauge
}

// Name returns the semantic convention name of the instrument.
func (LeaseAge) Name() string {
	return "webitel.dcctl.lease.age"
}

// Unit returns the semantic convention unit of the instrument
func (LeaseAge) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (LeaseAge) Description() string {
	return "Time since the counter of the lease last changed."
}

// Record records val to the current distribution for attrs.
//
// The webitelDatacenterName is the the name of the datacenter that holds the
// lease.
//
// The time MUST be measured with the monotonic clock of the instance that
// reports the metric.
func (m LeaseAge) Record(
	ctx context.Context,
	val float64,
	webitelDatacenterName string,
	attrs ...attribute.KeyValue,
) {
	if !m.Float64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Float64Gauge.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.datacenter.name", webitelDatacenterName),
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
				attribute.String("webitel.datacenter.name", webitelDatacenterName),
			)...,
		),
	)

	m.Float64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// The time MUST be measured with the monotonic clock of the instance that
// reports the metric.
func (m LeaseAge) RecordSet(ctx context.Context, val float64, set attribute.Set) {
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

// LeaseAgeObservable is an instrument used to record metric values conforming to
// the "webitel.dcctl.lease.age" semantic conventions. It represents the time
// since the counter of the lease last changed.
type LeaseAgeObservable struct {
	metric.Float64ObservableGauge
}

var newLeaseAgeObservableOpts = []metric.Float64ObservableGaugeOption{
	metric.WithDescription("Time since the counter of the lease last changed."),
	metric.WithUnit("s"),
}

// NewLeaseAgeObservable returns a new LeaseAgeObservable instrument.
func NewLeaseAgeObservable(
	m metric.Meter,
	opt ...metric.Float64ObservableGaugeOption,
) (LeaseAgeObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return LeaseAgeObservable{noop.Float64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newLeaseAgeObservableOpts
	} else {
		opt = append(opt, newLeaseAgeObservableOpts...)
	}

	i, err := m.Float64ObservableGauge(
		"webitel.dcctl.lease.age",
		opt...,
	)
	if err != nil {
		return LeaseAgeObservable{noop.Float64ObservableGauge{}}, err
	}
	return LeaseAgeObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m LeaseAgeObservable) Inst() metric.Float64ObservableGauge {
	return m.Float64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (LeaseAgeObservable) Name() string {
	return "webitel.dcctl.lease.age"
}

// Unit returns the semantic convention unit of the instrument
func (LeaseAgeObservable) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (LeaseAgeObservable) Description() string {
	return "Time since the counter of the lease last changed."
}

// AttrWebitelDatacenterName returns a required attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// datacenter that holds the lease.
func (LeaseAgeObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// OperationDuration is an instrument used to record metric values conforming to
// the "webitel.dcctl.operation.duration" semantic conventions. It represents the
// duration of operations, from creation to the end state.
type OperationDuration struct {
	metric.Float64Histogram
}

var newOperationDurationOpts = []metric.Float64HistogramOption{
	metric.WithDescription("Duration of operations, from creation to the end state."),
	metric.WithUnit("s"),
}

// NewOperationDuration returns a new OperationDuration instrument.
func NewOperationDuration(
	m metric.Meter,
	opt ...metric.Float64HistogramOption,
) (OperationDuration, error) {
	// Check if the meter is nil.
	if m == nil {
		return OperationDuration{noop.Float64Histogram{}}, nil
	}

	if len(opt) == 0 {
		opt = newOperationDurationOpts
	} else {
		opt = append(opt, newOperationDurationOpts...)
	}

	i, err := m.Float64Histogram(
		"webitel.dcctl.operation.duration",
		opt...,
	)
	if err != nil {
		return OperationDuration{noop.Float64Histogram{}}, err
	}
	return OperationDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m OperationDuration) Inst() metric.Float64Histogram {
	return m.Float64Histogram
}

// Name returns the semantic convention name of the instrument.
func (OperationDuration) Name() string {
	return "webitel.dcctl.operation.duration"
}

// Unit returns the semantic convention unit of the instrument
func (OperationDuration) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (OperationDuration) Description() string {
	return "Duration of operations, from creation to the end state."
}

// Record records val to the current distribution for attrs.
//
// The operationMode is the the mode in which the operation changes the
// coordination state.
//
// The operationState is the the state of the operation.
//
// The operationType is the the type of the operation.
//
// This metric MUST be recorded once per operation, by the instance that leads
// it, when the operation reaches `succeeded`, `failed` or `abandoned`.
func (m OperationDuration) Record(
	ctx context.Context,
	val float64,
	operationMode OperationModeAttr,
	operationState OperationStateAttr,
	operationType OperationTypeAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Float64Histogram.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Float64Histogram.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.dcctl.operation.mode", string(operationMode)),
			attribute.String("webitel.dcctl.operation.state", string(operationState)),
			attribute.String("webitel.dcctl.operation.type", string(operationType)),
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
				attribute.String("webitel.dcctl.operation.mode", string(operationMode)),
				attribute.String("webitel.dcctl.operation.state", string(operationState)),
				attribute.String("webitel.dcctl.operation.type", string(operationType)),
			)...,
		),
	)

	m.Float64Histogram.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// This metric MUST be recorded once per operation, by the instance that leads
// it, when the operation reaches `succeeded`, `failed` or `abandoned`.
func (m OperationDuration) RecordSet(ctx context.Context, val float64, set attribute.Set) {
	if !m.Float64Histogram.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Float64Histogram.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Float64Histogram.Record(ctx, val, *o...)
}

// PatroniDivergence is an instrument used to record metric values conforming to
// the "webitel.dcctl.patroni.divergence" semantic conventions. It represents the
// size of the data of the datacenter that diverged from the active datacenter.
type PatroniDivergence struct {
	metric.Int64Gauge
}

var newPatroniDivergenceOpts = []metric.Int64GaugeOption{
	metric.WithDescription("The size of the data of the datacenter that diverged from the active datacenter."),
	metric.WithUnit("By"),
}

// NewPatroniDivergence returns a new PatroniDivergence instrument.
func NewPatroniDivergence(
	m metric.Meter,
	opt ...metric.Int64GaugeOption,
) (PatroniDivergence, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniDivergence{noop.Int64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniDivergenceOpts
	} else {
		opt = append(opt, newPatroniDivergenceOpts...)
	}

	i, err := m.Int64Gauge(
		"webitel.dcctl.patroni.divergence",
		opt...,
	)
	if err != nil {
		return PatroniDivergence{noop.Int64Gauge{}}, err
	}
	return PatroniDivergence{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniDivergence) Inst() metric.Int64Gauge {
	return m.Int64Gauge
}

// Name returns the semantic convention name of the instrument.
func (PatroniDivergence) Name() string {
	return "webitel.dcctl.patroni.divergence"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniDivergence) Unit() string {
	return "By"
}

// Description returns the semantic convention description of the instrument
func (PatroniDivergence) Description() string {
	return "The size of the data of the datacenter that diverged from the active datacenter."
}

// Record records val to the current distribution for attrs.
//
// The webitelDatacenterName is the the name of the observed datacenter.
//
// The value is the distance between the WAL position of the datacenter and the
// point where its timeline diverged. This metric MUST NOT be reported for a
// datacenter that did not diverge.
func (m PatroniDivergence) Record(
	ctx context.Context,
	val int64,
	webitelDatacenterName string,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Gauge.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.datacenter.name", webitelDatacenterName),
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
				attribute.String("webitel.datacenter.name", webitelDatacenterName),
			)...,
		),
	)

	m.Int64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// The value is the distance between the WAL position of the datacenter and the
// point where its timeline diverged. This metric MUST NOT be reported for a
// datacenter that did not diverge.
func (m PatroniDivergence) RecordSet(ctx context.Context, val int64, set attribute.Set) {
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

// PatroniDivergenceObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.patroni.divergence" semantic conventions. It
// represents the size of the data of the datacenter that diverged from the
// active datacenter.
type PatroniDivergenceObservable struct {
	metric.Int64ObservableGauge
}

var newPatroniDivergenceObservableOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("The size of the data of the datacenter that diverged from the active datacenter."),
	metric.WithUnit("By"),
}

// NewPatroniDivergenceObservable returns a new PatroniDivergenceObservable
// instrument.
func NewPatroniDivergenceObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (PatroniDivergenceObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniDivergenceObservable{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniDivergenceObservableOpts
	} else {
		opt = append(opt, newPatroniDivergenceObservableOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.dcctl.patroni.divergence",
		opt...,
	)
	if err != nil {
		return PatroniDivergenceObservable{noop.Int64ObservableGauge{}}, err
	}
	return PatroniDivergenceObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniDivergenceObservable) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (PatroniDivergenceObservable) Name() string {
	return "webitel.dcctl.patroni.divergence"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniDivergenceObservable) Unit() string {
	return "By"
}

// Description returns the semantic convention description of the instrument
func (PatroniDivergenceObservable) Description() string {
	return "The size of the data of the datacenter that diverged from the active datacenter."
}

// AttrWebitelDatacenterName returns a required attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (PatroniDivergenceObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// PatroniLag is an instrument used to record metric values conforming to the
// "webitel.dcctl.patroni.lag" semantic conventions. It represents the
// replication lag of the datacenter that is not active.
type PatroniLag struct {
	metric.Int64Gauge
}

var newPatroniLagOpts = []metric.Int64GaugeOption{
	metric.WithDescription("The replication lag of the datacenter that is not active."),
	metric.WithUnit("By"),
}

// NewPatroniLag returns a new PatroniLag instrument.
func NewPatroniLag(
	m metric.Meter,
	opt ...metric.Int64GaugeOption,
) (PatroniLag, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniLag{noop.Int64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniLagOpts
	} else {
		opt = append(opt, newPatroniLagOpts...)
	}

	i, err := m.Int64Gauge(
		"webitel.dcctl.patroni.lag",
		opt...,
	)
	if err != nil {
		return PatroniLag{noop.Int64Gauge{}}, err
	}
	return PatroniLag{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniLag) Inst() metric.Int64Gauge {
	return m.Int64Gauge
}

// Name returns the semantic convention name of the instrument.
func (PatroniLag) Name() string {
	return "webitel.dcctl.patroni.lag"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniLag) Unit() string {
	return "By"
}

// Description returns the semantic convention description of the instrument
func (PatroniLag) Description() string {
	return "The replication lag of the datacenter that is not active."
}

// Record records val to the current distribution for attrs.
//
// The webitelDatacenterName is the the name of the observed datacenter.
//
// The value is the distance between the WAL position of the primary of the
// active datacenter and the received WAL position of the Patroni leader of the
// datacenter. For a cluster that spans datacenters, the value is the smallest
// lag of the members of the datacenter. This metric MUST NOT be reported for the
// active datacenter.
func (m PatroniLag) Record(
	ctx context.Context,
	val int64,
	webitelDatacenterName string,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Gauge.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.datacenter.name", webitelDatacenterName),
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
				attribute.String("webitel.datacenter.name", webitelDatacenterName),
			)...,
		),
	)

	m.Int64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// The value is the distance between the WAL position of the primary of the
// active datacenter and the received WAL position of the Patroni leader of the
// datacenter. For a cluster that spans datacenters, the value is the smallest
// lag of the members of the datacenter. This metric MUST NOT be reported for the
// active datacenter.
func (m PatroniLag) RecordSet(ctx context.Context, val int64, set attribute.Set) {
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

// PatroniLagObservable is an instrument used to record metric values conforming
// to the "webitel.dcctl.patroni.lag" semantic conventions. It represents the
// replication lag of the datacenter that is not active.
type PatroniLagObservable struct {
	metric.Int64ObservableGauge
}

var newPatroniLagObservableOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("The replication lag of the datacenter that is not active."),
	metric.WithUnit("By"),
}

// NewPatroniLagObservable returns a new PatroniLagObservable instrument.
func NewPatroniLagObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (PatroniLagObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniLagObservable{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniLagObservableOpts
	} else {
		opt = append(opt, newPatroniLagObservableOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.dcctl.patroni.lag",
		opt...,
	)
	if err != nil {
		return PatroniLagObservable{noop.Int64ObservableGauge{}}, err
	}
	return PatroniLagObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniLagObservable) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (PatroniLagObservable) Name() string {
	return "webitel.dcctl.patroni.lag"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniLagObservable) Unit() string {
	return "By"
}

// Description returns the semantic convention description of the instrument
func (PatroniLagObservable) Description() string {
	return "The replication lag of the datacenter that is not active."
}

// AttrWebitelDatacenterName returns a required attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (PatroniLagObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// PatroniMemberCount is an instrument used to record metric values conforming to
// the "webitel.dcctl.patroni.member.count" semantic conventions. It represents
// the number of Patroni members of the datacenter that currently have the role
// described by the `webitel.dcctl.patroni.role` attribute.
type PatroniMemberCount struct {
	metric.Int64UpDownCounter
}

var newPatroniMemberCountOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("Number of Patroni members of the datacenter that currently have the role described by the `webitel.dcctl.patroni.role` attribute."),
	metric.WithUnit("{member}"),
}

// NewPatroniMemberCount returns a new PatroniMemberCount instrument.
func NewPatroniMemberCount(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (PatroniMemberCount, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniMemberCount{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniMemberCountOpts
	} else {
		opt = append(opt, newPatroniMemberCountOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.dcctl.patroni.member.count",
		opt...,
	)
	if err != nil {
		return PatroniMemberCount{noop.Int64UpDownCounter{}}, err
	}
	return PatroniMemberCount{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniMemberCount) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (PatroniMemberCount) Name() string {
	return "webitel.dcctl.patroni.member.count"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniMemberCount) Unit() string {
	return "{member}"
}

// Description returns the semantic convention description of the instrument
func (PatroniMemberCount) Description() string {
	return "Number of Patroni members of the datacenter that currently have the role described by the `webitel.dcctl.patroni.role` attribute."
}

// Add adds incr to the existing count for attrs.
//
// The webitelDatacenterName is the the name of the observed datacenter.
//
// The patroniRole is the the Patroni role of a cluster member.
//
// The role `unknown` counts the members that do not answer while the Patroni of
// the cluster does. Nothing is reported for a datacenter whose Patroni does not
// answer at all.
func (m PatroniMemberCount) Add(
	ctx context.Context,
	incr int64,
	webitelDatacenterName string,
	patroniRole PatroniRoleAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.datacenter.name", webitelDatacenterName),
			attribute.String("webitel.dcctl.patroni.role", string(patroniRole)),
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
				attribute.String("webitel.datacenter.name", webitelDatacenterName),
				attribute.String("webitel.dcctl.patroni.role", string(patroniRole)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// The role `unknown` counts the members that do not answer while the Patroni of
// the cluster does. Nothing is reported for a datacenter whose Patroni does not
// answer at all.
func (m PatroniMemberCount) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// PatroniMemberCountObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.patroni.member.count" semantic conventions.
// It represents the number of Patroni members of the datacenter that currently
// have the role described by the `webitel.dcctl.patroni.role` attribute.
type PatroniMemberCountObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newPatroniMemberCountObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("Number of Patroni members of the datacenter that currently have the role described by the `webitel.dcctl.patroni.role` attribute."),
	metric.WithUnit("{member}"),
}

// NewPatroniMemberCountObservable returns a new PatroniMemberCountObservable
// instrument.
func NewPatroniMemberCountObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (PatroniMemberCountObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniMemberCountObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniMemberCountObservableOpts
	} else {
		opt = append(opt, newPatroniMemberCountObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.dcctl.patroni.member.count",
		opt...,
	)
	if err != nil {
		return PatroniMemberCountObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return PatroniMemberCountObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniMemberCountObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (PatroniMemberCountObservable) Name() string {
	return "webitel.dcctl.patroni.member.count"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniMemberCountObservable) Unit() string {
	return "{member}"
}

// Description returns the semantic convention description of the instrument
func (PatroniMemberCountObservable) Description() string {
	return "Number of Patroni members of the datacenter that currently have the role described by the `webitel.dcctl.patroni.role` attribute."
}

// AttrWebitelDatacenterName returns a required attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (PatroniMemberCountObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// AttrPatroniRole returns a required attribute for the
// "webitel.dcctl.patroni.role" semantic convention. It represents the Patroni
// role of a cluster member.
func (PatroniMemberCountObservable) AttrPatroniRole(val PatroniRoleAttr) attribute.KeyValue {
	return attribute.String("webitel.dcctl.patroni.role", string(val))
}

// PatroniTimeline is an instrument used to record metric values conforming to
// the "webitel.dcctl.patroni.timeline" semantic conventions. It represents the
// timeline of the Patroni leader of the datacenter.
type PatroniTimeline struct {
	metric.Int64Gauge
}

var newPatroniTimelineOpts = []metric.Int64GaugeOption{
	metric.WithDescription("The timeline of the Patroni leader of the datacenter."),
	metric.WithUnit("{timeline}"),
}

// NewPatroniTimeline returns a new PatroniTimeline instrument.
func NewPatroniTimeline(
	m metric.Meter,
	opt ...metric.Int64GaugeOption,
) (PatroniTimeline, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniTimeline{noop.Int64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniTimelineOpts
	} else {
		opt = append(opt, newPatroniTimelineOpts...)
	}

	i, err := m.Int64Gauge(
		"webitel.dcctl.patroni.timeline",
		opt...,
	)
	if err != nil {
		return PatroniTimeline{noop.Int64Gauge{}}, err
	}
	return PatroniTimeline{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniTimeline) Inst() metric.Int64Gauge {
	return m.Int64Gauge
}

// Name returns the semantic convention name of the instrument.
func (PatroniTimeline) Name() string {
	return "webitel.dcctl.patroni.timeline"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniTimeline) Unit() string {
	return "{timeline}"
}

// Description returns the semantic convention description of the instrument
func (PatroniTimeline) Description() string {
	return "The timeline of the Patroni leader of the datacenter."
}

// Record records val to the current distribution for attrs.
//
// The webitelDatacenterName is the the name of the observed datacenter.
//
// For a cluster that spans datacenters, the value is the largest timeline of the
// members of the datacenter.
func (m PatroniTimeline) Record(
	ctx context.Context,
	val int64,
	webitelDatacenterName string,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Gauge.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.datacenter.name", webitelDatacenterName),
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
				attribute.String("webitel.datacenter.name", webitelDatacenterName),
			)...,
		),
	)

	m.Int64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// For a cluster that spans datacenters, the value is the largest timeline of the
// members of the datacenter.
func (m PatroniTimeline) RecordSet(ctx context.Context, val int64, set attribute.Set) {
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

// PatroniTimelineObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.patroni.timeline" semantic conventions. It
// represents the timeline of the Patroni leader of the datacenter.
type PatroniTimelineObservable struct {
	metric.Int64ObservableGauge
}

var newPatroniTimelineObservableOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("The timeline of the Patroni leader of the datacenter."),
	metric.WithUnit("{timeline}"),
}

// NewPatroniTimelineObservable returns a new PatroniTimelineObservable
// instrument.
func NewPatroniTimelineObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (PatroniTimelineObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniTimelineObservable{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniTimelineObservableOpts
	} else {
		opt = append(opt, newPatroniTimelineObservableOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.dcctl.patroni.timeline",
		opt...,
	)
	if err != nil {
		return PatroniTimelineObservable{noop.Int64ObservableGauge{}}, err
	}
	return PatroniTimelineObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniTimelineObservable) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (PatroniTimelineObservable) Name() string {
	return "webitel.dcctl.patroni.timeline"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniTimelineObservable) Unit() string {
	return "{timeline}"
}

// Description returns the semantic convention description of the instrument
func (PatroniTimelineObservable) Description() string {
	return "The timeline of the Patroni leader of the datacenter."
}

// AttrWebitelDatacenterName returns a required attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (PatroniTimelineObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// PatroniWalPosition is an instrument used to record metric values conforming to
// the "webitel.dcctl.patroni.wal.position" semantic conventions. It represents
// the WAL position of the Patroni leader of the datacenter.
type PatroniWalPosition struct {
	metric.Int64Gauge
}

var newPatroniWalPositionOpts = []metric.Int64GaugeOption{
	metric.WithDescription("The WAL position of the Patroni leader of the datacenter."),
	metric.WithUnit("By"),
}

// NewPatroniWalPosition returns a new PatroniWalPosition instrument.
func NewPatroniWalPosition(
	m metric.Meter,
	opt ...metric.Int64GaugeOption,
) (PatroniWalPosition, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniWalPosition{noop.Int64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniWalPositionOpts
	} else {
		opt = append(opt, newPatroniWalPositionOpts...)
	}

	i, err := m.Int64Gauge(
		"webitel.dcctl.patroni.wal.position",
		opt...,
	)
	if err != nil {
		return PatroniWalPosition{noop.Int64Gauge{}}, err
	}
	return PatroniWalPosition{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniWalPosition) Inst() metric.Int64Gauge {
	return m.Int64Gauge
}

// Name returns the semantic convention name of the instrument.
func (PatroniWalPosition) Name() string {
	return "webitel.dcctl.patroni.wal.position"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniWalPosition) Unit() string {
	return "By"
}

// Description returns the semantic convention description of the instrument
func (PatroniWalPosition) Description() string {
	return "The WAL position of the Patroni leader of the datacenter."
}

// Record records val to the current distribution for attrs.
//
// The webitelDatacenterName is the the name of the observed datacenter.
//
// The value is the LSN as a byte offset: the current WAL location of a primary,
// and the received WAL location of a standby leader.
func (m PatroniWalPosition) Record(
	ctx context.Context,
	val int64,
	webitelDatacenterName string,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Gauge.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.datacenter.name", webitelDatacenterName),
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
				attribute.String("webitel.datacenter.name", webitelDatacenterName),
			)...,
		),
	)

	m.Int64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// The value is the LSN as a byte offset: the current WAL location of a primary,
// and the received WAL location of a standby leader.
func (m PatroniWalPosition) RecordSet(ctx context.Context, val int64, set attribute.Set) {
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

// PatroniWalPositionObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.patroni.wal.position" semantic conventions.
// It represents the WAL position of the Patroni leader of the datacenter.
type PatroniWalPositionObservable struct {
	metric.Int64ObservableGauge
}

var newPatroniWalPositionObservableOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("The WAL position of the Patroni leader of the datacenter."),
	metric.WithUnit("By"),
}

// NewPatroniWalPositionObservable returns a new PatroniWalPositionObservable
// instrument.
func NewPatroniWalPositionObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (PatroniWalPositionObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return PatroniWalPositionObservable{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newPatroniWalPositionObservableOpts
	} else {
		opt = append(opt, newPatroniWalPositionObservableOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.dcctl.patroni.wal.position",
		opt...,
	)
	if err != nil {
		return PatroniWalPositionObservable{noop.Int64ObservableGauge{}}, err
	}
	return PatroniWalPositionObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PatroniWalPositionObservable) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (PatroniWalPositionObservable) Name() string {
	return "webitel.dcctl.patroni.wal.position"
}

// Unit returns the semantic convention unit of the instrument
func (PatroniWalPositionObservable) Unit() string {
	return "By"
}

// Description returns the semantic convention description of the instrument
func (PatroniWalPositionObservable) Description() string {
	return "The WAL position of the Patroni leader of the datacenter."
}

// AttrWebitelDatacenterName returns a required attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (PatroniWalPositionObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// QuorumStatus is an instrument used to record metric values conforming to the
// "webitel.dcctl.quorum.status" semantic conventions. It represents the current
// status of the quorum.
type QuorumStatus struct {
	metric.Int64UpDownCounter
}

var newQuorumStatusOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("The current status of the quorum."),
	metric.WithUnit("1"),
}

// NewQuorumStatus returns a new QuorumStatus instrument.
func NewQuorumStatus(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (QuorumStatus, error) {
	// Check if the meter is nil.
	if m == nil {
		return QuorumStatus{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newQuorumStatusOpts
	} else {
		opt = append(opt, newQuorumStatusOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.dcctl.quorum.status",
		opt...,
	)
	if err != nil {
		return QuorumStatus{noop.Int64UpDownCounter{}}, err
	}
	return QuorumStatus{i}, nil
}

// Inst returns the underlying metric instrument.
func (m QuorumStatus) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (QuorumStatus) Name() string {
	return "webitel.dcctl.quorum.status"
}

// Unit returns the semantic convention unit of the instrument
func (QuorumStatus) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (QuorumStatus) Description() string {
	return "The current status of the quorum."
}

// Add adds incr to the existing count for attrs.
//
// The quorumMode is the the mode in which the coordination state can be changed.
//
// A timeseries is produced for every possible value of
// `webitel.dcctl.quorum.mode`. The value of this metric is 1 for the current
// mode, and 0 for the other modes.
func (m QuorumStatus) Add(
	ctx context.Context,
	incr int64,
	quorumMode QuorumModeAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.dcctl.quorum.mode", string(quorumMode)),
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
				attribute.String("webitel.dcctl.quorum.mode", string(quorumMode)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// A timeseries is produced for every possible value of
// `webitel.dcctl.quorum.mode`. The value of this metric is 1 for the current
// mode, and 0 for the other modes.
func (m QuorumStatus) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// QuorumStatusObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.quorum.status" semantic conventions. It
// represents the current status of the quorum.
type QuorumStatusObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newQuorumStatusObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("The current status of the quorum."),
	metric.WithUnit("1"),
}

// NewQuorumStatusObservable returns a new QuorumStatusObservable instrument.
func NewQuorumStatusObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (QuorumStatusObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return QuorumStatusObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newQuorumStatusObservableOpts
	} else {
		opt = append(opt, newQuorumStatusObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.dcctl.quorum.status",
		opt...,
	)
	if err != nil {
		return QuorumStatusObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return QuorumStatusObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m QuorumStatusObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (QuorumStatusObservable) Name() string {
	return "webitel.dcctl.quorum.status"
}

// Unit returns the semantic convention unit of the instrument
func (QuorumStatusObservable) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (QuorumStatusObservable) Description() string {
	return "The current status of the quorum."
}

// AttrQuorumMode returns a required attribute for the
// "webitel.dcctl.quorum.mode" semantic convention. It represents the mode in
// which the coordination state can be changed.
func (QuorumStatusObservable) AttrQuorumMode(val QuorumModeAttr) attribute.KeyValue {
	return attribute.String("webitel.dcctl.quorum.mode", string(val))
}

// SourceStatus is an instrument used to record metric values conforming to the
// "webitel.dcctl.source.status" semantic conventions. It represents the current
// status of the source.
type SourceStatus struct {
	metric.Int64UpDownCounter
}

var newSourceStatusOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("The current status of the source."),
	metric.WithUnit("1"),
}

// NewSourceStatus returns a new SourceStatus instrument.
func NewSourceStatus(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (SourceStatus, error) {
	// Check if the meter is nil.
	if m == nil {
		return SourceStatus{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newSourceStatusOpts
	} else {
		opt = append(opt, newSourceStatusOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.dcctl.source.status",
		opt...,
	)
	if err != nil {
		return SourceStatus{noop.Int64UpDownCounter{}}, err
	}
	return SourceStatus{i}, nil
}

// Inst returns the underlying metric instrument.
func (m SourceStatus) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (SourceStatus) Name() string {
	return "webitel.dcctl.source.status"
}

// Unit returns the semantic convention unit of the instrument
func (SourceStatus) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (SourceStatus) Description() string {
	return "The current status of the source."
}

// Add adds incr to the existing count for attrs.
//
// The sourceState is the the state of a source of dcctl.
//
// The sourceSystem is the the system dcctl reads the state from.
//
// All additional attrs passed are included in the recorded value.
//
// A timeseries is produced for every possible value of
// `webitel.dcctl.source.state`. The value of this metric is 1 for the current
// state of the source, and 0 for the other states.
func (m SourceStatus) Add(
	ctx context.Context,
	incr int64,
	sourceState SourceStateAttr,
	sourceSystem SourceSystemAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.dcctl.source.state", string(sourceState)),
			attribute.String("webitel.dcctl.source.system", string(sourceSystem)),
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
				attribute.String("webitel.dcctl.source.state", string(sourceState)),
				attribute.String("webitel.dcctl.source.system", string(sourceSystem)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// A timeseries is produced for every possible value of
// `webitel.dcctl.source.state`. The value of this metric is 1 for the current
// state of the source, and 0 for the other states.
func (m SourceStatus) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// AttrWebitelDatacenterName returns an optional attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (SourceStatus) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// SourceStatusObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.source.status" semantic conventions. It
// represents the current status of the source.
type SourceStatusObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newSourceStatusObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("The current status of the source."),
	metric.WithUnit("1"),
}

// NewSourceStatusObservable returns a new SourceStatusObservable instrument.
func NewSourceStatusObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (SourceStatusObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return SourceStatusObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newSourceStatusObservableOpts
	} else {
		opt = append(opt, newSourceStatusObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.dcctl.source.status",
		opt...,
	)
	if err != nil {
		return SourceStatusObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return SourceStatusObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m SourceStatusObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (SourceStatusObservable) Name() string {
	return "webitel.dcctl.source.status"
}

// Unit returns the semantic convention unit of the instrument
func (SourceStatusObservable) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (SourceStatusObservable) Description() string {
	return "The current status of the source."
}

// AttrSourceState returns a required attribute for the
// "webitel.dcctl.source.state" semantic convention. It represents the state of a
// source of dcctl.
func (SourceStatusObservable) AttrSourceState(val SourceStateAttr) attribute.KeyValue {
	return attribute.String("webitel.dcctl.source.state", string(val))
}

// AttrSourceSystem returns a required attribute for the
// "webitel.dcctl.source.system" semantic convention. It represents the system
// dcctl reads the state from.
func (SourceStatusObservable) AttrSourceSystem(val SourceSystemAttr) attribute.KeyValue {
	return attribute.String("webitel.dcctl.source.system", string(val))
}

// AttrWebitelDatacenterName returns an optional attribute for the
// "webitel.datacenter.name" semantic convention. It represents the name of the
// observed datacenter.
func (SourceStatusObservable) AttrWebitelDatacenterName(val string) attribute.KeyValue {
	return attribute.String("webitel.datacenter.name", val)
}

// StepDuration is an instrument used to record metric values conforming to the
// "webitel.dcctl.step.duration" semantic conventions. It represents the duration
// of the steps of operations.
type StepDuration struct {
	metric.Float64Histogram
}

var newStepDurationOpts = []metric.Float64HistogramOption{
	metric.WithDescription("Duration of the steps of operations."),
	metric.WithUnit("s"),
}

// NewStepDuration returns a new StepDuration instrument.
func NewStepDuration(
	m metric.Meter,
	opt ...metric.Float64HistogramOption,
) (StepDuration, error) {
	// Check if the meter is nil.
	if m == nil {
		return StepDuration{noop.Float64Histogram{}}, nil
	}

	if len(opt) == 0 {
		opt = newStepDurationOpts
	} else {
		opt = append(opt, newStepDurationOpts...)
	}

	i, err := m.Float64Histogram(
		"webitel.dcctl.step.duration",
		opt...,
	)
	if err != nil {
		return StepDuration{noop.Float64Histogram{}}, err
	}
	return StepDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m StepDuration) Inst() metric.Float64Histogram {
	return m.Float64Histogram
}

// Name returns the semantic convention name of the instrument.
func (StepDuration) Name() string {
	return "webitel.dcctl.step.duration"
}

// Unit returns the semantic convention unit of the instrument
func (StepDuration) Unit() string {
	return "s"
}

// Description returns the semantic convention description of the instrument
func (StepDuration) Description() string {
	return "Duration of the steps of operations."
}

// Record records val to the current distribution for attrs.
//
// The operationType is the the type of the operation.
//
// The stepState is the the state of the step of an operation.
//
// The stepType is the the type of the step of an operation.
//
// All additional attrs passed are included in the recorded value.
//
// This metric MUST be recorded once per step, by the instance that leads the
// operation, when the step reaches `succeeded`, `failed` or `skipped`.
func (m StepDuration) Record(
	ctx context.Context,
	val float64,
	operationType OperationTypeAttr,
	stepState StepStateAttr,
	stepType StepTypeAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Float64Histogram.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Float64Histogram.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.dcctl.operation.type", string(operationType)),
			attribute.String("webitel.dcctl.step.state", string(stepState)),
			attribute.String("webitel.dcctl.step.type", string(stepType)),
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
				attribute.String("webitel.dcctl.operation.type", string(operationType)),
				attribute.String("webitel.dcctl.step.state", string(stepState)),
				attribute.String("webitel.dcctl.step.type", string(stepType)),
			)...,
		),
	)

	m.Float64Histogram.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// This metric MUST be recorded once per step, by the instance that leads the
// operation, when the step reaches `succeeded`, `failed` or `skipped`.
func (m StepDuration) RecordSet(ctx context.Context, val float64, set attribute.Set) {
	if !m.Float64Histogram.Enabled(ctx) {
		return
	}
	if set.Len() == 0 {
		m.Float64Histogram.Record(ctx, val)
		return
	}

	o := metricpool.RecordOptions()
	defer metricpool.PutRecordOptions(o)

	*o = append(*o, metric.WithAttributeSet(set))
	m.Float64Histogram.Record(ctx, val, *o...)
}

// AttrJobStage returns an optional attribute for the "webitel.dcctl.job.stage"
// semantic convention. It represents the start stage of a job managed by dcctl.
func (StepDuration) AttrJobStage(val int) attribute.KeyValue {
	return attribute.Int("webitel.dcctl.job.stage", val)
}

// VoterGeneration is an instrument used to record metric values conforming to
// the "webitel.dcctl.voter.generation" semantic conventions. It represents the
// generation held by a voter.
type VoterGeneration struct {
	metric.Int64Gauge
}

var newVoterGenerationOpts = []metric.Int64GaugeOption{
	metric.WithDescription("The generation held by a voter."),
	metric.WithUnit("{generation}"),
}

// NewVoterGeneration returns a new VoterGeneration instrument.
func NewVoterGeneration(
	m metric.Meter,
	opt ...metric.Int64GaugeOption,
) (VoterGeneration, error) {
	// Check if the meter is nil.
	if m == nil {
		return VoterGeneration{noop.Int64Gauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newVoterGenerationOpts
	} else {
		opt = append(opt, newVoterGenerationOpts...)
	}

	i, err := m.Int64Gauge(
		"webitel.dcctl.voter.generation",
		opt...,
	)
	if err != nil {
		return VoterGeneration{noop.Int64Gauge{}}, err
	}
	return VoterGeneration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m VoterGeneration) Inst() metric.Int64Gauge {
	return m.Int64Gauge
}

// Name returns the semantic convention name of the instrument.
func (VoterGeneration) Name() string {
	return "webitel.dcctl.voter.generation"
}

// Unit returns the semantic convention unit of the instrument
func (VoterGeneration) Unit() string {
	return "{generation}"
}

// Description returns the semantic convention description of the instrument
func (VoterGeneration) Description() string {
	return "The generation held by a voter."
}

// Record records val to the current distribution for attrs.
//
// The voterName is the the name of the voter.
//
// This metric MUST NOT be reported for a voter that is not reachable or holds no
// coordination state.
func (m VoterGeneration) Record(
	ctx context.Context,
	val int64,
	voterName string,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64Gauge.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64Gauge.Record(ctx, val, metric.WithAttributes(
			attribute.String("webitel.dcctl.voter.name", voterName),
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
				attribute.String("webitel.dcctl.voter.name", voterName),
			)...,
		),
	)

	m.Int64Gauge.Record(ctx, val, *o...)
}

// RecordSet records val to the current distribution for set.
//
// This metric MUST NOT be reported for a voter that is not reachable or holds no
// coordination state.
func (m VoterGeneration) RecordSet(ctx context.Context, val int64, set attribute.Set) {
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

// VoterGenerationObservable is an instrument used to record metric values
// conforming to the "webitel.dcctl.voter.generation" semantic conventions. It
// represents the generation held by a voter.
type VoterGenerationObservable struct {
	metric.Int64ObservableGauge
}

var newVoterGenerationObservableOpts = []metric.Int64ObservableGaugeOption{
	metric.WithDescription("The generation held by a voter."),
	metric.WithUnit("{generation}"),
}

// NewVoterGenerationObservable returns a new VoterGenerationObservable
// instrument.
func NewVoterGenerationObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableGaugeOption,
) (VoterGenerationObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return VoterGenerationObservable{noop.Int64ObservableGauge{}}, nil
	}

	if len(opt) == 0 {
		opt = newVoterGenerationObservableOpts
	} else {
		opt = append(opt, newVoterGenerationObservableOpts...)
	}

	i, err := m.Int64ObservableGauge(
		"webitel.dcctl.voter.generation",
		opt...,
	)
	if err != nil {
		return VoterGenerationObservable{noop.Int64ObservableGauge{}}, err
	}
	return VoterGenerationObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m VoterGenerationObservable) Inst() metric.Int64ObservableGauge {
	return m.Int64ObservableGauge
}

// Name returns the semantic convention name of the instrument.
func (VoterGenerationObservable) Name() string {
	return "webitel.dcctl.voter.generation"
}

// Unit returns the semantic convention unit of the instrument
func (VoterGenerationObservable) Unit() string {
	return "{generation}"
}

// Description returns the semantic convention description of the instrument
func (VoterGenerationObservable) Description() string {
	return "The generation held by a voter."
}

// AttrVoterName returns a required attribute for the "webitel.dcctl.voter.name"
// semantic convention. It represents the name of the voter.
func (VoterGenerationObservable) AttrVoterName(val string) attribute.KeyValue {
	return attribute.String("webitel.dcctl.voter.name", val)
}

// VoterStatus is an instrument used to record metric values conforming to the
// "webitel.dcctl.voter.status" semantic conventions. It represents the current
// status of the voter.
type VoterStatus struct {
	metric.Int64UpDownCounter
}

var newVoterStatusOpts = []metric.Int64UpDownCounterOption{
	metric.WithDescription("The current status of the voter."),
	metric.WithUnit("1"),
}

// NewVoterStatus returns a new VoterStatus instrument.
func NewVoterStatus(
	m metric.Meter,
	opt ...metric.Int64UpDownCounterOption,
) (VoterStatus, error) {
	// Check if the meter is nil.
	if m == nil {
		return VoterStatus{noop.Int64UpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newVoterStatusOpts
	} else {
		opt = append(opt, newVoterStatusOpts...)
	}

	i, err := m.Int64UpDownCounter(
		"webitel.dcctl.voter.status",
		opt...,
	)
	if err != nil {
		return VoterStatus{noop.Int64UpDownCounter{}}, err
	}
	return VoterStatus{i}, nil
}

// Inst returns the underlying metric instrument.
func (m VoterStatus) Inst() metric.Int64UpDownCounter {
	return m.Int64UpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (VoterStatus) Name() string {
	return "webitel.dcctl.voter.status"
}

// Unit returns the semantic convention unit of the instrument
func (VoterStatus) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (VoterStatus) Description() string {
	return "The current status of the voter."
}

// Add adds incr to the existing count for attrs.
//
// The voterName is the the name of the voter.
//
// The voterState is the the state of the voter.
//
// A timeseries is produced for every possible value of
// `webitel.dcctl.voter.state`. The value of this metric is 1 for the current
// state of the voter, and 0 for the other states.
func (m VoterStatus) Add(
	ctx context.Context,
	incr int64,
	voterName string,
	voterState VoterStateAttr,
	attrs ...attribute.KeyValue,
) {
	if !m.Int64UpDownCounter.Enabled(ctx) {
		return
	}
	if len(attrs) == 0 {
		m.Int64UpDownCounter.Add(ctx, incr, metric.WithAttributes(
			attribute.String("webitel.dcctl.voter.name", voterName),
			attribute.String("webitel.dcctl.voter.state", string(voterState)),
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
				attribute.String("webitel.dcctl.voter.name", voterName),
				attribute.String("webitel.dcctl.voter.state", string(voterState)),
			)...,
		),
	)

	m.Int64UpDownCounter.Add(ctx, incr, *o...)
}

// AddSet adds incr to the existing count for set.
//
// A timeseries is produced for every possible value of
// `webitel.dcctl.voter.state`. The value of this metric is 1 for the current
// state of the voter, and 0 for the other states.
func (m VoterStatus) AddSet(ctx context.Context, incr int64, set attribute.Set) {
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

// VoterStatusObservable is an instrument used to record metric values conforming
// to the "webitel.dcctl.voter.status" semantic conventions. It represents the
// current status of the voter.
type VoterStatusObservable struct {
	metric.Int64ObservableUpDownCounter
}

var newVoterStatusObservableOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("The current status of the voter."),
	metric.WithUnit("1"),
}

// NewVoterStatusObservable returns a new VoterStatusObservable instrument.
func NewVoterStatusObservable(
	m metric.Meter,
	opt ...metric.Int64ObservableUpDownCounterOption,
) (VoterStatusObservable, error) {
	// Check if the meter is nil.
	if m == nil {
		return VoterStatusObservable{noop.Int64ObservableUpDownCounter{}}, nil
	}

	if len(opt) == 0 {
		opt = newVoterStatusObservableOpts
	} else {
		opt = append(opt, newVoterStatusObservableOpts...)
	}

	i, err := m.Int64ObservableUpDownCounter(
		"webitel.dcctl.voter.status",
		opt...,
	)
	if err != nil {
		return VoterStatusObservable{noop.Int64ObservableUpDownCounter{}}, err
	}
	return VoterStatusObservable{i}, nil
}

// Inst returns the underlying metric instrument.
func (m VoterStatusObservable) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (VoterStatusObservable) Name() string {
	return "webitel.dcctl.voter.status"
}

// Unit returns the semantic convention unit of the instrument
func (VoterStatusObservable) Unit() string {
	return "1"
}

// Description returns the semantic convention description of the instrument
func (VoterStatusObservable) Description() string {
	return "The current status of the voter."
}

// AttrVoterName returns a required attribute for the "webitel.dcctl.voter.name"
// semantic convention. It represents the name of the voter.
func (VoterStatusObservable) AttrVoterName(val string) attribute.KeyValue {
	return attribute.String("webitel.dcctl.voter.name", val)
}

// AttrVoterState returns a required attribute for the
// "webitel.dcctl.voter.state" semantic convention. It represents the state of
// the voter.
func (VoterStatusObservable) AttrVoterState(val VoterStateAttr) attribute.KeyValue {
	return attribute.String("webitel.dcctl.voter.state", string(val))
}
