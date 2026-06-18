package pgw

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Tracer is the tracing interface for pgw operations. Implement it to plug in
// OpenTelemetry or any other tracing backend and pass it via WithTracer.
type Tracer interface {
	// ShouldTrace returns true if the context warrants creating a trace span.
	ShouldTrace(ctx context.Context) bool
	// StartTrace begins a span for the given database method and SQL statement.
	// The returned finish function must be called with the operation error (nil on success).
	StartTrace(ctx context.Context, method, sql string, args []any) (context.Context, func(err error))
}

// pgxTracerAdapter bridges Tracer to pgx's native tracer interfaces so a single
// Tracer implementation covers queries, batches, COPY FROM, and prepared statements.
type pgxTracerAdapter struct {
	t Tracer
}

var (
	_ pgx.QueryTracer    = (*pgxTracerAdapter)(nil)
	_ pgx.BatchTracer    = (*pgxTracerAdapter)(nil)
	_ pgx.CopyFromTracer = (*pgxTracerAdapter)(nil)
	_ pgx.PrepareTracer  = (*pgxTracerAdapter)(nil)
)

type (
	ctxKeyQueryEnd   struct{}
	ctxKeyBatchEnd   struct{}
	ctxKeyCopyEnd    struct{}
	ctxKeyPrepareEnd struct{}
)

func (a *pgxTracerAdapter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if !a.t.ShouldTrace(ctx) {
		return ctx
	}
	ctx, end := a.t.StartTrace(ctx, "db.Query", data.SQL, data.Args)
	return context.WithValue(ctx, ctxKeyQueryEnd{}, end)
}

func (a *pgxTracerAdapter) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if end, ok := ctx.Value(ctxKeyQueryEnd{}).(func(error)); ok {
		end(data.Err)
	}
}

func (a *pgxTracerAdapter) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	if !a.t.ShouldTrace(ctx) {
		return ctx
	}
	ctx, end := a.t.StartTrace(ctx, "db.Batch", "", nil)
	return context.WithValue(ctx, ctxKeyBatchEnd{}, end)
}

func (a *pgxTracerAdapter) TraceBatchQuery(_ context.Context, _ *pgx.Conn, _ pgx.TraceBatchQueryData) {}

func (a *pgxTracerAdapter) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	if end, ok := ctx.Value(ctxKeyBatchEnd{}).(func(error)); ok {
		end(data.Err)
	}
}

func (a *pgxTracerAdapter) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	if !a.t.ShouldTrace(ctx) {
		return ctx
	}
	ctx, end := a.t.StartTrace(ctx, "db.CopyFrom", data.TableName.Sanitize(), nil)
	return context.WithValue(ctx, ctxKeyCopyEnd{}, end)
}

func (a *pgxTracerAdapter) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	if end, ok := ctx.Value(ctxKeyCopyEnd{}).(func(error)); ok {
		end(data.Err)
	}
}

func (a *pgxTracerAdapter) TracePrepareStart(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	if !a.t.ShouldTrace(ctx) {
		return ctx
	}
	ctx, end := a.t.StartTrace(ctx, "db.Prepare", data.SQL, nil)
	return context.WithValue(ctx, ctxKeyPrepareEnd{}, end)
}

func (a *pgxTracerAdapter) TracePrepareEnd(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareEndData) {
	if end, ok := ctx.Value(ctxKeyPrepareEnd{}).(func(error)); ok {
		end(data.Err)
	}
}
