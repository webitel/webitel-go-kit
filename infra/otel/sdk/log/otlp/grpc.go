package otlp

import (
	"context"

	"github.com/webitel/webitel-go-kit/infra/otel/sdk/log"

	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func grpcOptions(ctx context.Context, dsn string) ([]log.Option, error) {
	exporter, err := otlpgrpc.New(ctx)
	// // options ...
	// otlpgrpc.WithCompressor(compressor string),
	// otlpgrpc.WithDialOption(opts ...grpc.DialOption),
	// otlpgrpc.WithEndpoint(endpoint string),
	// otlpgrpc.WithEndpointURL(rawURL string),
	// otlpgrpc.WithGRPCConn(conn *grpc.ClientConn),
	// otlpgrpc.WithHeaders(headers map[string]string),
	// otlpgrpc.WithInsecure(),
	// otlpgrpc.WithReconnectionPeriod(rp time.Duration),
	// otlpgrpc.WithRetry(rc RetryConfig),
	// otlpgrpc.WithServiceConfig(serviceConfig string),
	// otlpgrpc.WithTLSCredentials(credential credentials.TransportCredentials),
	// otlpgrpc.WithTimeout(duration time.Duration),

	if err != nil {
		return nil, err
	}
	return []sdklog.LoggerProviderOption{
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(
				exporter,
				// // options
				// sdklog.WithMaxQueueSize(2048),
				// sdklog.WithExportMaxBatchSize(512),
				// sdklog.WithExportInterval(time.Second),
				// sdklog.WithExportTimeout(time.Second*30),
			),
		),
	}, nil
}
