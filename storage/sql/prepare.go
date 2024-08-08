package sql

import (
	"context"
	"database/sql/driver"
)

const (
	traceMethodPrepare = "db.Prepare"
)

type prepareContextFuncMiddleware = middleware[prepareContextFunc]

type prepareContextFunc func(ctx context.Context, query string) (driver.Stmt, error)

// nopPrepareContext prepares nothing.
func nopPrepareContext(_ context.Context, _ string) (driver.Stmt, error) {
	return nil, nil
}

func ensurePrepareContext(conn driver.Conn) prepareContextFunc {
	if p, ok := conn.(driver.ConnPrepareContext); ok {
		return p.PrepareContext
	}

	return func(_ context.Context, query string) (driver.Stmt, error) {
		return conn.Prepare(query)
	}
}

// prepareTrace creates a span for prepare.
func prepareTrace(t MethodTracer) prepareContextFuncMiddleware {
	return func(next prepareContextFunc) prepareContextFunc {
		return func(ctx context.Context, query string) (stmt driver.Stmt, err error) {
			ctx, end := t.StartTrace(ctx, traceMethodPrepare, query, nil)
			defer end(err)

			return next(ctx, query)
		}
	}
}

func prepareWrapResult(execFuncMiddlewares []execContextFuncMiddleware, execContextFuncMiddlewares []execContextFuncMiddleware, queryFuncMiddlewares []queryContextFuncMiddleware, queryContextFuncMiddlewares []queryContextFuncMiddleware) prepareContextFuncMiddleware {
	return func(next prepareContextFunc) prepareContextFunc {
		return func(ctx context.Context, query string) (driver.Stmt, error) {
			stmt, err := next(ctx, query)
			if err != nil {
				return nil, err
			}

			return wrapStmt(stmt, stmtConfig{
				query:                       query,
				execFuncMiddlewares:         execFuncMiddlewares,
				queryContextFuncMiddlewares: queryContextFuncMiddlewares,
				execContextFuncMiddlewares:  execContextFuncMiddlewares,
				queryFuncMiddlewares:        queryFuncMiddlewares,
			}), nil
		}
	}
}

type prepareConfig struct {
	execFuncMiddlewares         []execContextFuncMiddleware
	execContextFuncMiddlewares  []execContextFuncMiddleware
	queryFuncMiddlewares        []queryContextFuncMiddleware
	queryContextFuncMiddlewares []queryContextFuncMiddleware
}

func makePrepareContextFuncMiddlewares(t MethodTracer, cfg prepareConfig) []prepareContextFuncMiddleware {
	return []prepareContextFuncMiddleware{
		prepareTrace(t),
		prepareWrapResult(
			cfg.execFuncMiddlewares,
			cfg.execContextFuncMiddlewares,
			cfg.queryFuncMiddlewares,
			cfg.queryContextFuncMiddlewares,
		),
	}
}
