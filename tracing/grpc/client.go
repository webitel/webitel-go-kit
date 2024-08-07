package grpc

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/stats"

	"github.com/webitel/webitel-go-kit/internal"
	"github.com/webitel/webitel-go-kit/semconv"
	"github.com/webitel/webitel-go-kit/tracing"
)

type gRPCContextKey struct{}

type gRPCContext struct {
	messagesReceived int64
	messagesSent     int64
	metricAttrs      []attribute.KeyValue
	record           bool
}

type clientHandler struct {
	*config
}

// NewClientHandler creates a stats.Handler for a gRPC client.
func NewClientHandler(opts ...Option) stats.Handler {
	h := &clientHandler{
		config: newConfig(opts, "client"),
	}

	return h
}

// TagRPC can attach some information to the given context.
func (h *clientHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	name, attrs := internal.ParseFullMethod(info.FullMethodName)
	attrs = append(attrs, semconv.RPCSystemGRPC)
	ctx, _ = h.tracer.Start(
		ctx,
		name,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)

	gctx := gRPCContext{
		metricAttrs: attrs,
		record:      true,
	}
	if h.config.Filter != nil {
		gctx.record = h.config.Filter(info)
	}

	return tracing.Inject(context.WithValue(ctx, gRPCContextKey{}, &gctx), h.config.Propagators)
}

// HandleRPC processes the RPC stats.
func (h *clientHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	h.handleRPC(ctx, rs, false)
}

// TagConn can attach some information to the given context.
func (h *clientHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn processes the Conn stats.
func (h *clientHandler) HandleConn(context.Context, stats.ConnStats) {
	// no-op
}
