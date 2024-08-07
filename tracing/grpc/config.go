package grpc

import (
	"context"
	"net"
	"strconv"
	"sync/atomic"

	"go.opentelemetry.io/otel/codes"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/webitel/webitel-go-kit/internal"
	"github.com/webitel/webitel-go-kit/semconv"
)

const (
	// scopeName is the instrumentation scope name.
	scopeName = "github.com/webitel/webitel-go-kit/tracing/grpc"
)

// Filter is a predicate used to determine whether a given request in
// should be instrumented by the attached RPC tag info.
// A Filter must return true if the request should be instrumented.
type Filter func(*stats.RPCTagInfo) bool

// config is a group of options for this instrumentation.
type config struct {
	Filter           Filter
	Propagators      propagation.TextMapPropagator
	TracerProvider   trace.TracerProvider
	SpanStartOptions []trace.SpanStartOption

	ReceivedEvent bool
	SentEvent     bool

	tracer trace.Tracer
}

// Option applies an option value for a config.
type Option interface {
	apply(*config)
}

// newConfig returns a config configured with all the passed Options.
func newConfig(opts []Option, role string) *config {
	c := &config{
		Propagators:    otel.GetTextMapPropagator(),
		TracerProvider: otel.GetTracerProvider(),
	}

	for _, o := range opts {
		o.apply(c)
	}

	c.tracer = c.TracerProvider.Tracer(scopeName,
		trace.WithInstrumentationVersion(internal.Version()),
	)

	return c
}

type propagatorsOption struct{ p propagation.TextMapPropagator }

func (o propagatorsOption) apply(c *config) {
	if o.p != nil {
		c.Propagators = o.p
	}
}

// WithPropagators returns an Option to use the Propagators when extracting
// and injecting trace context from requests.
func WithPropagators(p propagation.TextMapPropagator) Option {
	return propagatorsOption{p: p}
}

type tracerProviderOption struct{ tp trace.TracerProvider }

func (o tracerProviderOption) apply(c *config) {
	if o.tp != nil {
		c.TracerProvider = o.tp
	}
}

// WithFilter returns an Option to use the request filter.
func WithFilter(f Filter) Option {
	return filterOption{f: f}
}

type filterOption struct {
	f Filter
}

func (o filterOption) apply(c *config) {
	if o.f != nil {
		c.Filter = o.f
	}
}

// WithTracerProvider returns an Option to use the TracerProvider when
// creating a Tracer.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return tracerProviderOption{tp: tp}
}

// Event type that can be recorded, see WithMessageEvents.
type Event int

// Different types of events that can be recorded, see WithMessageEvents.
const (
	ReceivedEvents Event = iota
	SentEvents
)

type messageEventsProviderOption struct {
	events []Event
}

func (m messageEventsProviderOption) apply(c *config) {
	for _, e := range m.events {
		switch e {
		case ReceivedEvents:
			c.ReceivedEvent = true
		case SentEvents:
			c.SentEvent = true
		}
	}
}

// WithMessageEvents configures the Handler to record the specified events
// (span.AddEvent) on spans. By default only summary attributes are added at the
// end of the request.
//
// Valid events are:
//   - ReceivedEvents: Record the number of bytes read after every gRPC read operation.
//   - SentEvents: Record the number of bytes written after every gRPC write operation.
func WithMessageEvents(events ...Event) Option {
	return messageEventsProviderOption{events: events}
}

type spanStartOption struct{ opts []trace.SpanStartOption }

func (o spanStartOption) apply(c *config) {
	c.SpanStartOptions = append(c.SpanStartOptions, o.opts...)
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanStartOption) Option {
	return spanStartOption{opts}
}

func (c *config) handleRPC(ctx context.Context, rs stats.RPCStats, isServer bool) { // nolint: revive  // isServer is not a control flag.
	span := trace.SpanFromContext(ctx)
	var metricAttrs []attribute.KeyValue
	var messageId int64

	gctx, _ := ctx.Value(gRPCContextKey{}).(*gRPCContext)
	if gctx != nil {
		if !gctx.record {
			return
		}
		metricAttrs = make([]attribute.KeyValue, 0, len(gctx.metricAttrs)+1)
		metricAttrs = append(metricAttrs, gctx.metricAttrs...)
	}

	switch rs := rs.(type) {
	case *stats.Begin:
	case *stats.InPayload:
		if gctx != nil {
			messageId = atomic.AddInt64(&gctx.messagesReceived, 1)
		}

		if c.ReceivedEvent {
			span.AddEvent("message",
				trace.WithAttributes(
					semconv.RPCMessageTypeReceived,
					semconv.RPCMessageIDKey.Int64(messageId),
					semconv.RPCMessageCompressedSizeKey.Int(rs.CompressedLength),
					semconv.RPCMessageUncompressedSizeKey.Int(rs.Length),
				),
			)
		}
	case *stats.OutPayload:
		if gctx != nil {
			messageId = atomic.AddInt64(&gctx.messagesSent, 1)
		}

		if c.SentEvent {
			span.AddEvent("message",
				trace.WithAttributes(
					semconv.RPCMessageTypeSent,
					semconv.RPCMessageIDKey.Int64(messageId),
					semconv.RPCMessageCompressedSizeKey.Int(rs.CompressedLength),
					semconv.RPCMessageUncompressedSizeKey.Int(rs.Length),
				),
			)
		}
	case *stats.OutTrailer:
	case *stats.OutHeader:
		if p, ok := peer.FromContext(ctx); ok {
			span.SetAttributes(peerAttr(p.Addr.String())...)
		}
	case *stats.End:
		var rpcStatusAttr attribute.KeyValue

		if rs.Error != nil {
			s, _ := status.FromError(rs.Error)
			if isServer {
				statusCode, msg := serverStatus(s)
				span.SetStatus(statusCode, msg)
			} else {
				span.SetStatus(codes.Error, s.Message())
			}

			rpcStatusAttr = semconv.RPCGRPCStatusCodeKey.Int(int(s.Code()))
		} else {
			rpcStatusAttr = semconv.RPCGRPCStatusCodeKey.Int(int(grpccodes.OK))
		}

		span.SetAttributes(rpcStatusAttr)
		span.End()
	default:
		return
	}
}

// peerAttr returns attributes about the peer address.
func peerAttr(addr string) []attribute.KeyValue {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return nil
	}

	if host == "" {
		host = "127.0.0.1"
	}

	port, err := strconv.Atoi(p)
	if err != nil {
		return nil
	}

	attr := []attribute.KeyValue{
		semconv.NetworkPeerAddress(host),
		semconv.NetworkPeerPort(port),
	}

	return attr
}

// peerFromCtx returns a peer address from a context, if one exists.
func peerFromCtx(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	return p.Addr.String()
}

// serverStatus returns a span status code and message for a given gRPC
// status code. It maps specific gRPC status codes to a corresponding span
// status code and message. This function is intended for use on the server
// side of a gRPC connection.
//
// If the gRPC status code is Unknown, DeadlineExceeded, Unimplemented,
// Internal, Unavailable, or DataLoss, it returns a span status code of Error
// and the message from the gRPC status. Otherwise, it returns a span status
// code of Unset and an empty message.
func serverStatus(grpcStatus *status.Status) (codes.Code, string) {
	switch grpcStatus.Code() {
	case grpccodes.Unknown,
		grpccodes.DeadlineExceeded,
		grpccodes.Unimplemented,
		grpccodes.Internal,
		grpccodes.Unavailable,
		grpccodes.DataLoss:
		return codes.Error, grpcStatus.Message()
	default:
		return codes.Unset, ""
	}
}
