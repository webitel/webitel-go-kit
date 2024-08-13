package otlp

import "github.com/webitel/webitel-go-kit/otel/sdk/trace"

func init() {

	trace.Register("otlphttp", HttpOptions)
	trace.Register("otlpgrpc", GrpcOptions)
	// trace.Register("otlp", HttpOptions)
}
