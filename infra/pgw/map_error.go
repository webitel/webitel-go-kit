package pgw

import (
	"github.com/jackc/pgx/v5/pgconn"
	werror "github.com/webitel/webitel-go-kit/pkg/errors"
)

func notFoundError(err error) error {
	return werror.Wrap(ErrNotFound, werror.WithCause(err))
}

func uniqueViolationError(err error, pgErr *pgconn.PgError) error {
	return werror.Wrap(
		ErrUnique,
		werror.WithCause(err),
		werror.WithValue("constraint", pgErr.ConstraintName),
	)
}

func foreignKeyViolationError(err error, pgErr *pgconn.PgError) error {
	return werror.Wrap(
		ErrForeignKey,
		werror.WithCause(err),
		werror.WithValue("constraint", pgErr.ConstraintName),
		werror.WithValue("table", pgErr.TableName),
	)
}

func checkViolationError(err error, pgErr *pgconn.PgError) error {
	return werror.Wrap(
		ErrCheck,
		werror.WithCause(err),
		werror.WithValue("constraint", pgErr.ConstraintName),
	)
}

func notNullViolationError(err error, pgErr *pgconn.PgError) error {
	return werror.Wrap(
		ErrNotNull,
		werror.WithCause(err),
		werror.WithValue("table", pgErr.TableName),
		werror.WithValue("column", pgErr.ColumnName),
	)
}

func dataExceptionError(err error, pgErr *pgconn.PgError) error {
	return werror.Wrap(
		ErrData,
		werror.WithCause(err),
		werror.WithValue("schema", pgErr.SchemaName),
		werror.WithValue("table", pgErr.TableName),
		werror.WithValue("column", pgErr.ColumnName),
	)
}

func connectionUnavailableError(err error, pgErr *pgconn.PgError) error {
	return werror.Wrap(
		ErrConnection,
		werror.WithCause(err),
		werror.WithValue("sqlstate", pgErr.Code),
	)
}

func transactionAbortedError(err error) error {
	return werror.Wrap(ErrAborted, werror.WithCause(err))
}

func programLimitExceededError(err error) error {
	return werror.Wrap(ErrProgramLimit, werror.WithCause(err))
}

func canceledError(err error) error {
	return werror.Wrap(ErrCanceled, werror.WithCause(err))
}

func deadlineExceededError(err error) error {
	return werror.Wrap(ErrDeadlineExceeded, werror.WithCause(err))
}

func internalError(err error) error {
	return werror.Wrap(ErrInternal, werror.WithCause(err))
}
