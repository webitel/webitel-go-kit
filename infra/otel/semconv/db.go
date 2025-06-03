package semconv

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	// DBRowsAffectedKey represents the number of rows affected.
	DBRowsAffectedKey = attribute.Key("db.rows_affected")

	// DBQueryParametersKey represents the query parameters.
	DBQueryParametersKey = attribute.Key("db.query.parameters")

	// DBBatchSizeKey represents the batch size.
	DBBatchSizeKey = attribute.Key("db.batch.size")

	// DBPrepareStmtNameKey represents the prepared statement name.
	DBPrepareStmtNameKey = attribute.Key("db.prepare_stmt.name")

	// DBSQLStateKey represents PostgreSQL error code,
	// see https://www.postgresql.org/docs/current/errcodes-appendix.html.
	DBSQLStateKey  = attribute.Key("db.sql_state")
	DBUserKey      = attribute.Key("db.user")
	DBStatementKey = attribute.Key("db.statement")
	DBSQLTableKey  = attribute.Key("db.sql_table")
)

var (
	// DBSystemPostgreSQL It represents the PostgreSQL as identified by the client instrumentation.
	DBSystemPostgreSQL = semconv.DBSystemPostgreSQL
)
