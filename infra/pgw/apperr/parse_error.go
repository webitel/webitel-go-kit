// Package apperr maps PostgreSQL and pgw errors onto coded pkg/errors.
package apperr

import (
	stderrors "errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/infra/pgw"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// ParseError maps a database or pgw error to a coded application error.
func ParseError(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := errors.Lookup(err, errors.ErrKeyCode); ok {
		return err
	}

	if stderrors.Is(err, pgx.ErrNoRows) {
		return notFound(err)
	}

	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) {
		return fromPgError(err, pgErr)
	}

	var dataErr *pgw.DataExceptionError
	if stderrors.As(err, &dataErr) {
		return fromDataException(err, dataErr)
	}

	return internalError(err)
}

func fromPgError(err error, pgErr *pgconn.PgError) error {
	switch pgErr.Code {
	case pgerrcode.UniqueViolation:
		return uniqueViolation(err, pgErr)
	case pgerrcode.ForeignKeyViolation:
		return foreignKeyViolation(err, pgErr)
	case pgerrcode.CheckViolation:
		return checkViolation(err, pgErr)
	case pgerrcode.NotNullViolation:
		return notNullViolation(err, pgErr)
	case pgerrcode.SerializationFailure, pgerrcode.DeadlockDetected:
		return transactionAborted(err)
	case pgerrcode.ProgramLimitExceeded:
		return programLimitExceeded(err)
	default:
		return internalError(err)
	}
}

func notFound(err error) error {
	return errors.NotFound(
		"entity does not exist or access is denied",
		errors.WithID("pgw.not_found"),
		errors.WithCause(err),
	)
}

func uniqueViolation(err error, pgErr *pgconn.PgError) error {
	return errors.New(
		"invalid input: entity already exists",
		errors.WithCode(codes.AlreadyExists),
		errors.WithID("pgw.unique"),
		errors.WithCause(err),
		errors.WithValue("constraint", pgErr.ConstraintName),
	)
}

func foreignKeyViolation(err error, pgErr *pgconn.PgError) error {
	return errors.New(
		"invalid input: referenced entity does not exist or is still referenced",
		errors.WithCode(codes.FailedPrecondition),
		errors.WithID("pgw.foreign_key"),
		errors.WithCause(err),
		errors.WithValue("constraint", pgErr.ConstraintName),
		errors.WithValue("table", pgErr.TableName),
	)
}

func checkViolation(err error, pgErr *pgconn.PgError) error {
	return errors.InvalidArgument(
		"invalid input: value violates a constraint",
		errors.WithID("pgw.check"),
		errors.WithCause(err),
		errors.WithValue("constraint", pgErr.ConstraintName),
	)
}

func notNullViolation(err error, pgErr *pgconn.PgError) error {
	return errors.InvalidArgument(
		"invalid input: required value is missing",
		errors.WithID("pgw.not_null"),
		errors.WithCause(err),
		errors.WithValue("table", pgErr.TableName),
		errors.WithValue("column", pgErr.ColumnName),
	)
}

func fromDataException(err error, dataErr *pgw.DataExceptionError) error {
	return errors.InvalidArgument(
		"invalid input: value violates a constraint",
		errors.WithID("pgw.data"),
		errors.WithCause(err),
		errors.WithValue("schema", dataErr.Schema),
		errors.WithValue("table", dataErr.Table),
		errors.WithValue("column", dataErr.Column),
	)
}

func transactionAborted(err error) error {
	return errors.Aborted(
		"transaction conflict; retry the operation",
		errors.WithID("pgw.aborted"),
		errors.WithCause(err),
	)
}

func programLimitExceeded(err error) error {
	return errors.InvalidArgument(
		"invalid input: program limit exceeded",
		errors.WithID("pgw.program_limit"),
		errors.WithCause(err),
	)
}

func internalError(err error) error {
	return errors.Internal(
		"storage error",
		errors.WithID("pgw.internal"),
		errors.WithCause(err),
	)
}
