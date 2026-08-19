package pgx

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	semconv2 "github.com/webitel/webitel-go-kit/infra/otel/semconv"
)

const (
	scopeName = "github.com/webitel/webitel-go-kit/infra/otel/instrumentation/pgx"
)

// Version is the current release version of the gRPC instrumentation.
func Version() string {
	return "0.0.0"
}

// Tracer is a wrapper around the pgx tracer interfaces which instrument
// queries.
type Tracer struct {
	tracer              trace.Tracer
	attrs               []attribute.KeyValue
	trimQuerySpanName   bool
	spanNameFunc        SpanNameFunc
	prefixQuerySpanName bool
	logSQLStatement     bool
	includeParams       bool
}

type tracerConfig struct {
	tp                  trace.TracerProvider
	attrs               []attribute.KeyValue
	trimQuerySpanName   bool
	spanNameFunc        SpanNameFunc
	prefixQuerySpanName bool
	logSQLStatement     bool
	includeParams       bool
}

// NewTracer returns a new Tracer.
func NewTracer(opts ...Option) *Tracer {
	cfg := &tracerConfig{
		tp: otel.GetTracerProvider(),
		attrs: []attribute.KeyValue{
			semconv2.DBSystemNamePostgresql,
		},
		trimQuerySpanName:   false,
		spanNameFunc:        nil,
		prefixQuerySpanName: true,
		logSQLStatement:     true,
		includeParams:       false,
	}

	for _, opt := range opts {
		opt.apply(cfg)
	}

	return &Tracer{
		tracer:              cfg.tp.Tracer(scopeName, trace.WithInstrumentationVersion(Version())),
		attrs:               cfg.attrs,
		trimQuerySpanName:   cfg.trimQuerySpanName,
		spanNameFunc:        cfg.spanNameFunc,
		prefixQuerySpanName: cfg.prefixQuerySpanName,
		logSQLStatement:     cfg.logSQLStatement,
		includeParams:       cfg.includeParams,
	}
}

func recordError(span trace.Span, err error) {
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			span.SetAttributes(semconv2.DBResponseStatusCodeKey.String(pgErr.Code))
		}
	}
}

// connectionAttributesFromConfig returns a slice of SpanStartOptions that contain
// attributes from the given connection config.
func connectionAttributesFromConfig(config *pgx.ConnConfig) []trace.SpanStartOption {
	if config != nil {
		return []trace.SpanStartOption{
			trace.WithAttributes(
				semconv2.ServerAddressKey.String(config.Host),
				semconv2.ServerPortKey.Int(int(config.Port)),
				semconv2.WebitelDBUserKey.String(config.User),
			),
		}
	}

	return nil
}

// TraceQueryStart is called at the beginning of Query, QueryRow, and Exec calls.
// The returned context is used for the rest of the call and will be passed to TraceQueryEnd.
func (t *Tracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.attrs...),
	}

	if conn != nil {
		opts = append(opts, connectionAttributesFromConfig(conn.Config())...)
	}

	if t.logSQLStatement {
		opts = append(opts, trace.WithAttributes(semconv2.DBQueryTextKey.String(data.SQL)))
		if t.includeParams {
			opts = append(opts, trace.WithAttributes(makeParamsAttributes(data.Args)...))
		}
	}

	spanName := data.SQL
	if t.trimQuerySpanName {
		spanName = SQLOperationName(data.SQL)
	}

	if t.prefixQuerySpanName {
		spanName = "db.Query." + spanName
	}

	ctx, _ = t.tracer.Start(ctx, spanName, opts...)

	return ctx
}

// TraceQueryEnd is called at the end of Query, QueryRow, and Exec calls.
func (t *Tracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span := trace.SpanFromContext(ctx)
	recordError(span, data.Err)

	if data.Err == nil {
		span.SetAttributes(rowCountAttribute(data.CommandTag))
	}

	span.End()
}

// TraceCopyFromStart is called at the beginning of CopyFrom calls. The
// returned context is used for the rest of the call and will be passed to
// TraceCopyFromEnd.
func (t *Tracer) TraceCopyFromStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.attrs...),
		trace.WithAttributes(semconv2.DBCollectionNameKey.String(data.TableName.Sanitize())),
	}

	if conn != nil {
		opts = append(opts, connectionAttributesFromConfig(conn.Config())...)
	}

	ctx, _ = t.tracer.Start(ctx, "db.CopyFrom."+data.TableName.Sanitize(), opts...)

	return ctx
}

// TraceCopyFromEnd is called at the end of CopyFrom calls.
func (t *Tracer) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	span := trace.SpanFromContext(ctx)
	recordError(span, data.Err)

	if data.Err == nil {
		// COPY reports rows copied, never rows returned.
		span.SetAttributes(semconv2.WebitelDBRowsAffectedKey.Int64(data.CommandTag.RowsAffected()))
	}

	span.End()
}

// TraceBatchStart is called at the beginning of SendBatch calls. The returned
// context is used for the rest of the call and will be passed to
// TraceBatchQuery and TraceBatchEnd.
func (t *Tracer) TraceBatchStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	var size int
	if b := data.Batch; b != nil {
		size = b.Len()
	}

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.attrs...),
	}

	// Only a real batch. The convention says the value SHOULD never be 1.
	if size > 1 {
		opts = append(opts, trace.WithAttributes(semconv2.DBOperationBatchSizeKey.Int(size)))
	}

	if conn != nil {
		opts = append(opts, connectionAttributesFromConfig(conn.Config())...)
	}

	ctx, _ = t.tracer.Start(ctx, "db.BatchStart.", opts...)

	return ctx
}

// TraceBatchQuery is called at the after each query in a batch.
func (t *Tracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return
	}

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.attrs...),
	}

	if conn != nil {
		opts = append(opts, connectionAttributesFromConfig(conn.Config())...)
	}

	if t.logSQLStatement {
		opts = append(opts, trace.WithAttributes(semconv2.DBQueryTextKey.String(data.SQL)))
		if t.includeParams {
			opts = append(opts, trace.WithAttributes(makeParamsAttributes(data.Args)...))
		}
	}

	var spanName string
	if t.trimQuerySpanName {
		spanName = SQLOperationName(data.SQL)
		if t.prefixQuerySpanName {
			spanName = "db.Query." + spanName
		}
	} else {
		spanName = data.SQL
		if t.prefixQuerySpanName {
			spanName = "db.BatchQuery." + spanName
		}
	}

	_, span := t.tracer.Start(ctx, spanName, opts...)
	recordError(span, data.Err)

	span.End()
}

// TraceBatchEnd is called at the end of SendBatch calls.
func (t *Tracer) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	span := trace.SpanFromContext(ctx)
	recordError(span, data.Err)

	span.End()
}

// TraceConnectStart is called at the beginning of Connect and ConnectConfig
// calls. The returned context is used for the rest of the call and will be
// passed to TraceConnectEnd.
func (t *Tracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.attrs...),
	}

	if data.ConnConfig != nil {
		opts = append(opts, connectionAttributesFromConfig(data.ConnConfig)...)
	}

	ctx, _ = t.tracer.Start(ctx, "db.Connect", opts...)

	return ctx
}

// TraceConnectEnd is called at the end of Connect and ConnectConfig calls.
func (t *Tracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	span := trace.SpanFromContext(ctx)
	recordError(span, data.Err)

	span.End()
}

// TracePrepareStart is called at the beginning of Prepare calls. The returned
// context is used for the rest of the call and will be passed to TracePrepareEnd.
func (t *Tracer) TracePrepareStart(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.attrs...),
	}

	if data.Name != "" {
		opts = append(opts, trace.WithAttributes(semconv2.WebitelDBPrepareStmtNameKey.String(data.Name)))
	}

	if conn != nil {
		opts = append(opts, connectionAttributesFromConfig(conn.Config())...)
	}

	if t.logSQLStatement {
		opts = append(opts, trace.WithAttributes(semconv2.DBQueryTextKey.String(data.SQL)))
	}

	spanName := data.SQL
	if t.trimQuerySpanName {
		spanName = SQLOperationName(data.SQL)
	}

	if t.prefixQuerySpanName {
		spanName = "db.Prepare." + spanName
	}

	ctx, _ = t.tracer.Start(ctx, spanName, opts...)

	return ctx
}

// TracePrepareEnd is called at the end of Prepare calls.
func (t *Tracer) TracePrepareEnd(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareEndData) {
	span := trace.SpanFromContext(ctx)
	recordError(span, data.Err)

	span.End()
}

// db.operation.parameter is a template attribute: one attribute per parameter,
// keyed by name — or by 0-based index when the parameters are positional.
// rowCountAttribute picks the right attribute for a command tag. TraceQueryEnd
// covers Query, QueryRow and Exec, and pgx reports one number for all of them:
// rows returned for SELECT, rows changed for a write. Only the first is what
// db.response.returned_rows means.
func rowCountAttribute(tag pgconn.CommandTag) attribute.KeyValue {
	if tag.Select() {
		return semconv2.DBResponseReturnedRowsKey.Int64(tag.RowsAffected())
	}

	return semconv2.WebitelDBRowsAffectedKey.Int64(tag.RowsAffected())
}

func makeParamsAttributes(args []any) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, len(args))
	for i := range args {
		key := attribute.Key(string(semconv2.DBOperationParameterKey) + "." + strconv.Itoa(i))
		attrs[i] = key.String(fmt.Sprintf("%+v", args[i]))
	}

	return attrs
}
