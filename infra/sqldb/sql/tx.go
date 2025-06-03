package sql

import (
	"context"
	"database/sql/driver"

	"go.opentelemetry.io/otel/trace"
)

const (
	traceMethodBegin    = "db.Begin"
	traceMethodCommit   = "db.Commit"
	traceMethodRollback = "db.Rollback"
)

var _ driver.Tx = (*tx)(nil)

type (
	beginFuncMiddleware = middleware[beginFunc]
	beginFunc           func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error)

	txFuncMiddleware = middleware[txFunc]
	txFunc           func() error
)

type tx struct {
	commit   txFunc
	rollback txFunc
}

func (t tx) Commit() error {
	return t.commit()
}

func (t tx) Rollback() error {
	return t.rollback()
}

// nopBegin pings nothing.
func nopBegin(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return nil, nil
}

func nopTxFunc() error {
	return nil
}

func ensureBegin(conn driver.Conn) beginFunc {
	if b, ok := conn.(driver.ConnBeginTx); ok {
		return b.BeginTx
	}

	return func(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
		return conn.Begin() // nolint: staticcheck
	}
}

// beginTrace traces begin.
func beginTrace(t MethodTracer) beginFuncMiddleware {
	return func(next beginFunc) beginFunc {
		return func(ctx context.Context, opts driver.TxOptions) (result driver.Tx, err error) {
			ctx, end := t.StartTrace(ctx, traceMethodBegin, "", nil)
			defer end(err)

			return next(ctx, opts)
		}
	}
}

func beginWrapTx(t MethodTracer) beginFuncMiddleware {
	return func(next beginFunc) beginFunc {
		return func(ctx context.Context, opts driver.TxOptions) (result driver.Tx, err error) {
			tx, err := next(ctx, opts)
			if err != nil {
				return nil, err
			}

			return wrapTx(ctx, tx, t), nil
		}
	}
}

func wrapTx(ctx context.Context, parent driver.Tx, t MethodTracer) driver.Tx {
	ctx = trace.ContextWithSpanContext(context.Background(), trace.SpanContextFromContext(ctx))

	return &tx{
		commit:   chainMiddlewares(makeTxFuncMiddlewares(ctx, t, traceMethodCommit), parent.Commit),
		rollback: chainMiddlewares(makeTxFuncMiddlewares(ctx, t, traceMethodRollback), parent.Rollback),
	}
}

func txTrace(ctx context.Context, t MethodTracer, method string) txFuncMiddleware {
	return func(next txFunc) txFunc {
		return func() (err error) {
			_, end := t.StartTrace(ctx, method, "", nil)
			defer end(err)

			return next()
		}
	}
}

func makeBeginFuncMiddlewares(t MethodTracer) []beginFuncMiddleware {
	return []beginFuncMiddleware{
		beginTrace(t), beginWrapTx(t),
	}
}

func makeTxFuncMiddlewares(ctx context.Context, t MethodTracer, traceMethod string) []txFuncMiddleware {
	middlewares := make([]txFuncMiddleware, 0, 2)
	if t != nil {
		middlewares = append(middlewares, txTrace(ctx, t, traceMethod))
	}

	return middlewares
}
