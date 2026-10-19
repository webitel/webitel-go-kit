package logging

import (
	"context"
	"log/slog"
	"math"
	"os"
	"runtime"
	"slices"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	eventNameKey = "event.name"
	scopeNameKey = "otel.scope.name"
)

// Logger writes structured records to the OpenTelemetry Logs API. It is safe
// for concurrent use, and the With methods return new loggers that share the
// underlying scope.
//
// Records without an [Ev] field pass through [LevelHandler] and the otelslog
// bridge; records with an [Ev] field are emitted directly with an event name.
type Logger struct {
	core   *core
	fields []Field
	groups []group
}

type group struct {
	name   string
	fields []Field
}

type core struct {
	config
	handler   LevelHandler
	otel      log.Logger
	bootstrap bool
}

// New returns a Logger configured by opts.
func New(opts ...Option) *Logger {
	cfg := newConfig(opts)
	if cfg.provider == nil {
		cfg.provider = otel.GetLoggerProvider()
	}
	return &Logger{core: newCore(cfg)}
}

// Bootstrap returns a Logger writing text to stderr without OpenTelemetry,
// to wrote logs before the logger provider is configured.
// [WithProvider] option is ignored. An [Ev] field adds the attribute event.name
// instead of making the record an event, and a component other than
// [DefaultComponent] adds the attribute otel.scope.name.
func Bootstrap(opts ...Option) *Logger {
	cfg := newConfig(opts)
	cfg.provider = nil
	return &Logger{core: newBootstrapCore(cfg)}
}

func (l *Logger) DebugCtx(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelDebug, msg, nil, fields)
}

func (l *Logger) InfoCtx(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelInfo, msg, nil, fields)
}

func (l *Logger) WarnCtx(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelWarn, msg, nil, fields)
}

func (l *Logger) ErrorCtx(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelError, msg, nil, fields)
}

func (l *Logger) Log(ctx context.Context, lvl Level, msg string, fields ...Field) {
	l.log(ctx, lvl, msg, nil, fields)
}

func (l *Logger) Emit(ctx context.Context, e Event, fields ...Field) {
	l.log(ctx, LevelInfo, "", e, fields)
}

func (l *Logger) Enabled(ctx context.Context, lvl Level) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	return l.core.enabled(ctx, lvl, "", false)
}

func (l *Logger) EventEnabled(ctx context.Context, e Event) bool {
	if e == nil {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return l.core.enabled(ctx, LevelInfo, e.EventName(), true)
}

// With returns a Logger that adds fields to every record. Fields are kept
// as they are and rendered for each record, so sensitivity classes follow
// the [redact] policy at the time of the record.
func (l *Logger) With(fields ...Field) *Logger {
	if len(fields) == 0 {
		return l
	}

	res := *l
	if n := len(l.groups); n == 0 {
		res.fields = append(slices.Clip(l.fields), fields...)
	} else {
		res.groups = slices.Clone(l.groups)
		res.groups[n-1].fields = append(slices.Clip(l.groups[n-1].fields), fields...)
	}
	return &res
}

// WithGroup returns a Logger that nests the fields added later, by With or
// by a log call, under name. Event attributes stay at the top level. An
// empty name returns l.
func (l *Logger) WithGroup(name string) *Logger {
	if name == "" {
		return l
	}

	res := *l
	res.groups = append(slices.Clip(l.groups), group{name: name})
	return &res
}

// WithComponent returns a Logger that writes to the instrumentation scope
// c, keeping the fields and groups of l.
func (l *Logger) WithComponent(c Component) *Logger {
	newCfg := l.core.config
	newCfg.component = c

	res := *l
	if l.core.bootstrap {
		res.core = newBootstrapCore(newCfg)
	} else {
		res.core = newCore(newCfg)
	}
	return &res
}

// Slog returns an [slog.Logger] writing to the same handler, for code that
// needs the standard library API. Its arguments are not typed fields: values
// go through otelslog reflection and are not redacted by this package. The
// fields of l are rendered once, when Slog is called.
func (l *Logger) Slog() *slog.Logger {
	var h slog.Handler = l.core.handler

	h = h.WithAttrs(l.core.slogAttrs(l.fields))
	for _, g := range l.groups {
		h = h.WithGroup(g.name).WithAttrs(l.core.slogAttrs(g.fields))
	}
	return slog.New(h)
}

// OTel returns the OpenTelemetry logger of the instrumentation scope of l.
func (l *Logger) OTel() log.Logger {
	return l.core.otel
}

func (l *Logger) log(ctx context.Context, lvl Level, msg string, ev Event, fields []Field) {
	if ctx == nil {
		ctx = context.Background()
	}

	evName, isEvent := l.eventName(ev, fields)
	if !l.core.enabled(ctx, lvl, evName, isEvent) {
		return
	}

	var pc uintptr
	if l.core.source {
		var pcs [1]uintptr
		// skip runtime.Callers, Logger.log and Logger.<Level>Ctx
		runtime.Callers(3, pcs[:])
		pc = pcs[0]
	}

	e := l.build(ev, fields)
	e.eventName = evName
	if isEvent && !l.core.bootstrap {
		l.core.emitEvent(ctx, lvl, msg, pc, &e)
		return
	}
	l.core.emitSlog(ctx, lvl, msg, pc, &e)
}

func (l *Logger) eventName(ev Event, fields []Field) (string, bool) {
	name := ""
	found := false

	if ev != nil {
		name, found = ev.EventName(), true
	}

	scan := func(fs []Field) {
		for _, f := range fs {
			if f.kind == kindEvent {
				name, found = f.val.Any().(Event).EventName(), true
			}
		}
	}

	scan(l.fields)
	for _, g := range l.groups {
		scan(g.fields)
	}
	scan(fields)

	return name, found
}

func (l *Logger) build(ev Event, fields []Field) entry {
	var e entry
	if ev != nil {
		e.renderField(nil, Ev(ev))
	}

	root := e.render(nil, l.fields)
	if len(l.groups) == 0 {
		root = e.render(root, fields)
	} else {
		var inner []attribute.KeyValue
		last := len(l.groups) - 1
		for i := last; i >= 0; i-- {
			kvs := e.render(nil, l.groups[i].fields)
			if i == last {
				kvs = e.render(kvs, fields)
			} else if len(inner) > 0 {
				kvs = append(kvs, attribute.Map(l.groups[i+1].name, inner...))
			}
			inner = kvs
		}
		if len(inner) > 0 {
			root = append(root, attribute.Map(l.groups[0].name, inner...))
		}
	}
	e.attrs = append(root, e.eventAttrs...)
	return e
}

func newCore(cfg config) *core {
	if cfg.provider == nil {
		return newBootstrapCore(cfg)
	}
	name := string(cfg.component)
	h := otelslog.NewHandler(name,
		otelslog.WithLoggerProvider(cfg.provider),
		otelslog.WithVersion(cfg.version),
		otelslog.WithSchemaURL(cfg.schemaURL),
		otelslog.WithAttributes(cfg.attrs...),
		otelslog.WithSource(cfg.source),
	)
	return &core{
		config:  cfg,
		handler: NewLevelHandler(cfg.level, h),
		otel:    cfg.provider.Logger(name, cfg.otelLoggerOptions()...),
	}
}

func newBootstrapCore(cfg config) *core {
	var h slog.Handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource:   cfg.source,
		Level:       slog.Level(math.MinInt),
		ReplaceAttr: levelName,
	})
	if cfg.component != DefaultComponent {
		h = h.WithAttrs([]slog.Attr{slog.String(scopeNameKey, string(cfg.component))})
	}
	return &core{
		config:    cfg,
		handler:   NewLevelHandler(cfg.level, h),
		otel:      noop.NewLoggerProvider().Logger(string(cfg.component)),
		bootstrap: true,
	}
}

func (c *core) slogAttrs(fields []Field) []slog.Attr {
	var e entry
	kvs := e.render(nil, fields)
	kvs = append(kvs, e.eventAttrs...)
	return c.toSlog(kvs, e.err)
}

func (c *core) toSlog(kvs []attribute.KeyValue, err error) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(kvs)+1)
	for _, kv := range kvs {
		if c.bootstrap {
			attrs = append(attrs, KeyValueToAttr(kv))
		} else {
			attrs = append(attrs, otelAttr(kv))
		}
	}
	if err != nil && !c.bootstrap {
		attrs = append(attrs, slog.Any("", err))
	}
	return attrs
}

func (c *core) enabled(ctx context.Context, lvl Level, eventName string, event bool) bool {
	if !event || c.bootstrap {
		return c.handler.Enabled(ctx, slog.Level(lvl))
	}
	if slog.Level(lvl) < c.handler.minLevel() {
		return false
	}
	return c.otel.Enabled(ctx, log.EnabledParameters{Severity: lvl.Severity(), EventName: eventName})
}

func (c *core) emitSlog(ctx context.Context, lvl Level, msg string, pc uintptr, e *entry) {
	r := slog.NewRecord(time.Now(), slog.Level(lvl), msg, pc)
	r.AddAttrs(c.toSlog(e.attrs, e.err)...)
	if c.bootstrap && e.hasEvent {
		r.AddAttrs(slog.String(eventNameKey, e.eventName))
	}

	_ = c.handler.Handle(ctx, r)
}

func (c *core) emitEvent(ctx context.Context, lvl Level, msg string, pc uintptr, e *entry) {
	var r log.Record
	r.SetTimestamp(time.Now())
	r.SetSeverity(lvl.Severity())
	r.SetSeverityText(slog.Level(lvl).String())
	r.SetEventName(e.eventName)
	if msg != "" {
		r.SetBody(attribute.StringValue(msg))
	}

	if pc != 0 {
		f, _ := runtime.CallersFrames([]uintptr{pc}).Next()
		r.AddAttributes(
			semconv.CodeFilePath(f.File),
			semconv.CodeFunctionName(f.Function),
			semconv.CodeLineNumber(f.Line),
		)
	}
	r.AddAttributes(e.attrs...)
	if e.err != nil {
		r.SetErr(e.err)
	}

	c.otel.Emit(ctx, r)
}

func levelName(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.LevelKey {
		if lvl, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(Level(lvl).String())
		}
	}
	return a
}
