package otlp

import "github.com/webitel/webitel-go-kit/infra/otel/sdk/log"

func init() {

	log.Register("otlphttp", httpOptions)
	log.Register("otlpgrpc", grpcOptions)
	// log.Register("otlp", httpOptions)
}
