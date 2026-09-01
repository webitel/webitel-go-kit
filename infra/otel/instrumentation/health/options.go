package health

import (
	"go.opentelemetry.io/otel/metric"
)

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*bindingConfig)
}

type optionFunc func(*bindingConfig)

func (o optionFunc) apply(c *bindingConfig) {
	o(c)
}

type bindingConfig struct {
	mp metric.MeterProvider
}

// WithMeterProvider specifies a meter provider to use for creating the
// instruments. If none is specified, the global meter provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(cfg *bindingConfig) {
		if provider != nil {
			cfg.mp = provider
		}
	})
}
