package otlp

import (
	"context"

	otlphttp "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func HttpOptions(ctx context.Context, dsn string) ([]sdkmetric.Option, error) {
	exporter, err := otlphttp.New(ctx)
	// // options ...
	// otlphttp.WithAggregationSelector(selector metric.AggregationSelector),
	// otlphttp.WithCompression(compression Compression),
	// otlphttp.WithEndpoint(endpoint string),
	// otlphttp.WithEndpointURL(u string),
	// otlphttp.WithHeaders(headers map[string]string),
	// otlphttp.WithInsecure(),
	// otlphttp.WithProxy(pf HTTPTransportProxyFunc),
	// otlphttp.WithRetry(rc RetryConfig),
	// otlphttp.WithTLSClientConfig(tlsCfg *tls.Config),
	// otlphttp.WithTemporalitySelector(selector metric.TemporalitySelector),
	// otlphttp.WithTimeout(duration time.Duration),
	// otlphttp.WithURLPath(urlPath string),

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
