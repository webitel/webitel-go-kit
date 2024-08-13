package bridge

import (
	"fmt"
	"slices"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type Options struct {
	Provider  log.LoggerProvider
	SchemaURL string
	Version   string
	// Common
	TimeStamp   string // timestamp format
	PrittyPrint string // indent string
}

func (c Options) logger(name string) log.Logger {
	var opts []log.LoggerOption
	if c.Version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.Version))
	}
	if c.SchemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.SchemaURL))
	}
	return c.Provider.Logger(name, opts...)
}

type Option func(conf *Options)

func newOptions(opts ...Option) (conf Options) {
	conf = Options{
		Provider: global.GetLoggerProvider(),
	}
	for _, opt := range opts {
		opt(&conf)
	}
	return
}

func WithVersion(ver string) Option {
	return func(conf *Options) {
		conf.Version = ver
	}
}

func WithSchemaURL(url string) Option {
	return func(conf *Options) {
		conf.SchemaURL = url
	}
}

func WithProvider(provider log.LoggerProvider) Option {
	return func(conf *Options) {
		conf.Provider = provider
	}
}

type Handler struct {
	Options
	log.Logger
	*Attributes
}

func NewHandler(name string, opts ...Option) *Handler {
	conf := newOptions(opts...)
	return &Handler{
		Options:    conf,
		Logger:     conf.logger(name),
		Attributes: nil,
	}
}

func (h *Handler) WithName(name string) *Handler {
	// h2 := *h
	return &Handler{
		Options:    h.Options,
		Attributes: h.Attributes.Clone(),
		Logger:     h.Options.logger(name),
	}
}

func (h *Handler) WithAttrs(data ...log.KeyValue) *Handler {
	// h2 := *h
	h2 := &Handler{
		Options:    h.Options,
		Attributes: h.Attributes.Clone(),
		Logger:     h.Logger,
	}
	if n := len(data); n > 0 {
		if h2.Attributes == nil {
			h2.Attributes = &Attributes{}
		}
		h2.Attributes.Add(data)
	}
	return h2
}

// func Enabled(ctx context.Context, logger log.Logger, level log.Severity) bool {
// 	var test log.Record
// 	test.SetSeverity(log.Severity(level))
// 	return logger.Enabled(ctx, test)
// }

type Attributes struct {
	data []log.KeyValue
}

func (e *Attributes) Len() int {
	if e != nil {
		return len(e.data)
	}
	return 0
}

func (e *Attributes) Clone() *Attributes {
	if e == nil {
		return nil
	}
	return &Attributes{data: slices.Clone(e.data)}
}

func (e *Attributes) List() []log.KeyValue {
	if e != nil {
		return e.data
	}
	return nil
}

func (e *Attributes) Add(data []log.KeyValue) {
	e.data = slices.Concat(e.data, data)
}

// Value convertion
func Value(v any, conv ...func(v any) log.Value) log.Value {
	// func BoolValue(v bool) Value
	// func BytesValue(v []byte) Value
	// func Float64Value(v float64) Value
	// func Int64Value(v int64) Value
	// func IntValue(v int) Value
	// func MapValue(kvs ...KeyValue) Value
	// func SliceValue(vs ...Value) Value
	// func StringValue(v string) Value
	switch v := v.(type) {
	case bool:
		return log.BoolValue(v)
	case []byte:
		return log.BytesValue(v)
	case float64:
		return log.Float64Value(v)
	case int64:
		return log.Int64Value(v)
	case uint64:
		return log.Int64Value(int64(v))
	case int:
		return log.IntValue(v)
	case string:
		return log.StringValue(v)
	case map[string]any:
		group := make([]log.KeyValue, len(v))
		for k, v := range v {
			group = append(group, log.KeyValue{
				Key:   k,
				Value: Value(v, conv...),
			})
		}
		return log.MapValue(group...)
	}
	return log.StringValue(fmt.Sprintf("%+v", v))
}
