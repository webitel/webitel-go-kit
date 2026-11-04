package pgw

import (
	"github.com/jackc/pgx/v5/pgconn"
	werror "github.com/webitel/webitel-go-kit/pkg/errors"
)

func notFoundError(err error) error {
	return werror.NotFound(
		"entity does not exist or access is denied",
		werror.WithID("pgw.not_found"),
		werror.WithCause(err),
	)
}

func uniqueViolationError(err error, pgErr *pgconn.PgError) error {
	return werror.AlreadyExists(
		"invalid input: entity already exists",
		werror.WithID("pgw.unique"),
		werror.WithCause(err),
		werror.WithValue("constraint", pgErr.ConstraintName),
	)
}

func foreignKeyViolationError(err error, pgErr *pgconn.PgError) error {
	return werror.FailedPrecondition(
		"invalid input: referenced entity does not exist or is still referenced",
		werror.WithID("pgw.foreign_key"),
		werror.WithCause(err),
		werror.WithValue("constraint", pgErr.ConstraintName),
		werror.WithValue("table", pgErr.TableName),
	)
}

func checkViolationError(err error, pgErr *pgconn.PgError) error {
	return werror.InvalidArgument(
		"invalid input: value violates a constraint",
		werror.WithID("pgw.check"),
		werror.WithCause(err),
		werror.WithValue("constraint", pgErr.ConstraintName),
	)
}

func notNullViolationError(err error, pgErr *pgconn.PgError) error {
	return werror.InvalidArgument(
		"invalid input: required value is missing",
		werror.WithID("pgw.not_null"),
		werror.WithCause(err),
		werror.WithValue("table", pgErr.TableName),
		werror.WithValue("column", pgErr.ColumnName),
	)
}

func dataExceptionError(err error, pgErr *pgconn.PgError) error {
	return werror.InvalidArgument(
		"invalid input: value violates a constraint",
		werror.WithID("pgw.data"),
		werror.WithCause(err),
		werror.WithValue("schema", pgErr.SchemaName),
		werror.WithValue("table", pgErr.TableName),
		werror.WithValue("column", pgErr.ColumnName),
	)
}

func connectionUnavailableError(err error, pgErr *pgconn.PgError) error {
	return werror.Unavailable(
		"database connection error",
		werror.WithID("pgw.connection"),
		werror.WithCause(err),
		werror.WithValue("sqlstate", pgErr.Code),
	)
}

func transactionAbortedError(err error) error {
	return werror.Aborted(
		"transaction conflict; retry the operation",
		werror.WithID("pgw.aborted"),
		werror.WithCause(err),
	)
}

func programLimitExceededError(err error) error {
	return werror.InvalidArgument(
		"invalid input: program limit exceeded",
		werror.WithID("pgw.program_limit"),
		werror.WithCause(err),
	)
}

func canceledError(err error) error {
	return werror.Canceled(
		"operation canceled",
		werror.WithID("pgw.canceled"),
		werror.WithCause(err),
	)
}

func deadlineExceededError(err error) error {
	return werror.DeadlineExceeded(
		"operation deadline exceeded",
		werror.WithID("pgw.deadline_exceeded"),
		werror.WithCause(err),
	)
}

func internalError(err error) error {
	return werror.Internal(
		"storage error",
		werror.WithID("pgw.internal"),
		werror.WithCause(err),
	)
}
