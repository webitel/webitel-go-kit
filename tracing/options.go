package tracing

import (
	"go.opentelemetry.io/otel/attribute"
)

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func WithServiceName(service string) Option {
	return optionFunc(func(c *config) {
		c.serviceName = service
	})
}

func WithServiceVersion(version string) Option {
	return optionFunc(func(c *config) {
		c.serviceVersion = version
	})
}

func WithExporter(exporter string) Option {
	return optionFunc(func(c *config) {
		c.enabled = exporter
	})
}

func WithAddress(address string) Option {
	return optionFunc(func(c *config) {
		c.address = address
	})
}

// WithAttributes specifies additional attributes to be added to the span.
func WithAttributes(attrs ...attribute.KeyValue) Option {
	return optionFunc(func(cfg *config) {
		cfg.customAttribs = append(cfg.customAttribs, attrs...)
	})
}

func WithSamplerParam(p float64) Option {
	return optionFunc(func(cfg *config) {
		cfg.samplerParam = p
	})
}
