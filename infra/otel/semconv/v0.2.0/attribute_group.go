// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

import "go.opentelemetry.io/otel/attribute"

// Namespace: webitel
const (
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
	// "webitel.health.check.name" semantic conventions. It represents the name of
	// the health check.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "postgres", "rabbitmq", "consul"
	// Note: A health check is a probe that the node runs in the background on its
	// own schedule; its last result is cached. Reading a health check metric MUST
	// NOT trigger a run of the check.
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

// WebitelHealthCheckName returns an attribute KeyValue conforming to the
// "webitel.health.check.name" semantic conventions. It represents the name of
// the health check.
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
