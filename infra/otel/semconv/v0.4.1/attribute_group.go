// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

import "go.opentelemetry.io/otel/attribute"

// Namespace: webitel
const (
	// WebitelDatacenterNameKey is the attribute Key conforming to the
	// "webitel.datacenter.name" semantic conventions. It represents the name of the
	// datacenter.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "dc_a", "dc_b"
	// Note: The value MUST be the name of the datacenter as the deployment is
	// configured with it.
	WebitelDatacenterNameKey = attribute.Key("webitel.datacenter.name")

	// WebitelDcctlAlarmNameKey is the attribute Key conforming to the
	// "webitel.dcctl.alarm.name" semantic conventions. It represents the identifier
	// of the alarm.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: The value is the `id` of an alarm of the dcctl alarm catalog.
	WebitelDcctlAlarmNameKey = attribute.Key("webitel.dcctl.alarm.name")

	// WebitelDcctlDatacenterStateKey is the attribute Key conforming to the
	// "webitel.dcctl.datacenter.state" semantic conventions. It represents the
	// state of the datacenter.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: The state is the first of the values, in the order listed, whose
	// condition holds. A datacenter is in exactly one state. A condition whose
	// source is not reachable is skipped. The value `undetermined` is reported when
	// the voters have a conflict, or when voters that do not answer may hold a
	// newer record. dcctl never guesses `active` or `standby` then.
	WebitelDcctlDatacenterStateKey = attribute.Key("webitel.dcctl.datacenter.state")

	// WebitelDcctlJobStageKey is the attribute Key conforming to the
	// "webitel.dcctl.job.stage" semantic conventions. It represents the start stage
	// of a job managed by dcctl.
	//
	// Type: int
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: 1, 2, 3
	// Note: The value is the `dcctl_stage` of the job. Jobs start in ascending
	// order of the stage and stop in descending order.
	WebitelDcctlJobStageKey = attribute.Key("webitel.dcctl.job.stage")

	// WebitelDcctlJobStateKey is the attribute Key conforming to the
	// "webitel.dcctl.job.state" semantic conventions. It represents the state of a
	// job managed by dcctl.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlJobStateKey = attribute.Key("webitel.dcctl.job.state")

	// WebitelDcctlOperationModeKey is the attribute Key conforming to the
	// "webitel.dcctl.operation.mode" semantic conventions. It represents the mode
	// in which the operation changes the coordination state.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlOperationModeKey = attribute.Key("webitel.dcctl.operation.mode")

	// WebitelDcctlOperationStateKey is the attribute Key conforming to the
	// "webitel.dcctl.operation.state" semantic conventions. It represents the state
	// of the operation.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlOperationStateKey = attribute.Key("webitel.dcctl.operation.state")

	// WebitelDcctlOperationTypeKey is the attribute Key conforming to the
	// "webitel.dcctl.operation.type" semantic conventions. It represents the type
	// of the operation.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlOperationTypeKey = attribute.Key("webitel.dcctl.operation.type")

	// WebitelDcctlPatroniRoleKey is the attribute Key conforming to the
	// "webitel.dcctl.patroni.role" semantic conventions. It represents the Patroni
	// role of a cluster member.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlPatroniRoleKey = attribute.Key("webitel.dcctl.patroni.role")

	// WebitelDcctlQuorumModeKey is the attribute Key conforming to the
	// "webitel.dcctl.quorum.mode" semantic conventions. It represents the mode in
	// which the coordination state can be changed.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlQuorumModeKey = attribute.Key("webitel.dcctl.quorum.mode")

	// WebitelDcctlSourceStateKey is the attribute Key conforming to the
	// "webitel.dcctl.source.state" semantic conventions. It represents the state of
	// a source of dcctl.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlSourceStateKey = attribute.Key("webitel.dcctl.source.state")

	// WebitelDcctlSourceSystemKey is the attribute Key conforming to the
	// "webitel.dcctl.source.system" semantic conventions. It represents the system
	// dcctl reads the state from.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlSourceSystemKey = attribute.Key("webitel.dcctl.source.system")

	// WebitelDcctlStepStateKey is the attribute Key conforming to the
	// "webitel.dcctl.step.state" semantic conventions. It represents the state of
	// the step of an operation.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlStepStateKey = attribute.Key("webitel.dcctl.step.state")

	// WebitelDcctlStepTypeKey is the attribute Key conforming to the
	// "webitel.dcctl.step.type" semantic conventions. It represents the type of the
	// step of an operation.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: The stage of the managed jobs a step stops or starts is not part of the
	// value, it is set in `webitel.dcctl.job.stage`.
	WebitelDcctlStepTypeKey = attribute.Key("webitel.dcctl.step.type")

	// WebitelDcctlVoterNameKey is the attribute Key conforming to the
	// "webitel.dcctl.voter.name" semantic conventions. It represents the name of
	// the voter.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "dc_a", "dc_b", "arbiter", "consul"
	// Note: The value MUST be the name of the datacenter whose Consul is the voter,
	// or `arbiter` for the arbiter. In the stretch profile the single cluster-wide
	// Consul voter is named `consul`.
	WebitelDcctlVoterNameKey = attribute.Key("webitel.dcctl.voter.name")

	// WebitelDcctlVoterStateKey is the attribute Key conforming to the
	// "webitel.dcctl.voter.state" semantic conventions. It represents the state of
	// the voter.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	WebitelDcctlVoterStateKey = attribute.Key("webitel.dcctl.voter.state")

	// WebitelDomainIDKey is the attribute Key conforming to the "webitel.domain.id"
	// semantic conventions. It represents the ID of the domain (tenant) the
	// operation belongs to.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "1", "42"
	// Note: The value MUST be the decimal domain ID taken from a verified
	// authorization context, never from client input. It is propagated in W3C
	// baggage under the same key.
	WebitelDomainIDKey = attribute.Key("webitel.domain.id")

	// WebitelHealthCheckGroupKey is the attribute Key conforming to the
	// "webitel.health.check.group" semantic conventions. It represents the group of
	// the health check, which determines how a failure of the check affects the
	// readiness of the node.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: The group is assigned when the check is registered. A check of shared
	// infrastructure SHOULD NOT be assigned to `critical`.
	WebitelHealthCheckGroupKey = attribute.Key("webitel.health.check.group")

	// WebitelHealthCheckNameKey is the attribute Key conforming to the
	// "webitel.health.check.name" semantic conventions. It represents the name that
	// tells apart health checks of the same system.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "whatsapp", "standby"
	// Note: Instrumentations SHOULD set the name only when a node runs more than
	// one check of the same `webitel.health.check.system`. The value MUST be
	// static: it MUST NOT contain a host, an address, a tenant or any other value
	// that varies at runtime.
	WebitelHealthCheckNameKey = attribute.Key("webitel.health.check.name")

	// WebitelHealthCheckStateKey is the attribute Key conforming to the
	// "webitel.health.check.state" semantic conventions. It represents the state of
	// the health check.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: A check MUST move to `fail` only after several consecutive failures,
	// and MUST move to `ok` on the first success.
	WebitelHealthCheckStateKey = attribute.Key("webitel.health.check.state")

	// WebitelHealthCheckSystemKey is the attribute Key conforming to the
	// "webitel.health.check.system" semantic conventions. It represents the system
	// the health check probes.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: A health check is a probe that the node runs in the background on its
	// own schedule; its last result is cached. Reading a health check metric MUST
	// NOT trigger a run of the check. A system that has a value in `db.system.name`
	// , `messaging.system` or `rpc.system.name` MUST use that value. A check of a
	// listener of the node itself SHOULD use the protocol of the listener, such as
	// `grpc` or `http`. A custom value MUST be of low cardinality.
	WebitelHealthCheckSystemKey = attribute.Key("webitel.health.check.system")

	// WebitelHealthStateKey is the attribute Key conforming to the
	// "webitel.health.state" semantic conventions. It represents the readiness
	// state of the node.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: The node MUST be `not_ready` before any check has passed, while a
	// `liveness` or `critical` check fails, and while the node shuts down.
	WebitelHealthStateKey = attribute.Key("webitel.health.state")

	// WebitelKBArticleIndexEmbeddedKey is the attribute Key conforming to the
	// "webitel.kb.article.index.embedded" semantic conventions. It represents the
	// whether the article version was embedded for vector search.
	//
	// Type: boolean
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: true, false
	// Note: `false` for a space without vector search, where a version is only
	// indexed for full-text search.
	WebitelKBArticleIndexEmbeddedKey = attribute.Key("webitel.kb.article.index.embedded")

	// WebitelKBArticleIndexStateKey is the attribute Key conforming to the
	// "webitel.kb.article.index.state" semantic conventions. It represents the
	// state of the index of an article.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: The state is that of the latest version of the article.
	WebitelKBArticleIndexStateKey = attribute.Key("webitel.kb.article.index.state")

	// WebitelKBRerankModelKey is the attribute Key conforming to the
	// "webitel.kb.rerank.model" semantic conventions. It represents the name of the
	// model a rerank request is made to, as named by the provider.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "rerank-v3.5", "bge-reranker-v2-m3"
	WebitelKBRerankModelKey = attribute.Key("webitel.kb.rerank.model")

	// WebitelKBRerankProviderKey is the attribute Key conforming to the
	// "webitel.kb.rerank.provider" semantic conventions. It represents the name of
	// the rerank provider.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "cohere", "bge-reranker"
	WebitelKBRerankProviderKey = attribute.Key("webitel.kb.rerank.provider")

	// WebitelOutboxRelayStateKey is the attribute Key conforming to the
	// "webitel.outbox.relay.state" semantic conventions. It represents the state of
	// the node in the outbox relay.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: Only the node that holds the relay lock MUST be in the `leader` state.
	WebitelOutboxRelayStateKey = attribute.Key("webitel.outbox.relay.state")
)

// WebitelDatacenterName returns an attribute KeyValue conforming to the
// "webitel.datacenter.name" semantic conventions. It represents the name of the
// datacenter.
func WebitelDatacenterName(val string) attribute.KeyValue {
	return WebitelDatacenterNameKey.String(val)
}

// WebitelDcctlJobStage returns an attribute KeyValue conforming to the
// "webitel.dcctl.job.stage" semantic conventions. It represents the start stage
// of a job managed by dcctl.
func WebitelDcctlJobStage(val int) attribute.KeyValue {
	return WebitelDcctlJobStageKey.Int(val)
}

// WebitelDcctlVoterName returns an attribute KeyValue conforming to the
// "webitel.dcctl.voter.name" semantic conventions. It represents the name of the
// voter.
func WebitelDcctlVoterName(val string) attribute.KeyValue {
	return WebitelDcctlVoterNameKey.String(val)
}

// WebitelDomainID returns an attribute KeyValue conforming to the
// "webitel.domain.id" semantic conventions. It represents the ID of the domain
// (tenant) the operation belongs to.
func WebitelDomainID(val string) attribute.KeyValue {
	return WebitelDomainIDKey.String(val)
}

// WebitelHealthCheckName returns an attribute KeyValue conforming to the
// "webitel.health.check.name" semantic conventions. It represents the name that
// tells apart health checks of the same system.
func WebitelHealthCheckName(val string) attribute.KeyValue {
	return WebitelHealthCheckNameKey.String(val)
}

// WebitelKBArticleIndexEmbedded returns an attribute KeyValue conforming to the
// "webitel.kb.article.index.embedded" semantic conventions. It represents the
// whether the article version was embedded for vector search.
func WebitelKBArticleIndexEmbedded(val bool) attribute.KeyValue {
	return WebitelKBArticleIndexEmbeddedKey.Bool(val)
}

// WebitelKBRerankModel returns an attribute KeyValue conforming to the
// "webitel.kb.rerank.model" semantic conventions. It represents the name of the
// model a rerank request is made to, as named by the provider.
func WebitelKBRerankModel(val string) attribute.KeyValue {
	return WebitelKBRerankModelKey.String(val)
}

// WebitelKBRerankProvider returns an attribute KeyValue conforming to the
// "webitel.kb.rerank.provider" semantic conventions. It represents the name of
// the rerank provider.
func WebitelKBRerankProvider(val string) attribute.KeyValue {
	return WebitelKBRerankProviderKey.String(val)
}

// Enum values for webitel.dcctl.alarm.name
var (
	// Two valid records have the same generation and different active datacenters.
	// Stability: development
	WebitelDcctlAlarmNameConflict = WebitelDcctlAlarmNameKey.String("conflict")
	// A voter holds a record with a generation above the effective one, without a
	// majority and without `forced`.
	// Stability: development
	WebitelDcctlAlarmNameUncommitted = WebitelDcctlAlarmNameKey.String("uncommitted")
	// The voter of a datacenter holds a generation lower than the effective one.
	// Stability: development
	WebitelDcctlAlarmNameStale = WebitelDcctlAlarmNameKey.String("stale")
	// A managed job of a datacenter that is not active is not stopped.
	// Stability: development
	WebitelDcctlAlarmNameSplitBrain = WebitelDcctlAlarmNameKey.String("split_brain")
	// The Patroni leader of a datacenter that is not active is a primary.
	// Stability: development
	WebitelDcctlAlarmNameDBSplit = WebitelDcctlAlarmNameKey.String("db_split")
	// More than one primary is reachable across all datacenters.
	// Stability: development
	WebitelDcctlAlarmNameMultiplePrimaries = WebitelDcctlAlarmNameKey.String("multiple_primaries")
	// The active datacenter has no primary.
	// Stability: development
	WebitelDcctlAlarmNameDBPrimaryMissing = WebitelDcctlAlarmNameKey.String("db_primary_missing")
	// The datacenter has been unreachable longer than the degraded threshold.
	// Stability: development
	WebitelDcctlAlarmNameDatacenterUnreachable = WebitelDcctlAlarmNameKey.String("datacenter_unreachable")
	// The datacenter of the Patroni leader is not the active datacenter.
	// Stability: development
	WebitelDcctlAlarmNameDBLeaderMismatch = WebitelDcctlAlarmNameKey.String("db_leader_mismatch")
	// The lease is not renewed, or does not match the effective state.
	// Stability: development
	WebitelDcctlAlarmNameLeaseLost = WebitelDcctlAlarmNameKey.String("lease_lost")
	// A managed job of the active datacenter is not healthy.
	// Stability: development
	WebitelDcctlAlarmNameAppsDegraded = WebitelDcctlAlarmNameKey.String("apps_degraded")
	// A Patroni member of the datacenter is unreachable or unusable, or Patroni is
	// paused.
	// Stability: development
	WebitelDcctlAlarmNameDBDegraded = WebitelDcctlAlarmNameKey.String("db_degraded")
	// The Patroni leader of a datacenter that is not active does not follow the
	// active datacenter.
	// Stability: development
	WebitelDcctlAlarmNameStandbyNotFollowing = WebitelDcctlAlarmNameKey.String("standby_not_following")
	// The voter has been unreachable longer than the degraded threshold.
	// Stability: development
	WebitelDcctlAlarmNameVoterUnreachable = WebitelDcctlAlarmNameKey.String("voter_unreachable")
	// Fewer voters than a majority are reachable.
	// Stability: development
	WebitelDcctlAlarmNameQuorumLost = WebitelDcctlAlarmNameKey.String("quorum_lost")
	// The arbiter is reachable and its storage is empty.
	// Stability: development
	WebitelDcctlAlarmNameArbiterUninitialized = WebitelDcctlAlarmNameKey.String("arbiter_uninitialized")
	// An operation is in the `interrupted` state.
	// Stability: development
	WebitelDcctlAlarmNameOperationInterrupted = WebitelDcctlAlarmNameKey.String("operation_interrupted")
)

// Enum values for webitel.dcctl.datacenter.state
var (
	// No source of the datacenter is reachable.
	// Stability: development
	WebitelDcctlDatacenterStateUnreachable = WebitelDcctlDatacenterStateKey.String("unreachable")
	// No voter holds the coordination state.
	// Stability: development
	WebitelDcctlDatacenterStateUninitialized = WebitelDcctlDatacenterStateKey.String("uninitialized")
	// The role of the datacenter cannot be told, because the voters give no decided
	// effective state.
	// Stability: development
	WebitelDcctlDatacenterStateUndetermined = WebitelDcctlDatacenterStateKey.String("undetermined")
	// The datacenter is the active datacenter of the effective state.
	// Stability: development
	WebitelDcctlDatacenterStateActive = WebitelDcctlDatacenterStateKey.String("active")
	// The datacenter is not active, and a managed job of it is not stopped.
	// Stability: development
	WebitelDcctlDatacenterStateSplitBrain = WebitelDcctlDatacenterStateKey.String("split_brain")
	// The datacenter is not active, and its Patroni leader is a primary.
	// Stability: development
	WebitelDcctlDatacenterStateDBSplit = WebitelDcctlDatacenterStateKey.String("db_split")
	// The voter of the datacenter holds a generation lower than the effective one.
	// Stability: development
	WebitelDcctlDatacenterStateStale = WebitelDcctlDatacenterStateKey.String("stale")
	// The datacenter is marked as needing recovery in the effective state.
	// Stability: development
	WebitelDcctlDatacenterStateNeedsRecovery = WebitelDcctlDatacenterStateKey.String("needs_recovery")
	// The datacenter is not active, and none of the other states apply.
	// Stability: development
	WebitelDcctlDatacenterStateStandby = WebitelDcctlDatacenterStateKey.String("standby")
)

// Enum values for webitel.dcctl.job.state
var (
	// The job is running and healthy.
	// Stability: development
	WebitelDcctlJobStateHealthy = WebitelDcctlJobStateKey.String("healthy")
	// The job is not stopped and is not healthy, including while it starts.
	// Stability: development
	WebitelDcctlJobStateUnhealthy = WebitelDcctlJobStateKey.String("unhealthy")
	// The job is stopped and has allocations that are still pending, running or
	// unknown.
	// Stability: development
	WebitelDcctlJobStateStopping = WebitelDcctlJobStateKey.String("stopping")
	// The job is stopped and has no allocations left.
	// Stability: development
	WebitelDcctlJobStateStopped = WebitelDcctlJobStateKey.String("stopped")
)

// Enum values for webitel.dcctl.operation.mode
var (
	// The operation changes the state with a majority of voters.
	// Stability: development
	WebitelDcctlOperationModeQuorum = WebitelDcctlOperationModeKey.String("quorum")
	// The operation changes the state without a majority of voters.
	// Stability: development
	WebitelDcctlOperationModeForced = WebitelDcctlOperationModeKey.String("forced")
)

// Enum values for webitel.dcctl.operation.state
var (
	// The operation is in progress.
	// Stability: development
	WebitelDcctlOperationStateRunning = WebitelDcctlOperationStateKey.String("running")
	// The operation completed.
	// Stability: development
	WebitelDcctlOperationStateSucceeded = WebitelDcctlOperationStateKey.String("succeeded")
	// The operation failed.
	// Stability: development
	WebitelDcctlOperationStateFailed = WebitelDcctlOperationStateKey.String("failed")
	// The operation was interrupted and waits to be resumed or abandoned.
	// Stability: development
	WebitelDcctlOperationStateInterrupted = WebitelDcctlOperationStateKey.String("interrupted")
	// The operation was abandoned.
	// Stability: development
	WebitelDcctlOperationStateAbandoned = WebitelDcctlOperationStateKey.String("abandoned")
)

// Enum values for webitel.dcctl.operation.type
var (
	// Planned switch of the active datacenter.
	// Stability: development
	WebitelDcctlOperationTypeSwitch = WebitelDcctlOperationTypeKey.String("switch")
	// Forced switch of the active datacenter.
	// Stability: development
	WebitelDcctlOperationTypeFailover = WebitelDcctlOperationTypeKey.String("failover")
	// Demotion of the database of a datacenter to a standby.
	// Stability: development
	WebitelDcctlOperationTypeDemote = WebitelDcctlOperationTypeKey.String("demote")
	// Repointing of the standby database of a datacenter to the active datacenter.
	// Stability: development
	WebitelDcctlOperationTypeRepoint = WebitelDcctlOperationTypeKey.String("repoint")
	// Removal of a datacenter from the datacenters that need recovery.
	// Stability: development
	WebitelDcctlOperationTypeMarkRecovered = WebitelDcctlOperationTypeKey.String("mark_recovered")
	// Resolution of a conflict between records of the coordination state.
	// Stability: development
	WebitelDcctlOperationTypeResolveConflict = WebitelDcctlOperationTypeKey.String("resolve_conflict")
	// Initialization of the coordination state.
	// Stability: development
	WebitelDcctlOperationTypeInitialize = WebitelDcctlOperationTypeKey.String("initialize")
	// Initialization of the arbiter.
	// Stability: development
	WebitelDcctlOperationTypeInitializeArbiter = WebitelDcctlOperationTypeKey.String("initialize_arbiter")
	// Synchronization of managed jobs with the active datacenter.
	// Stability: development
	WebitelDcctlOperationTypeSyncApps = WebitelDcctlOperationTypeKey.String("sync_apps")
)

// Enum values for webitel.dcctl.patroni.role
var (
	// The member is a primary.
	// Stability: development
	WebitelDcctlPatroniRolePrimary = WebitelDcctlPatroniRoleKey.String("primary")
	// The member is the leader of a standby cluster.
	// Stability: development
	WebitelDcctlPatroniRoleStandbyLeader = WebitelDcctlPatroniRoleKey.String("standby_leader")
	// The member is a replica.
	// Stability: development
	WebitelDcctlPatroniRoleReplica = WebitelDcctlPatroniRoleKey.String("replica")
	// The member has a role other than the listed ones.
	// Stability: development
	WebitelDcctlPatroniRoleOther = WebitelDcctlPatroniRoleKey.String("other")
	// The role is not known because the member is not reachable.
	// Stability: development
	WebitelDcctlPatroniRoleUnknown = WebitelDcctlPatroniRoleKey.String("unknown")
)

// Enum values for webitel.dcctl.quorum.mode
var (
	// A majority of voters is reachable.
	// Stability: development
	WebitelDcctlQuorumModeQuorum = WebitelDcctlQuorumModeKey.String("quorum")
	// A majority of voters is not reachable, and only a forced operation can change
	// the state.
	// Stability: development
	WebitelDcctlQuorumModeForcedOnly = WebitelDcctlQuorumModeKey.String("forced_only")
)

// Enum values for webitel.dcctl.source.state
var (
	// All the queries to the source succeed.
	// Stability: development
	WebitelDcctlSourceStateOk = WebitelDcctlSourceStateKey.String("ok")
	// Some of the queries to the source fail.
	// Stability: development
	WebitelDcctlSourceStatePartial = WebitelDcctlSourceStateKey.String("partial")
	// The source is not reachable.
	// Stability: development
	WebitelDcctlSourceStateUnreachable = WebitelDcctlSourceStateKey.String("unreachable")
)

// Enum values for webitel.dcctl.source.system
var (
	// [Consul]
	// Stability: development
	//
	// [Consul]: https://www.consul.io/
	WebitelDcctlSourceSystemConsul = WebitelDcctlSourceSystemKey.String("consul")
	// [Nomad]
	// Stability: development
	//
	// [Nomad]: https://www.nomadproject.io/
	WebitelDcctlSourceSystemNomad = WebitelDcctlSourceSystemKey.String("nomad")
	// [Patroni]
	// Stability: development
	//
	// [Patroni]: https://patroni.readthedocs.io/
	WebitelDcctlSourceSystemPatroni = WebitelDcctlSourceSystemKey.String("patroni")
	// The dcctl arbiter.
	// Stability: development
	WebitelDcctlSourceSystemArbiter = WebitelDcctlSourceSystemKey.String("arbiter")
)

// Enum values for webitel.dcctl.step.state
var (
	// The step has not started.
	// Stability: development
	WebitelDcctlStepStatePending = WebitelDcctlStepStateKey.String("pending")
	// The step is in progress.
	// Stability: development
	WebitelDcctlStepStateRunning = WebitelDcctlStepStateKey.String("running")
	// The step completed.
	// Stability: development
	WebitelDcctlStepStateSucceeded = WebitelDcctlStepStateKey.String("succeeded")
	// The step failed.
	// Stability: development
	WebitelDcctlStepStateFailed = WebitelDcctlStepStateKey.String("failed")
	// The step was skipped.
	// Stability: development
	WebitelDcctlStepStateSkipped = WebitelDcctlStepStateKey.String("skipped")
)

// Enum values for webitel.dcctl.step.type
var (
	// The managed jobs of the datacenter are checked before the operation changes
	// them.
	// Stability: development
	WebitelDcctlStepTypeAppsPreflight = WebitelDcctlStepTypeKey.String("apps_preflight")
	// The managed jobs of a stage are stopped.
	// Stability: development
	WebitelDcctlStepTypeAppsStop = WebitelDcctlStepTypeKey.String("apps_stop")
	// The managed jobs of a stage are started.
	// Stability: development
	WebitelDcctlStepTypeAppsStart = WebitelDcctlStepTypeKey.String("apps_start")
	// The generation of the coordination state is changed.
	// Stability: development
	WebitelDcctlStepTypeGeneration = WebitelDcctlStepTypeKey.String("generation")
	// The result of the operation is verified.
	// Stability: development
	WebitelDcctlStepTypeVerify = WebitelDcctlStepTypeKey.String("verify")
	// The Patroni cluster of the datacenter is demoted to a standby.
	// Stability: development
	WebitelDcctlStepTypePatroniDemote = WebitelDcctlStepTypeKey.String("patroni_demote")
	// The standby waits to catch up with the primary.
	// Stability: development
	WebitelDcctlStepTypePatroniCatchup = WebitelDcctlStepTypeKey.String("patroni_catchup")
	// The step waits until the datacenter has no primary.
	// Stability: development
	WebitelDcctlStepTypePatroniNoPrimary = WebitelDcctlStepTypeKey.String("patroni_no_primary")
	// The Patroni cluster of the datacenter is promoted.
	// Stability: development
	WebitelDcctlStepTypePatroniPromote = WebitelDcctlStepTypeKey.String("patroni_promote")
	// The standby cluster is pointed to the active datacenter.
	// Stability: development
	WebitelDcctlStepTypePatroniRepoint = WebitelDcctlStepTypeKey.String("patroni_repoint")
	// The standby cluster starts to follow the active datacenter.
	// Stability: development
	WebitelDcctlStepTypePatroniFollow = WebitelDcctlStepTypeKey.String("patroni_follow")
	// Patroni switches the leader over to the target datacenter.
	// Stability: development
	WebitelDcctlStepTypePatroniSwitchover = WebitelDcctlStepTypeKey.String("patroni_switchover")
	// Patroni fails over inside one cluster that has no leader (stretch profile).
	// Stability: development
	WebitelDcctlStepTypePatroniFailover = WebitelDcctlStepTypeKey.String("patroni_failover")
)

// Enum values for webitel.dcctl.voter.state
var (
	// The voter is reachable and holds the coordination state.
	// Stability: development
	WebitelDcctlVoterStateOk = WebitelDcctlVoterStateKey.String("ok")
	// The voter is not reachable.
	// Stability: development
	WebitelDcctlVoterStateUnreachable = WebitelDcctlVoterStateKey.String("unreachable")
	// The voter is reachable and holds no coordination state.
	// Stability: development
	WebitelDcctlVoterStateUninitialized = WebitelDcctlVoterStateKey.String("uninitialized")
	// The voter answered, but its `dcctl/state` record cannot be decoded.
	// Stability: development
	WebitelDcctlVoterStateInvalid = WebitelDcctlVoterStateKey.String("invalid")
)

// Enum values for webitel.health.check.group
var (
	// The check tests whether the process is operational. A failure takes the node
	// out of rotation and fails its liveness probe.
	// Stability: development
	WebitelHealthCheckGroupLiveness = WebitelHealthCheckGroupKey.String("liveness")
	// The check tests a dependency local to the node. A failure takes the node out
	// of rotation.
	// Stability: development
	WebitelHealthCheckGroupCritical = WebitelHealthCheckGroupKey.String("critical")
	// The check tests a shared dependency. A failure marks the node degraded and
	// leaves it in rotation.
	// Stability: development
	WebitelHealthCheckGroupInformational = WebitelHealthCheckGroupKey.String("informational")
)

// Enum values for webitel.health.check.state
var (
	// The check passes.
	// Stability: development
	WebitelHealthCheckStateOk = WebitelHealthCheckStateKey.String("ok")
	// The check fails.
	// Stability: development
	WebitelHealthCheckStateFail = WebitelHealthCheckStateKey.String("fail")
	// The check has not run yet, or its result is stale.
	// Stability: development
	WebitelHealthCheckStateUnknown = WebitelHealthCheckStateKey.String("unknown")
)

// Enum values for webitel.health.check.system
var (
	// [PostgreSQL]
	// Stability: development
	//
	// [PostgreSQL]: https://www.postgresql.org/
	WebitelHealthCheckSystemPostgreSQL = WebitelHealthCheckSystemKey.String("postgresql")
	// [Redis]
	// Stability: development
	//
	// [Redis]: https://redis.io/
	WebitelHealthCheckSystemRedis = WebitelHealthCheckSystemKey.String("redis")
	// [RabbitMQ]
	// Stability: development
	//
	// [RabbitMQ]: https://www.rabbitmq.com/
	WebitelHealthCheckSystemRabbitMQ = WebitelHealthCheckSystemKey.String("rabbitmq")
	// [gRPC]
	// Stability: development
	//
	// [gRPC]: https://grpc.io/
	WebitelHealthCheckSystemGRPC = WebitelHealthCheckSystemKey.String("grpc")
	// [HTTP]
	// Stability: development
	//
	// [HTTP]: https://www.rfc-editor.org/rfc/rfc9110.html
	WebitelHealthCheckSystemHTTP = WebitelHealthCheckSystemKey.String("http")
)

// Enum values for webitel.health.state
var (
	// The node is in rotation and all its checks pass.
	// Stability: development
	WebitelHealthStateReady = WebitelHealthStateKey.String("ready")
	// The node is in rotation and at least one `informational` check fails.
	// Stability: development
	WebitelHealthStateDegraded = WebitelHealthStateKey.String("degraded")
	// The node is out of rotation.
	// Stability: development
	WebitelHealthStateNotReady = WebitelHealthStateKey.String("not_ready")
)

// Enum values for webitel.kb.article.index.state
var (
	// The latest version of the article is waiting to be indexed.
	// Stability: development
	WebitelKBArticleIndexStatePending = WebitelKBArticleIndexStateKey.String("pending")
	// The latest version of the article is being indexed.
	// Stability: development
	WebitelKBArticleIndexStateIndexing = WebitelKBArticleIndexStateKey.String("indexing")
	// The latest version of the article is searchable.
	// Stability: development
	WebitelKBArticleIndexStateIndexed = WebitelKBArticleIndexStateKey.String("indexed")
	// The latest version of the article could not be indexed.
	// Stability: development
	WebitelKBArticleIndexStateFailed = WebitelKBArticleIndexStateKey.String("failed")
)

// Enum values for webitel.outbox.relay.state
var (
	// The node runs the outbox relay.
	// Stability: development
	WebitelOutboxRelayStateLeader = WebitelOutboxRelayStateKey.String("leader")
	// The node does not run the outbox relay.
	// Stability: development
	WebitelOutboxRelayStateFollower = WebitelOutboxRelayStateKey.String("follower")
)
