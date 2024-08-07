package logging

import (
	"io"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*Logging)
}

type optionFunc func(*Logging)

func (o optionFunc) apply(c *Logging) {
	o(c)
}

func WithServiceName(service string) Option {
	return optionFunc(func(c *Logging) {
		c.serviceName = service
	})
}

func WithServiceVersion(version string) Option {
	return optionFunc(func(c *Logging) {
		c.serviceVersion = version
	})
}

func WithExporter(exporter string) Option {
	return optionFunc(func(c *Logging) {
		c.enabled = Exporter(exporter)
	})
}

func WithAddress(address string) Option {
	return optionFunc(func(c *Logging) {
		c.address = address
	})
}

// WithAttributes specifies additional attributes to be added to the span.
func WithAttributes(attrs ...attribute.KeyValue) Option {
	return optionFunc(func(c *Logging) {
		c.customAttribs = append(c.customAttribs, attrs...)
	})
}

// WithSTDOutWriter sets the export stream destination.
func WithSTDOutWriter(w io.Writer) Option {
	return optionFunc(func(c *Logging) {
		c.writer = w
	})
}

// WithoutSTDOutTimestamps sets the export stream to not include timestamps.
func WithoutSTDOutTimestamps() Option {
	return optionFunc(func(c *Logging) {
		c.timestamps = false
	})
}

// WithMaxQueueSize sets the maximum queue size used by the Batcher.
// After the size is reached log records are dropped.
func WithMaxQueueSize(size int) Option {
	return optionFunc(func(c *Logging) {
		c.batcherMaxQueueSize = size
	})
}

// WithExportInterval sets the maximum duration between batched exports.
func WithExportInterval(d time.Duration) Option {
	return optionFunc(func(c *Logging) {
		c.batcherExportInterval = d
	})
}

// WithExportTimeout sets the duration after which a batched export is canceled.
func WithExportTimeout(d time.Duration) Option {
	return optionFunc(func(c *Logging) {
		c.batcherExportTimeout = d
	})
}

// WithExportMaxBatchSize sets the maximum batch size of every export.
// A batch will be split into multiple exports to not exceed this size.
func WithExportMaxBatchSize(size int) Option {
	return optionFunc(func(c *Logging) {
		c.batcherMaxExportBatchSize = size
	})
}
