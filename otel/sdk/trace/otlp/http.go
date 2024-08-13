package otlp

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otlphttp "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdk "go.opentelemetry.io/otel/sdk/trace"
)

func HttpOptions(ctx context.Context, dsn string) ([]sdk.TracerProviderOption, error) {
	exporter, err := otlptrace.New(
		ctx, otlphttp.NewClient(
		// // options ...
		// otlphttp.WithCompression(compression Compression),
		// otlphttp.WithEndpoint(endpoint string),
		// otlphttp.WithEndpointURL(u string),
		// otlphttp.WithHeaders(headers map[string]string),
		// otlphttp.WithInsecure(),
		// otlphttp.WithProxy(pf HTTPTransportProxyFunc),
		// otlphttp.WithRetry(rc RetryConfig),
		// otlphttp.WithTLSClientConfig(tlsCfg *tls.Config),
		// otlphttp.WithTimeout(duration time.Duration),
		// otlphttp.WithURLPath(urlPath string),
		),
	)
	if err != nil {
		return nil, err
	}
	return []sdk.TracerProviderOption{
		sdk.WithBatcher(exporter),
	}, nil
}
