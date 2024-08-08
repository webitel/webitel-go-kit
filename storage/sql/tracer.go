package sql

import (
	"context"
	"database/sql/driver"
)

type MethodTracer interface {
	ShouldTrace(ctx context.Context) bool
	StartTrace(ctx context.Context, method string, query string, args []driver.NamedValue) (context.Context, func(err error))
}

type noopTracer struct{}

func (n *noopTracer) ShouldTrace(ctx context.Context) bool {
	return false
}

func (n *noopTracer) StartTrace(ctx context.Context, method string, query string, args []driver.NamedValue) (context.Context, func(err error)) {
	return ctx, func(err error) {}
}

var _ MethodTracer = &noopTracer{}
