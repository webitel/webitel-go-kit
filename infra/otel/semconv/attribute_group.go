// Copyright (c) 2024 Webitel
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
