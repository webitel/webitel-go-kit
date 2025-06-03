package otlp

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdk "go.opentelemetry.io/otel/sdk/trace"
)

func GrpcOptions(ctx context.Context, dsn string) ([]sdk.TracerProviderOption, error) {
	exporter, err := otlptrace.New(
		ctx, otlpgrpc.NewClient(
		// // options ...
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
		// otlpgrpc.WithTimeout(duration time.Duration),
		),
	)
	if err != nil {
		return nil, err
	}
	return []sdk.TracerProviderOption{
		sdk.WithBatcher(exporter),
	}, nil
}
