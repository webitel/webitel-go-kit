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
	// Note: `unknown` is never a stored status, so it is never a transition target
	// and has no member here.
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
	// liveness probe; a failure also fails readiness.
	// Stability: development
	WebitelHealthCheckGroupLiveness = WebitelHealthCheckGroupKey.String("liveness")
	// node-local dependency; a failure takes the node out of rotation.
	// Stability: development
	WebitelHealthCheckGroupCritical = WebitelHealthCheckGroupKey.String("critical")
	// dependency whose failure only degrades; the node stays in rotation.
	// Stability: development
	WebitelHealthCheckGroupInformational = WebitelHealthCheckGroupKey.String("informational")
)

// Enum values for webitel.health.check.status
var (
	// passing check.
	// Stability: development
	WebitelHealthCheckStatusOk = WebitelHealthCheckStatusKey.String("ok")
	// check that failed past its threshold.
	// Stability: development
	WebitelHealthCheckStatusFail = WebitelHealthCheckStatusKey.String("fail")
)
