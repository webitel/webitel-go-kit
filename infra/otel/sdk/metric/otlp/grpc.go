package otlp

import (
	"context"

	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func GrpcOptions(ctx context.Context, dsn string) ([]sdkmetric.Option, error) {
	exporter, err := otlpgrpc.New(ctx)
	// // options ...
	// otlpgrpc.WithAggregationSelector(selector metric.AggregationSelector),
	// otlpgrpc.WithCompressor(compressor string),
	// otlpgrpc.WithDialOption(opts ...grpc.DialOption),
	// otlpgrpc.WithEndpoint(endpoint string),
	// otlpgrpc.WithEndpointURL(u string),
	// otlpgrpc.WithGRPCConn(conn *grpc.ClientConn),
	// otlpgrpc.WithHeaders(headers map[string]string),
	// otlpgrpc.WithInsecure(),
	// otlpgrpc.WithReconnectionPeriod(rp time.Duration),
	// otlpgrpc.WithRetry(settings RetryConfig),
	// otlpgrpc.WithServiceConfig(serviceConfig string),
	// otlpgrpc.WithTLSCredentials(creds credentials.TransportCredentials),
	// otlpgrpc.WithTemporalitySelector(selector metric.TemporalitySelector),
	// otlpgrpc.WithTimeout(duration time.Duration),

	if err != nil {
		return nil, err
	}
	return []sdkmetric.Option{
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				// sdk.WithInterval(time.Second*60),
				// sdk.WithTimeout(time.Second*30),
			),
		),
	}, nil
}
