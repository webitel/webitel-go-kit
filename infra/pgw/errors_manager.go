package pgw

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
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

	var pgErr *pgconn.PgError
	ok := errors.As(err, &pgErr)
	if !ok {
		return err
	}

	switch pgErr.Code {
	case "08000", "08006":
		// connection exceptions
		return e.parseConnectionException(pgErr)
	case "23502":
		// data exception: not null violation
		return e.parseNotNullViolation(pgErr)
	case "23503":
		// data exception: foreign key violation
		return e.parseForeignKeyViolation(pgErr)
	case "23505":
		// data exception: unique violation
		return e.parseUniqueViolation(pgErr)
	case "23514":
		// data exception: check violation
		return e.parseCheckViolation(pgErr)
	}

	return err
}

func (e *errorsManager) parseConnectionException(pgErr *pgconn.PgError) error {
	return &ConnectionExceptionError{
		Code:   pgErr.Code,
		Detail: pgErr.Message,
	}
}

func (e *errorsManager) parseNotNullViolation(pgErr *pgconn.PgError) error {
	processor, ok := e.getNotNullViolation(pgErr.TableName, pgErr.ColumnName)
	if ok {
		return processor(pgErr)
	}
	return &DataExceptionError{
		Schema: pgErr.SchemaName,
		Table:  pgErr.TableName,
		Column: pgErr.ColumnName,
		Detail: pgErr.Message,
	}
}

func (e *errorsManager) parseForeignKeyViolation(pgErr *pgconn.PgError) error {
	processor, ok := e.getForeignKeyViolation(pgErr.ConstraintName)
	if ok {
		return processor(pgErr)
	}
	return &DataExceptionError{
		Schema: pgErr.SchemaName,
		Table:  pgErr.TableName,
		Column: pgErr.ColumnName,
		Detail: pgErr.Message,
	}
}

func (e *errorsManager) parseUniqueViolation(pgErr *pgconn.PgError) error {
	processor, ok := e.getUniqueViolation(pgErr.ConstraintName)
	if ok {
		return processor(pgErr)
	}
	return &DataExceptionError{
		Schema: pgErr.SchemaName,
		Table:  pgErr.TableName,
		Column: pgErr.ColumnName,
		Detail: pgErr.Message,
	}
}

func (e *errorsManager) parseCheckViolation(pgErr *pgconn.PgError) error {
	processor, ok := e.getCheckViolation(pgErr.ConstraintName)
	if ok {
		return processor(pgErr)
	}
	return &DataExceptionError{
		Schema: pgErr.SchemaName,
		Table:  pgErr.TableName,
		Column: pgErr.ColumnName,
		Detail: pgErr.Message,
	}
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
