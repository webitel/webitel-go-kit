// Package throttle limits repeated log records before they reach a
// LoggerProvider.
//
// Records with the same key share a quota: the first [Config.First] records of
// a window pass, then every [Config.Every]-th. The key is the instrumentation
// scope name, severity, event name, body, and the tenant from the
// [semconv.WebitelDomainIDKey] baggage member, so one tenant cannot suppress the
// records of another. The body is part of the key, so messages must be
// constant strings with the data in attributes.
//
// A passed record that follows suppressed records carries their number in the
// [SuppressedCountKey] attribute. Records suppressed after the last passed record
// of a window are not reported.
package throttle

import (
	"context"
	"hash/maphash"
	"time"

	semconv "github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.3.0"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

// SuppressedCountKey is the attribute with the number of records suppressed
// since the previous passed record of the same key.
const SuppressedCountKey attribute.Key = "webitel.log.suppressed"

// LoggerProvider throttles the records of the loggers it returns and passes
// the rest to the wrapped provider. Enabled is not throttled.
type LoggerProvider struct {
	embedded.LoggerProvider

	provider log.LoggerProvider
	limiter  *limiter
}

var _ log.LoggerProvider = (*LoggerProvider)(nil)

// NewLoggerProvider returns a LoggerProvider that throttles the records of
// provider with cfg.
func NewLoggerProvider(provider log.LoggerProvider, cfg Config) *LoggerProvider {
	return &LoggerProvider{
		provider: provider,
		limiter:  newLimiter(cfg.withDefaults(), time.Now),
	}
}

func (p *LoggerProvider) Logger(name string, opts ...log.LoggerOption) log.Logger {
	return &throttleLogger{
		logger:  p.provider.Logger(name, opts...),
		scope:   name,
		limiter: p.limiter,
	}
}

type throttleLogger struct {
	embedded.Logger

	logger  log.Logger
	scope   string
	limiter *limiter
}

func (l *throttleLogger) Emit(ctx context.Context, r log.Record) {
	suppressed, pass := l.limiter.allow(l.key(ctx, &r))
	if !pass {
		return
	}

	if suppressed > 0 {
		r = r.Clone()
		r.AddAttributes(SuppressedCountKey.Int64(suppressed))
	}
	l.logger.Emit(ctx, r)
}

func (l *throttleLogger) Enabled(ctx context.Context, param log.EnabledParameters) bool {
	return l.logger.Enabled(ctx, param)
}

func (l *throttleLogger) key(ctx context.Context, r *log.Record) uint64 {
	var h maphash.Hash
	h.SetSeed(l.limiter.seed)
	write := func(s string) {
		_, _ = h.WriteString(s)
		_ = h.WriteByte(0)
	}

	write(l.scope)
	write(r.Severity().String())
	write(r.EventName())
	write(baggage.FromContext(ctx).Member(string(semconv.WebitelDomainIDKey)).Value())
	write(r.Body().String())

	return h.Sum64()
}
