package logradapter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-logr/logr"

	"github.com/webitel/webitel-go-kit/pkg/logging"
)

const Component logging.Component = "logr"

const (
	nameKey = "logger"
	badKey  = "!BADKEY"
)

// New returns a [logr.Logger] writing to the instrumentation scope
// [Component]. Names from WithName are joined with "/" into the attribute
// logger. Values implementing [logr.Marshaler] are replaced by MarshalLog.
func New(l *logging.Logger, opts ...Option) logr.Logger {
	cfg := config{
		component: Component,
		verbosity: defaultVerbosity,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return logr.New(&sink{
		log:       l.WithComponent(cfg.component),
		verbosity: cfg.verbosity,
	})
}

type sink struct {
	log       *logging.Logger
	verbosity func(v int) logging.Level
	name      string
}

var _ logr.LogSink = (*sink)(nil)

func (s *sink) Init(logr.RuntimeInfo) {}

func (s *sink) Enabled(level int) bool {
	return s.log.Enabled(context.Background(), s.verbosity(level))
}

func (s *sink) Info(level int, msg string, kvs ...any) {
	s.log.Log(context.Background(), s.verbosity(level), msg, s.fields(kvs, nil)...)
}

func (s *sink) Error(err error, msg string, kvs ...any) {
	s.log.ErrorCtx(context.Background(), msg, s.fields(kvs, err)...)
}

func (s *sink) WithValues(kvs ...any) logr.LogSink {
	res := *s
	res.log = s.log.With(appendKVs(nil, kvs)...)
	return &res
}

func (s *sink) WithName(name string) logr.LogSink {
	res := *s
	if s.name == "" {
		res.name = name
	} else {
		res.name = s.name + "/" + name
	}
	return &res
}

func (s *sink) fields(kvs []any, err error) []logging.Field {
	out := make([]logging.Field, 0, 2+len(kvs)/2)
	if s.name != "" {
		out = append(out, logging.Str(nameKey, s.name))
	}
	out = appendKVs(out, kvs)
	if err != nil {
		out = append(out, logging.Err(err))
	}
	return out
}

func appendKVs(out []logging.Field, kvs []any) []logging.Field {
	for len(kvs) > 0 {
		switch key := kvs[0].(type) {
		case string:
			if len(kvs) == 1 {
				return append(out, logging.Str(badKey, key))
			}
			out = append(out, logging.Any(key, marshal(kvs[1])))
			kvs = kvs[2:]
		case slog.Attr:
			out = append(out, logging.Attr(key))
			kvs = kvs[1:]
		default:
			out = append(out, logging.Any(badKey, marshal(key)))
			kvs = kvs[1:]
		}
	}
	return out
}

func marshal(v any) (res any) {
	m, ok := v.(logr.Marshaler)
	if !ok {
		return v
	}

	defer func() {
		if r := recover(); r != nil {
			res = fmt.Sprintf("<panic: %v>", r)
		}
	}()
	return m.MarshalLog()
}

func defaultVerbosity(v int) logging.Level {
	return logging.LevelInfo - logging.Level(v)
}
