package logging

import (
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
)

type Option func(*config)

type config struct {
	provider  log.LoggerProvider
	component Component
	level     slog.Leveler
	version   string
	schemaURL string
	source    bool
	attrs     []attribute.KeyValue
}

func newConfig(opts []Option) config {
	cfg := config{
		component: DefaultComponent,
		level:     LevelInfo,
	}

	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

func (cfg config) otelLoggerOptions() []log.LoggerOption {
	var opts []log.LoggerOption
	if cfg.version != "" {
		opts = append(opts, log.WithInstrumentationVersion(cfg.version))
	}
	if cfg.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(cfg.schemaURL))
	}
	if len(cfg.attrs) > 0 {
		opts = append(opts, log.WithInstrumentationAttributes(cfg.attrs...))
	}
	return opts
}

// WithProvider sets the logger provider instead of the default otel.GetLoggerProvider.
// Logger created before the global provider is installed starts emitting once it appears.
func WithProvider(p log.LoggerProvider) Option {
	return func(c *config) {
		c.provider = p
	}
}

// WithComponent sets the instrumentation scope name.
func WithComponent(name Component) Option {
	return func(c *config) {
		c.component = name
	}
}

// WithLevel sets the minimum level of records.
func WithLevel(l slog.Leveler) Option {
	return func(c *config) {
		c.level = l
	}
}

// WithVersion sets the instrumentation scope version.
func WithVersion(v string) Option {
	return func(c *config) {
		c.version = v
	}
}

// WithSchemaURL sets the instrumentation scope schema URL
// of the semantic conventions package in use.
func WithSchemaURL(u string) Option {
	return func(c *config) {
		c.schemaURL = u
	}
}

// WithSource adds the code-related attributes of the call site to each record.
func WithSource(on bool) Option {
	return func(c *config) {
		c.source = on
	}
}

// WithAttributes sets instrumentation scope attributes.
func WithAttributes(kv ...attribute.KeyValue) Option {
	return func(c *config) {
		c.attrs = kv
	}
}
