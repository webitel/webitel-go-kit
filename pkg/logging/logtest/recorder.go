// Package logtest records OpenTelemetry log records in memory for tests.
//
// [Recorder] implements [log.LoggerProvider] from the API module only, so
// tests can build a real logger over it without the SDK:
//
//	rec := logtest.NewRecorder()
//	l := logging.New(logging.WithProvider(rec))
//	l.InfoCtx(ctx, "hello")
//	got := rec.Records()
package logtest

import (
	"context"
	"slices"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

// Scope is the instrumentation scope a record was emitted to.
type Scope struct {
	Name       string
	Version    string
	SchemaURL  string
	Attributes attribute.Set
}

// Record is one emitted record with its scope and context.
type Record struct {
	log.Record
	Scope   Scope
	Context context.Context
}

// Attributes returns the attributes of r in emit order.
func (r Record) Attributes() []attribute.KeyValue {
	kvs := make([]attribute.KeyValue, 0, r.AttributesLen())
	r.WalkAttributes(func(kv attribute.KeyValue) bool {
		kvs = append(kvs, kv)
		return true
	})
	return kvs
}

// EnabledFunc decides whether a logger of scope accepts a record.
type EnabledFunc func(ctx context.Context, scope Scope, param log.EnabledParameters) bool

// Option configures a [Recorder].
type Option func(*Recorder)

// WithEnabled sets the function answering Enabled. By default every record
// is enabled.
func WithEnabled(f EnabledFunc) Option {
	return func(r *Recorder) {
		r.enabled = f
	}
}

// Recorder is a [log.LoggerProvider] keeping every emitted record. It is
// safe for concurrent use.
type Recorder struct {
	embedded.LoggerProvider

	enabled EnabledFunc

	mu      sync.Mutex
	records []Record
}

var _ log.LoggerProvider = (*Recorder)(nil)

// NewRecorder returns an empty Recorder.
func NewRecorder(opts ...Option) *Recorder {
	r := &Recorder{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Logger implements [log.LoggerProvider].
func (r *Recorder) Logger(name string, opts ...log.LoggerOption) log.Logger {
	cfg := log.NewLoggerConfig(opts...)
	return &logger{
		rec: r,
		scope: Scope{
			Name:       name,
			Version:    cfg.InstrumentationVersion(),
			SchemaURL:  cfg.SchemaURL(),
			Attributes: cfg.InstrumentationAttributes(),
		},
	}
}

// Records returns a copy of the recorded records in emit order.
func (r *Recorder) Records() []Record {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.records)
}

// Reset removes all recorded records.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = nil
}

type logger struct {
	embedded.Logger

	rec   *Recorder
	scope Scope
}

func (l *logger) Emit(ctx context.Context, record log.Record) {
	l.rec.mu.Lock()
	defer l.rec.mu.Unlock()
	l.rec.records = append(l.rec.records, Record{Record: record.Clone(), Scope: l.scope, Context: ctx})
}

func (l *logger) Enabled(ctx context.Context, param log.EnabledParameters) bool {
	if l.rec.enabled == nil {
		return true
	}
	return l.rec.enabled(ctx, l.scope, param)
}
