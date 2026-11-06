// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

import "go.opentelemetry.io/otel/attribute"

// Namespace: webitel
const (
	// WebitelHealthCheckGroupKey is the attribute Key conforming to the
	// "webitel.health.check.group" semantic conventions. It represents the group
	// the check belongs to, which decides how a failure affects readiness.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: The group is chosen at registration, not by the check itself. Making
	// shared infrastructure `critical` is usually wrong: one outage then takes the
	// whole fleet out of rotation at once.
	WebitelHealthCheckGroupKey = attribute.Key("webitel.health.check.group")

	// WebitelHealthCheckNameKey is the attribute Key conforming to the
	// "webitel.health.check.name" semantic conventions. It represents the name the
	// check was registered under.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "postgres", "rabbitmq", "consul"
	// Note: A check is a probe the node runs in the background on its own schedule
	// — reaching a dependency, or testing an internal condition — and whose
	// last result is cached. Reading these metrics never triggers a run, so
	// scraping adds no load to the dependency being checked.
	WebitelHealthCheckNameKey = attribute.Key("webitel.health.check.name")

	// WebitelHealthCheckStatusKey is the attribute Key conforming to the
	// "webitel.health.check.status" semantic conventions. It represents the status
	// a check transitioned into.
	//
	// Type: Enum
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples:
	// Note: A check goes `fail` only after several consecutive failures, and
	// recovers on the first success, so a flapping dependency does not flip the
	// node on every run. `unknown` — never ran, or the result went stale — is
	// never a stored status, so it is never a transition target and has no member
	// here.
	WebitelHealthCheckStatusKey = attribute.Key("webitel.health.check.status")
)

// WebitelHealthCheckName returns an attribute KeyValue conforming to the
// "webitel.health.check.name" semantic conventions. It represents the name the
// check was registered under.
func WebitelHealthCheckName(val string) attribute.KeyValue {
	return WebitelHealthCheckNameKey.String(val)
}

// Enum values for webitel.health.check.group
var (
	// Is this process wedged? A failure takes the node out of rotation and fails
	// its liveness probe.
	// Stability: development
	WebitelHealthCheckGroupLiveness = WebitelHealthCheckGroupKey.String("liveness")
	// A node-local fault, where moving traffic to another node genuinely helps; a
	// failure takes the node out of rotation.
	// Stability: development
	WebitelHealthCheckGroupCritical = WebitelHealthCheckGroupKey.String("critical")
	// Everything else, typically shared infrastructure; a failure marks the node
	// degraded but leaves it in rotation.
	// Stability: development
	WebitelHealthCheckGroupInformational = WebitelHealthCheckGroupKey.String("informational")
)

// Enum values for webitel.health.check.status
var (
	// Passing check.
	// Stability: development
	WebitelHealthCheckStatusOk = WebitelHealthCheckStatusKey.String("ok")
	// Check that failed past its threshold.
	// Stability: development
	WebitelHealthCheckStatusFail = WebitelHealthCheckStatusKey.String("fail")
)
