package otlp

import "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric"

func init() {

	metric.Register("otlphttp", HttpOptions)
	metric.Register("otlpgrpc", GrpcOptions)
	// metric.Register("otlp", HttpOptions)
}
