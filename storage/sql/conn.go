package sql

import (
	"context"
	"database/sql/driver"
	"errors"
)

type connConfig struct {
	pingFuncMiddlewares         []pingFuncMiddleware
	execContextFuncMiddlewares  []execContextFuncMiddleware
	queryContextFuncMiddlewares []queryContextFuncMiddleware
	beginFuncMiddlewares        []beginFuncMiddleware
	prepareFuncMiddlewares      []prepareContextFuncMiddleware
}

type conn struct {
	ping    pingFunc
	exec    execContextFunc
	query   queryContextFunc
	begin   beginFunc
	prepare prepareContextFunc

	close func() error
}

func (c conn) Ping(ctx context.Context) error {
	return c.ping(ctx)
}

// Deprecated: Drivers should implement ExecerContext instead.
func (c conn) Exec(string, []driver.Value) (driver.Result, error) {
	return nil, errors.New("method is deprecated")
}

func (c conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.exec(ctx, query, args)
}

// Deprecated: Drivers should implement QueryerContext instead.
func (c conn) Query(string, []driver.Value) (driver.Rows, error) {
	return nil, errors.New("method is deprecated")
}

func (c conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.query(ctx, query, args)
}

func (c conn) Prepare(query string) (driver.Stmt, error) {
	return c.prepare(context.Background(), query)
}

func (c conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return c.prepare(ctx, query)
}

func (c conn) Begin() (driver.Tx, error) {
	return c.begin(context.Background(), driver.TxOptions{})
}

func (c conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.begin(ctx, opts)
}

func (c conn) Close() error {
	return c.close()
}

func wrapConn(parent driver.Conn, opt connConfig) driver.Conn {
	c := makeConn(parent, opt)

	var (
		n, hasNameValueChecker = parent.(driver.NamedValueChecker)
		s, hasSessionResetter  = parent.(driver.SessionResetter)
	)

	switch {
	default:
		// case !hasNameValueChecker && !hasSessionResetter:
		return c

	case hasNameValueChecker && !hasSessionResetter:
		return struct {
			conn
			driver.NamedValueChecker
		}{c, n}

	case !hasNameValueChecker && hasSessionResetter:
		return struct {
			conn
			driver.SessionResetter
		}{c, s}

	case hasNameValueChecker && hasSessionResetter:
		return struct {
			conn
			driver.NamedValueChecker
			driver.SessionResetter
		}{c, n, s}
	}
}

func makeConn(parent driver.Conn, cfg connConfig) conn {
	c := conn{
		ping:  nopPing,
		exec:  skippedExecContext,
		query: skippedQueryContext,
		close: parent.Close,
	}

	if p, ok := parent.(driver.Pinger); ok {
		c.ping = chainMiddlewares(cfg.pingFuncMiddlewares, p.Ping)
	}

	if p, ok := parent.(driver.ExecerContext); ok {
		c.exec = chainMiddlewares(cfg.execContextFuncMiddlewares, p.ExecContext)
	}

	if p, ok := parent.(driver.QueryerContext); ok {
		c.query = chainMiddlewares(cfg.queryContextFuncMiddlewares, p.QueryContext)
	}

	c.begin = chainMiddlewares(cfg.beginFuncMiddlewares, ensureBegin(parent))
	c.prepare = chainMiddlewares(cfg.prepareFuncMiddlewares, ensurePrepareContext(parent))

	return c
}

type middleware[T any] func(next T) T

// chainMiddlewares builds an inline middleware stack in the order they are passed.
func chainMiddlewares[T any](middlewares []middleware[T], last T) T {
	// return ahead of time if there are not any middlewares for the chain
	if len(middlewares) == 0 {
		return last
	}

	// wrap the end execer with the middleware chain
	h := middlewares[len(middlewares)-1](last)
	for i := len(middlewares) - 2; i >= 0; i-- {
		h = middlewares[i](h)
	}

	return h
}
