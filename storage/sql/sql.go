package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strconv"
	"sync"
)

const _maxDriver = 150

const (
	dbSQLClientLatencyMs = "db.sql.client.latency"
	dbSQLClientCalls     = "db.sql.client.calls"

	unitDimensionless = "1"
	unitBytes         = "By"
	unitMilliseconds  = "ms"
)

var regMu sync.Mutex

// Register initializes and registers our otelsql wrapped database driver identified by its driverName and using provided
// options. On success, it returns the generated driverName to use when calling sql.Open.
//
// It is possible to register multiple wrappers for the same database driver if needing different options for
// different connections.
func Register(driverName string, options ...Option) (string, error) {
	return RegisterWithSource(driverName, "", options...)
}

// RegisterWithSource initializes and registers our otelsql wrapped database driver
// identified by its driverName, using provided options.
//
// source is useful if some drivers do not accept the empty string when opening the DB. On success, it returns the
// generated driverName to use when calling sql.Open.
//
// It is possible to register multiple wrappers for the same database driver if needing different options for
// different connections.
func RegisterWithSource(driverName string, source string, options ...Option) (string, error) {
	// retrieve the driver implementation we need to wrap with instrumentation
	db, err := sql.Open(driverName, source)
	if err != nil {
		return "", err
	}

	dri := db.Driver()
	if err = db.Close(); err != nil {
		return "", err
	}

	regMu.Lock()
	defer regMu.Unlock()

	// Since we might want to register multiple otelsql drivers to have different options, but potentially the same
	// underlying database driver, we cycle through to find available driver names.
	driverName += "-otelsql-"

	for i := int64(0); i < _maxDriver; i++ {
		found := false
		regName := driverName + strconv.FormatInt(i, 10)
		for _, name := range sql.Drivers() {
			if name == regName {
				found = true
			}
		}

		if !found {
			sql.Register(regName, Wrap(dri, options...))

			return regName, nil
		}
	}

	return "", errors.New("unable to register driver, all slots have been taken")
}

// Wrap takes a SQL driver and wraps it with OpenTelemetry instrumentation.
func Wrap(d driver.Driver, opts ...Option) driver.Driver {
	c := config{
		tracer: &noopTracer{},
	}

	for _, option := range opts {
		option.apply(&c)
	}

	return wrapDriver(d, &c)
}

func wrapDriver(d driver.Driver, c *config) driver.Driver {
	drv := otDriver{
		parent:     d,
		connConfig: newConnConfig(c),
		close:      func() error { return nil },
	}

	if _, ok := d.(driver.DriverContext); ok {
		return struct {
			driver.Driver
			driver.DriverContext
		}{drv, drv}
	}

	return struct{ driver.Driver }{drv}
}

func newConnConfig(c *config) connConfig {
	return connConfig{
		pingFuncMiddlewares:         makePingFuncMiddlewares(tracerOrNil(c.tracer, c.ping)),
		execContextFuncMiddlewares:  makeExecContextFuncMiddlewares(c.tracer, newExecConfig(c, metricMethodExec, traceMethodExec)),
		queryContextFuncMiddlewares: makeQueryerContextMiddlewares(c.tracer, newQueryConfig(c, metricMethodQuery, traceMethodQuery)),
		beginFuncMiddlewares:        makeBeginFuncMiddlewares(c.tracer),
		prepareFuncMiddlewares: makePrepareContextFuncMiddlewares(c.tracer, prepareConfig{
			execFuncMiddlewares:         makeExecContextFuncMiddlewares(tracerOrNil(c.tracer, c.allowRoot), newExecConfig(c, metricMethodStmtExec, traceMethodStmtExec)),
			execContextFuncMiddlewares:  makeExecContextFuncMiddlewares(c.tracer, newExecConfig(c, metricMethodStmtExec, traceMethodStmtExec)),
			queryFuncMiddlewares:        makeQueryerContextMiddlewares(tracerOrNil(c.tracer, c.allowRoot), newQueryConfig(c, metricMethodStmtQuery, traceMethodStmtQuery)),
			queryContextFuncMiddlewares: makeQueryerContextMiddlewares(c.tracer, newQueryConfig(c, metricMethodStmtQuery, traceMethodStmtQuery)),
		}),
	}
}

var _ driver.Driver = (*otDriver)(nil)

type otDriver struct {
	parent    driver.Driver
	connector driver.Connector
	close     func() error

	connConfig connConfig
}

func (d otDriver) Open(name string) (driver.Conn, error) {
	c, err := d.parent.Open(name)
	if err != nil {
		return nil, err
	}

	return wrapConn(c, d.connConfig), nil
}

func (d otDriver) Close() error {
	return d.close()
}

func (d otDriver) OpenConnector(name string) (driver.Connector, error) {
	var err error

	d.connector, err = d.parent.(driver.DriverContext).OpenConnector(name)
	if err != nil {
		return nil, err
	}

	if c, ok := d.connector.(io.Closer); ok {
		d.close = c.Close
	}

	return d, err
}

func (d otDriver) Connect(ctx context.Context) (driver.Conn, error) {
	c, err := d.connector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return wrapConn(c, d.connConfig), nil
}

func (d otDriver) Driver() driver.Driver {
	return d
}

func valuesToNamedValues(values []driver.Value) []driver.NamedValue {
	if values == nil {
		return nil
	}

	namedValues := make([]driver.NamedValue, len(values))
	for i, v := range values {
		namedValues[i] = driver.NamedValue{
			Ordinal: i + 1,
			Value:   v,
		}
	}

	return namedValues
}

func namedValuesToValues(namedValues []driver.NamedValue) []driver.Value {
	if namedValues == nil {
		return nil
	}

	values := make([]driver.Value, len(namedValues))
	for i, v := range namedValues {
		values[i] = v.Value
	}

	return values
}

func tracerOrNil(t MethodTracer, shouldTrace bool) MethodTracer {
	if shouldTrace {
		return t
	}

	return nil
}

type queryCtxKey struct{}

// QueryFromContext gets the query from context.
func QueryFromContext(ctx context.Context) string {
	query, ok := ctx.Value(queryCtxKey{}).(string)
	if !ok {
		return ""
	}

	return query
}

// ContextWithQuery attaches the query to the parent context.
func ContextWithQuery(ctx context.Context, query string) context.Context {
	return context.WithValue(ctx, queryCtxKey{}, query)
}
