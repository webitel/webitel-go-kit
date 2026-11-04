package pgw

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	werror "github.com/webitel/webitel-go-kit/pkg/errors"
	"github.com/webitel/webitel-go-kit/pkg/safemap"
)

type errorsManager struct {
	notNullViolations    *safemap.SafeMap[string, *safemap.SafeMap[string, ErrorProcessor]]
	uniqueViolations     *safemap.SafeMap[string, ErrorProcessor]
	foreignKeyViolations *safemap.SafeMap[string, ErrorProcessor]
	checkViolations      *safemap.SafeMap[string, ErrorProcessor]
}

func newErrorsManager() *errorsManager {
	return &errorsManager{
		notNullViolations:    safemap.New[string, *safemap.SafeMap[string, ErrorProcessor]](nil),
		uniqueViolations:     safemap.New[string, ErrorProcessor](nil),
		foreignKeyViolations: safemap.New[string, ErrorProcessor](nil),
		checkViolations:      safemap.New[string, ErrorProcessor](nil),
	}
}

type ErrorProcessor func(originalError *pgconn.PgError) error

func (e *errorsManager) ParsePgError(err error) error {
	if err == nil {
		return nil
	}

	// already coded (registered processor or a previous pass)
	if _, ok := werror.Lookup(err, werror.ErrKeyCode); ok {
		return err
	}

	switch {
	case errors.Is(err, context.Canceled):
		return canceledError(err)
	case errors.Is(err, context.DeadlineExceeded):
		return deadlineExceededError(err)
	case errors.Is(err, pgx.ErrNoRows):
		return notFoundError(err)
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	if pgerrcode.IsConnectionException(pgErr.Code) {
		return connectionUnavailableError(err, pgErr)
	}

	if pgerrcode.IsDataException(pgErr.Code) {
		return dataExceptionError(err, pgErr)
	}

	if pgerrcode.IsProgramLimitExceeded(pgErr.Code) {
		return programLimitExceededError(err)
	}

	switch pgErr.Code {
	case pgerrcode.NotNullViolation:
		return e.parseNotNullViolation(err, pgErr)
	case pgerrcode.ForeignKeyViolation:
		return e.parseForeignKeyViolation(err, pgErr)
	case pgerrcode.UniqueViolation:
		return e.parseUniqueViolation(err, pgErr)
	case pgerrcode.CheckViolation:
		return e.parseCheckViolation(err, pgErr)
	case pgerrcode.SerializationFailure, pgerrcode.DeadlockDetected:
		return transactionAbortedError(err)
	default:
		return internalError(err)
	}
}

func (e *errorsManager) parseNotNullViolation(err error, pgErr *pgconn.PgError) error {
	if processor, ok := e.getNotNullViolation(pgErr.TableName, pgErr.ColumnName); ok {
		return processor(pgErr)
	}
	return notNullViolationError(err, pgErr)
}

func (e *errorsManager) parseForeignKeyViolation(err error, pgErr *pgconn.PgError) error {
	if processor, ok := e.getForeignKeyViolation(pgErr.ConstraintName); ok {
		return processor(pgErr)
	}
	return foreignKeyViolationError(err, pgErr)
}

func (e *errorsManager) parseUniqueViolation(err error, pgErr *pgconn.PgError) error {
	if processor, ok := e.getUniqueViolation(pgErr.ConstraintName); ok {
		return processor(pgErr)
	}
	return uniqueViolationError(err, pgErr)
}

func (e *errorsManager) parseCheckViolation(err error, pgErr *pgconn.PgError) error {
	if processor, ok := e.getCheckViolation(pgErr.ConstraintName); ok {
		return processor(pgErr)
	}
	return checkViolationError(err, pgErr)
}

func (e *errorsManager) RegisterNotNullViolationProcessor(table string, column string, processor ErrorProcessor) error {
	columnsMap, ok := e.notNullViolations.Get(table)
	if !ok {
		columnsMap = safemap.New[string, ErrorProcessor](nil)
		e.notNullViolations.Set(table, columnsMap)
	}

	if _, ok := columnsMap.Get(column); ok {
		return fmt.Errorf("column %s is already registered for not null violation on table %s", column, table)
	}

	columnsMap.Set(column, processor)
	return nil
}

func (e *errorsManager) getNotNullViolation(table string, column string) (ErrorProcessor, bool) {
	columnsMap, ok := e.notNullViolations.Get(table)
	if !ok {
		return nil, false
	}
	err, ok := columnsMap.Get(column)
	return err, ok
}

func (e *errorsManager) RegisterUniqueViolation(constraintName string, processor ErrorProcessor) error {
	e.uniqueViolations.Set(constraintName, processor)
	return nil
}

func (e *errorsManager) getUniqueViolation(constraintName string) (ErrorProcessor, bool) {
	err, ok := e.uniqueViolations.Get(constraintName)
	return err, ok
}

func (e *errorsManager) RegisterForeignKeyViolation(constraintName string, processor ErrorProcessor) error {
	e.foreignKeyViolations.Set(constraintName, processor)
	return nil
}

func (e *errorsManager) getForeignKeyViolation(constraintName string) (ErrorProcessor, bool) {
	err, ok := e.foreignKeyViolations.Get(constraintName)
	return err, ok
}

func (e *errorsManager) RegisterCheckViolation(constraintName string, processor ErrorProcessor) error {
	e.checkViolations.Set(constraintName, processor)
	return nil
}

func (e *errorsManager) getCheckViolation(constraintName string) (ErrorProcessor, bool) {
	err, ok := e.checkViolations.Get(constraintName)
	return err, ok
}
