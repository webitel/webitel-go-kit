package tracing

import (
	"time"

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
	return optionFunc(func(c *config) {
		c.customAttribs = append(c.customAttribs, attrs...)
	})
}

func WithSamplerParam(p float64) Option {
	return optionFunc(func(c *config) {
		c.samplerParam = p
	})
}

// WithMaxQueueSize returns a Option that configures the
// maximum queue size allowed for a BatchSpanProcessor.
func WithMaxQueueSize(size int) Option {
	return optionFunc(func(c *config) {
		c.batcherMaxQueueSize = size
	})
}

// WithMaxExportBatchSize returns a Option that configures
// the maximum export batch size allowed for a BatchSpanProcessor.
func WithMaxExportBatchSize(size int) Option {
	return optionFunc(func(c *config) {
		c.batcherMaxExportBatchSize = size
	})
}

// WithBatchTimeout returns a Option that configures the
// maximum delay allowed for a BatchSpanProcessor before it will export any
// held span (whether the queue is full or not).
func WithBatchTimeout(delay time.Duration) Option {
	return optionFunc(func(c *config) {
		c.batcherBatchTimeout = delay
	})
}

// WithExportTimeout returns a Option that configures the
// amount of time a BatchSpanProcessor waits for an exporter to export before
// abandoning the export.
func WithExportTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) {
		c.batcherExportTimeout = timeout
	})
}

// WithBlocking returns a Option that configures a
// BatchSpanProcessor to wait for enqueue operations to succeed instead of
// dropping data when the queue is full.
func WithBlocking() Option {
	return optionFunc(func(c *config) {
		c.batcherWithBlocking = true
	})
}
