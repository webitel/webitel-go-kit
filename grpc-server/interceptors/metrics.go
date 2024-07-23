package interceptor

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-go-kit/metrics"
)

type grpcType string

const (
	Unary        grpcType = "unary"
	ClientStream grpcType = "client_stream"
	ServerStream grpcType = "server_stream"
	BidiStream   grpcType = "bidi_stream"
)

// MetricsUnaryServerInterceptor is a gRPC server-side interceptor that provides Prometheus monitoring for Unary RPCs.
func MetricsUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		service, method := splitFullMethodName(info.FullMethod)
		h, err := handler(ctx, req)

		// get status code from error
		s := fromError(err)
		ex := exemplarFromContext(ctx)
		incrementWithExemplar(metrics.GRPCServer.HandledCounter, ex, string(Unary), service, method, s.Code().String())
		observeWithExemplar(metrics.GRPCServer.HandledHistogram, time.Since(start).Seconds(), ex, string(Unary), service, method)

		return h, err
	}
}

func incrementWithExemplar(c *prometheus.CounterVec, exemplar prometheus.Labels, lvals ...string) {
	c.WithLabelValues(lvals...).(prometheus.ExemplarAdder).AddWithExemplar(1, exemplar)
}

func observeWithExemplar(h *prometheus.HistogramVec, value float64, exemplar prometheus.Labels, lvals ...string) {
	h.WithLabelValues(lvals...).(prometheus.ExemplarObserver).ObserveWithExemplar(value, exemplar)
}

func exemplarFromContext(ctx context.Context) prometheus.Labels {
	if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
		return prometheus.Labels{"traceID": span.TraceID().String()}
	}

	return nil
}

// fromError returns a grpc status. If the error code is neither a valid grpc status nor a context error, codes.Unknown
// will be set.
func fromError(err error) *status.Status {
	s, ok := status.FromError(err)

	// Mirror what the grpc server itself does, i.e. also convert context errors to status
	if !ok {
		s = status.FromContextError(err)
	}

	return s
}
