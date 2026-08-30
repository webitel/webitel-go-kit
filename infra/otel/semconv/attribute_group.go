// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

import "go.opentelemetry.io/otel/attribute"

// Namespace: webitel
const (
	// WebitelDBPrepareStmtNameKey is the attribute Key conforming to the
	// "webitel.db.prepare_stmt.name" semantic conventions. It represents the name
	// of the prepared statement.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "stmt_find_user_by_id", "lrupsc_1_0"
	WebitelDBPrepareStmtNameKey = attribute.Key("webitel.db.prepare_stmt.name")

	// WebitelDBUserKey is the attribute Key conforming to the "webitel.db.user"
	// semantic conventions. It represents the database user the connection
	// authenticates as.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	//
	// Examples: "opensips", "webitel"
	// Note: A connection property, not a tenant or end-user identifier. MUST NOT be
	// used as a metric attribute.
	WebitelDBUserKey = attribute.Key("webitel.db.user")

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

// WebitelDBPrepareStmtName returns an attribute KeyValue conforming to the
// "webitel.db.prepare_stmt.name" semantic conventions. It represents the name of
// the prepared statement.
func WebitelDBPrepareStmtName(val string) attribute.KeyValue {
	return WebitelDBPrepareStmtNameKey.String(val)
}

// WebitelDBUser returns an attribute KeyValue conforming to the
// "webitel.db.user" semantic conventions. It represents the database user the
// connection authenticates as.
func WebitelDBUser(val string) attribute.KeyValue {
	return WebitelDBUserKey.String(val)
}

// WebitelHealthCheckName returns an attribute KeyValue conforming to the
// "webitel.health.check.name" semantic conventions. It represents the name the
// check was registered under.
func WebitelHealthCheckName(val string) attribute.KeyValue {
	return WebitelHealthCheckNameKey.String(val)
}

// Enum values for webitel.health.check.group
var (
	// Is this process wedged; counts towards readiness.
	// Stability: development
	WebitelHealthCheckGroupLiveness = WebitelHealthCheckGroupKey.String("liveness")
	// A node-local failure that takes the node out of rotation.
	// Stability: development
	WebitelHealthCheckGroupCritical = WebitelHealthCheckGroupKey.String("critical")
	// Only degrades; the node stays in rotation.
	// Stability: development
	WebitelHealthCheckGroupInformational = WebitelHealthCheckGroupKey.String("informational")
)

// Enum values for webitel.health.check.status
var (
	// The check is passing.
	// Stability: development
	WebitelHealthCheckStatusOk = WebitelHealthCheckStatusKey.String("ok")
	// The check has failed its threshold.
	// Stability: development
	WebitelHealthCheckStatusFail = WebitelHealthCheckStatusKey.String("fail")
)
