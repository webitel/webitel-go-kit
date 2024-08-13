package otlp

import "github.com/webitel/webitel-go-kit/otel/sdk/log"

func init() {

	log.Register("otlphttp", HttpOptions)
	log.Register("otlpgrpc", GrpcOptions)
	// log.Register("otlp", HttpOptions)
}
