package sql

import (
	"context"
	"database/sql/driver"
)

const (
	metricMethodExec = "go.sql.exec"
	traceMethodExec  = "db.Exec"

	metricMethodQuery = "go.sql.query"
	traceMethodQuery  = "db.Query"
)

type (
	execContextFuncMiddleware = middleware[execContextFunc]
	execContextFunc           func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error)

	queryContextFuncMiddleware = middleware[queryContextFunc]
	queryContextFunc           func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error)
)

// nopExecContext executes nothing.
func nopExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	return nil, nil
}

// skippedExecContext always returns driver.ErrSkip.
func skippedExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	return nil, driver.ErrSkip
}

// nopQueryContext queries nothing.
func nopQueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	return nil, nil
}

// skippedQueryContext always returns driver.ErrSkip.
func skippedQueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	return nil, driver.ErrSkip
}

// execTrace creates a span for exec.
func execTrace(t MethodTracer, method string) execContextFuncMiddleware {
	return func(next execContextFunc) execContextFunc {
		return func(ctx context.Context, query string, args []driver.NamedValue) (result driver.Result, err error) {
			ctx = ContextWithQuery(ctx, query)
			ctx, end := t.StartTrace(ctx, method, query, args)
			defer end(err)

			return next(ctx, query, args)
		}
	}
}

func execWrapResult(t MethodTracer, traceLastInsertID bool, traceRowsAffected bool) execContextFuncMiddleware {
	return func(next execContextFunc) execContextFunc {
		return func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			result, err := next(ctx, query, args)
			if err != nil {
				return nil, err
			}

			return wrapResult(ctx, result, t, traceLastInsertID, traceRowsAffected), nil
		}
	}
}

// queryTrace creates a span for query.
func queryTrace(t MethodTracer, method string) queryContextFuncMiddleware {
	return func(next queryContextFunc) queryContextFunc {
		return func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			var err error

			ctx, end := t.StartTrace(ctx, method, query, args)
			defer end(err)

			r, err := next(ctx, query, args)

			return r, err
		}
	}
}

func makeExecContextFuncMiddlewares(t MethodTracer, cfg execConfig) []execContextFuncMiddleware {
	middlewares := make([]middleware[execContextFunc], 0, 3)
	middlewares = append(middlewares, execTrace(t, cfg.traceMethod))
	if cfg.traceLastInsertID || cfg.traceRowsAffected {
		middlewares = append(middlewares, execWrapResult(t, cfg.traceLastInsertID, cfg.traceRowsAffected))
	}

	return middlewares
}

type execConfig struct {
	metricMethod      string
	traceMethod       string
	traceLastInsertID bool
	traceRowsAffected bool
}

func newExecConfig(c *config, metricMethod, traceMethod string) execConfig {
	return execConfig{
		metricMethod:      metricMethod,
		traceMethod:       traceMethod,
		traceLastInsertID: c.lastInsertID,
		traceRowsAffected: c.rowsAffected,
	}
}

func makeQueryerContextMiddlewares(t MethodTracer, cfg queryConfig) []queryContextFuncMiddleware {
	middlewares := make([]queryContextFuncMiddleware, 0, 3)
	if t != nil {
		middlewares = append(middlewares, queryTrace(t, cfg.traceMethod))
	}

	return middlewares
}

type queryConfig struct {
	metricMethod   string
	traceMethod    string
	traceRowsNext  bool
	traceRowsClose bool
}

func newQueryConfig(c *config, metricMethod, traceMethod string) queryConfig {
	return queryConfig{
		metricMethod:   metricMethod,
		traceMethod:    traceMethod,
		traceRowsNext:  c.rowsNext,
		traceRowsClose: c.rowsClose,
	}
}
