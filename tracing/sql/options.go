package sql

import (
	"context"
	"database/sql/driver"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/webitel/webitel-go-kit/semconv"
	"github.com/webitel/webitel-go-kit/tracing/internal"
)

type spanNameFormatter func(ctx context.Context, op, query string) string

type errorToSpanStatus func(err error) (codes.Code, string)

type queryTracer func(ctx context.Context, query string, args []driver.NamedValue) []attribute.KeyValue

// Option specifies instrumentation configuration options.
type Option interface {
	apply(c *Tracer)
}

type optionFunc func(*Tracer)

func (o optionFunc) apply(c *Tracer) {
	o(c)
}

// WithTracerProvider sets tracer provider.
func WithTracerProvider(p trace.TracerProvider) Option {
	return optionFunc(func(c *Tracer) {
		c.tracerProvider = p
	})
}

// // WithInstanceName sets database instance name.
// func WithInstanceName(instanceName string) Option {
// 	return WithDefaultAttributes(dbInstance.String(instanceName))
// }

// WithSystem sets database system name.
// See: semconv.DBSystemKey.
func WithSystem(system attribute.KeyValue) Option {
	return WithDefaultAttributes(semconv.DBSystemPostgreSQL)
}

// // WithDatabaseName sets database name.
// func WithDatabaseName(system string) Option {
// 	return WithDefaultAttributes(semconv.DBNameKey.String(system))
// }

// WithDefaultAttributes will be set to each span as default.
func WithDefaultAttributes(attrs ...attribute.KeyValue) Option {
	return optionFunc(func(o *Tracer) {
		o.attributes = append(o.attributes, attrs...)
	})
}

func WithSpanNameFormatter(f spanNameFormatter) Option {
	return optionFunc(func(o *Tracer) {
		o.formatSpanName = f
	})
}

func WithErrorToSpanStatus(f errorToSpanStatus) Option {
	return optionFunc(func(o *Tracer) {
		o.errorToStatus = f
	})
}

// TraceQuery sets a custom function that will return a list of attributes to add to the spans with a given query and args.
//
// For example:
//
//	otelsql.TraceQuery(func(sql string, args []driver.NamedValue) []attribute.KeyValue {
//		attrs := make([]attribute.KeyValue, 0)
//		attrs = append(attrs, semconv.DBStatementKey.String(sql))
//
//		for _, arg := range args {
//			if arg.Name != "password" {
//				attrs = append(attrs, sqlattribute.FromNamedValue(arg))
//			}
//		}
//
//		return attrs
//	})
func TraceQuery(f queryTracer) Option {
	return optionFunc(func(o *Tracer) {
		o.queryTracer = f
	})
}

// TraceQueryWithArgs will add to the spans the given sql query and all arguments.
func TraceQueryWithArgs() Option {
	return TraceQuery(traceQueryWithArgs)
}

// TraceQueryWithoutArgs will add to the spans the given sql query without any arguments.
func TraceQueryWithoutArgs() Option {
	return TraceQuery(traceQueryWithoutArgs)
}

func formatSpanName(_ context.Context, method, query string) string {
	qop := internal.SQLOperationName(query)

	var sb strings.Builder
	sb.Grow(len(method) + len(qop))
	sb.WriteString(method)
	sb.WriteString(".")
	sb.WriteString(qop)

	return sb.String()
}

func spanStatusFromError(err error) (codes.Code, string) {
	if err == nil {
		return codes.Ok, ""
	}

	return codes.Error, err.Error()
}

func traceNoQuery(context.Context, string, []driver.NamedValue) []attribute.KeyValue {
	return nil
}

func traceQueryWithoutArgs(_ context.Context, sql string, _ []driver.NamedValue) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.DBStatementKey.String(sql),
	}
}

func traceQueryWithArgs(_ context.Context, sql string, args []driver.NamedValue) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 1+len(args))
	attrs = append(attrs, semconv.DBStatementKey.String(sql))
	for _, arg := range args {
		attrs = append(attrs, internal.FromNamedValue(arg))
	}

	return attrs
}
