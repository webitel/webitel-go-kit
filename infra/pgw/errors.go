package pgw

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

type Error interface {
	error
	pgwError()
}

type ConnectionExceptionError struct {
	Code   string
	Detail string

	pgErr *pgconn.PgError
}

func (e *ConnectionExceptionError) Error() string {
	return fmt.Sprintf("code: %s, detail: %s", e.Code, e.Detail)
}

func (e *ConnectionExceptionError) pgwError() {}

func (e *ConnectionExceptionError) Unwrap() error {
	if e.pgErr == nil { // a typed nil would look like a non-nil error to errors.As
		return nil
	}
	return e.pgErr
}

type DataExceptionError struct {
	Code           string // SQLSTATE
	ConstraintName string
	Schema         string
	Column         string
	Table          string
	Detail         string

	pgErr *pgconn.PgError
}

func (e *DataExceptionError) Error() string {
	return fmt.Sprintf("schema: %s, table: %s, column: %s, detail: %s", e.Schema, e.Table, e.Column, e.Detail)
}

func (e *DataExceptionError) pgwError() {}

func (e *DataExceptionError) Unwrap() error {
	if e.pgErr == nil {
		return nil
	}
	return e.pgErr
}
