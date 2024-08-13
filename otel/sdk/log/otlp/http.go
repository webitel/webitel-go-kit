package otlp

import (
	"context"

	"github.com/webitel/webitel-go-kit/otel/sdk/log"

	otlphttp "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func HttpOptions(ctx context.Context, dsn string) ([]log.Option, error) {
	exporter, err := otlphttp.New(ctx)
	// // options ...
	// otlphttp.WithCompression(compression Compression),
	// otlphttp.WithEndpoint(endpoint string),
	// otlphttp.WithEndpointURL(rawURL string),
	// otlphttp.WithHeaders(headers map[string]string),
	// otlphttp.WithInsecure(),
	// otlphttp.WithProxy(pf HTTPTransportProxyFunc),
	// otlphttp.WithRetry(rc RetryConfig),
	// otlphttp.WithTLSClientConfig(tlsCfg *tls.Config),
	// otlphttp.WithTimeout(duration time.Duration),
	// otlphttp.WithURLPath(urlPath string),

	if err != nil {
		return nil, err
	}
	return []sdklog.LoggerProviderOption{
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(
				exporter,
				// // options ...
				// sdklog.WithExportInterval(time.Second),
				// sdklog.WithExportMaxBatchSize(512),
				// sdklog.WithExportTimeout(time.Second*30),
				// sdklog.WithMaxQueueSize(2048),
			),
		),
	}, nil
}
