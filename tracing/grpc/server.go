package grpc

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/stats"

	"github.com/webitel/webitel-go-kit/internal"
	"github.com/webitel/webitel-go-kit/semconv"
	"github.com/webitel/webitel-go-kit/tracing"
)

type serverHandler struct {
	*config
}

// NewServerHandler creates a stats.Handler for a gRPC server.
func NewServerHandler(opts ...Option) stats.Handler {
	h := &serverHandler{
		config: newConfig(opts, "server"),
	}

	return h
}

// TagConn can attach some information to the given context.
func (h *serverHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn processes the Conn stats.
func (h *serverHandler) HandleConn(ctx context.Context, info stats.ConnStats) {
}

// TagRPC can attach some information to the given context.
func (h *serverHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	ctx = tracing.Extract(ctx, h.config.Propagators)

	name, attrs := internal.ParseFullMethod(info.FullMethodName)
	attrs = append(attrs, semconv.RPCSystemGRPC)
	ctx, _ = h.tracer.Start(
		trace.ContextWithRemoteSpanContext(ctx, trace.SpanContextFromContext(ctx)),
		name,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attrs...),
	)

	gctx := gRPCContext{
		metricAttrs: attrs,
		record:      true,
	}
	if h.config.Filter != nil {
		gctx.record = h.config.Filter(info)
	}
	return context.WithValue(ctx, gRPCContextKey{}, &gctx)
}

// HandleRPC processes the RPC stats.
func (h *serverHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	h.handleRPC(ctx, rs, true)
}
