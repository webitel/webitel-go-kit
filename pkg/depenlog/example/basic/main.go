package main

import (
	"context"

	gokitlog "github.com/webitel/webitel-go-kit/pkg/depenlog"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/grpclog"
)

func main() {
	l, err := gokitlog.New(gokitlog.Config{
		Level:   "debug",
		JSON:    true,
		Console: true,
	})
	if err != nil {
		panic(err)
	}

	l.Info("service starting", "addr", ":8080")
	l.Debug("verbose detail", "cache_size", 1024)

	db := gokitlog.WithComponent(l, "postgres")
	db.Info("connected", "host", "db-1")

	traceID, _ := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
	spanID, _ := trace.SpanIDFromHex("b7ad6b7169203331")
	ctx := trace.ContextWithSpanContext(context.Background(),
		trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		}),
	)
	l.InfoContext(ctx, "handling request", semconv.UserIDKey, 42)

	l.Error("request failed", "err", context.DeadlineExceeded)

	grpclog.Error("grpc framework log routed through the unified logger")
}
