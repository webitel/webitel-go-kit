package pgw

import "fmt"

type Error interface {
	error
	pgwError()
}

type ConnectionExceptionError struct {
	Code   string
	Detail string
}

func (e *ConnectionExceptionError) Error() string {
	return fmt.Sprintf("code: %s, detail: %s", e.Code, e.Detail)
}

func (e *ConnectionExceptionError) pgwError() {}

type DataExceptionError struct {
	Schema string
	Column string
	Table  string
	Detail string
}

func (e *DataExceptionError) Error() string {
	return fmt.Sprintf("schema: %s, table: %s, column: %s, detail: %s", e.Schema, e.Table, e.Column, e.Detail)
}

func (e *DataExceptionError) pgwError() {}
