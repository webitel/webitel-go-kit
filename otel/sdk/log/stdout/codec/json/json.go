package json

import (
	"encoding/json"
	"errors"
	"io"

	logv "github.com/webitel/webitel-go-kit/otel/log"
	"github.com/webitel/webitel-go-kit/otel/sdk/log/stdout/codec"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdk "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

type Encoder struct {
	opts codec.Options
	json *json.Encoder
}

var _ codec.Encoder = (*Encoder)(nil)

func NewCodec(out io.Writer, opts ...codec.Option) codec.Encoder {
	enc := &Encoder{
		opts: codec.NewOptions(opts...),
		json: json.NewEncoder(out),
	}
	if enc.opts.PrettyPrint != "" {
		enc.json.SetIndent(
			"", enc.opts.PrettyPrint,
		)
	}
	return enc
}

// Options of current configuration
func (enc *Encoder) Options() codec.Options {
	return enc.opts
}

type logValue struct {
	log.Value
}

// MarshalJSON implements a custom marshal function to encode log.Value.
func (v logValue) MarshalJSON() ([]byte, error) {
	var eval any
	switch v.Kind() {
	case log.KindString:
		eval = v.AsString()
	case log.KindInt64:
		eval = v.AsInt64()
	case log.KindFloat64:
		eval = v.AsFloat64()
	case log.KindBool:
		eval = v.AsBool()
	case log.KindBytes:
		eval = v.AsBytes()
	case log.KindMap:
		src := v.AsMap()
		dst := make(map[string]logValue, len(src))
		for _, att := range src {
			dst[att.Key] = logValue{att.Value}
		}

		eval = dst
	case log.KindSlice:
		src := v.AsSlice()
		dst := make([]logValue, 0, len(src))
		for _, e := range src {
			dst = append(dst, logValue{e})
		}

		eval = dst
	case log.KindEmpty:
		eval = nil
	default:
		return nil, errors.New("invalid Kind")
	}

	return json.Marshal(eval)
}

// type keyValue struct {
// 	Key   string
// 	Value value
// }

type attributes map[string]logValue

type attValue struct {
	attribute.Value
}

// MarshalJSON implements a custom marshal function to encode log.Value.
func (v attValue) MarshalJSON() ([]byte, error) {
	var eval any = v.Value
	switch v.Type() {
	// INVALID is used for a Value with no value set.
	case attribute.INVALID:
		eval = nil
	// BOOL is a boolean Type Value.
	case attribute.BOOL:
		eval = v.AsBool()
	// INT64 is a 64-bit signed integral Type Value.
	case attribute.INT64:
		eval = v.AsInt64()
	// FLOAT64 is a 64-bit floating point Type Value.
	case attribute.FLOAT64:
		eval = v.AsFloat64()
	// STRING is a string Type Value.
	case attribute.STRING:
		eval = v.AsString()
	// BOOLSLICE is a slice of booleans Type Value.
	case attribute.BOOLSLICE:
		eval = v.AsBoolSlice()
	// INT64SLICE is a slice of 64-bit signed integral numbers Type Value.
	case attribute.INT64SLICE:
		eval = v.AsInt64Slice()
	// FLOAT64SLICE is a slice of 64-bit floating point numbers Type Value.
	case attribute.FLOAT64SLICE:
		eval = v.AsFloat64Slice()
	// STRINGSLICE is a slice of strings Type Value.
	case attribute.STRINGSLICE:
		eval = v.AsStringSlice()
		// default:
		// 	eval = v.Value
	}
	return json.Marshal(eval)
}

type Resource resource.Resource

func (e Resource) MarshalJSON() ([]byte, error) {
	src := resource.Resource(e)
	attr := src.Attributes()
	n := len(attr)
	if n == 0 {
		return nil, nil
	}
	obj := make(map[string]attValue, n)
	for _, att := range attr {
		obj[string(att.Key)] = attValue{att.Value}
	}
	return json.Marshal(obj)
}

// instrumentation.Scope JSON
type instrumentationJSON struct {
	// Name is the name of the instrumentation scope. This should be the
	// Go package name of that scope.
	Name string `json:"name,omitempty"`
	// Version is the version of the instrumentation scope.
	Version string `json:"version,omitempty"`
	// SchemaURL of the telemetry emitted by the scope.
	SchemaURL string `json:"schema_url,omitempty"`
}

type instrumentationScope instrumentation.Scope

func (e instrumentationScope) MarshalJSON() ([]byte, error) {
	if e.Name == "" &&
		e.Version == "" &&
		e.SchemaURL == "" {
		return nil, nil
	}
	return json.Marshal(instrumentationJSON{
		Name:      e.Name,
		Version:   e.Version,
		SchemaURL: e.SchemaURL,
	})
}

// record is a JSON-serializable representation of a record.
// https://opentelemetry.io/docs/specs/otel/logs/data-model/#log-and-event-record-definition
type record struct {
	Timestamp         *codec.Timestamp     `json:"date,omitempty"`
	ObservedTimestamp *codec.Timestamp     `json:"-"`
	Severity          log.Severity         `json:"-"`
	SeverityText      string               `json:"level,omitempty"`
	Message           logValue             `json:"message,omitempty"`
	TraceID           *trace.TraceID       `json:"trace_id,omitempty"`
	TraceFlags        *trace.TraceFlags    `json:"trace_flags,omitempty"`
	SpanID            *trace.SpanID        `json:"span_id,omitempty"`
	Scope             instrumentationScope `json:"scope,omitempty"`
	Resource          Resource             `json:"resource,omitempty"`
	Attributes        attributes           `json:"attributes,omitempty"`
	DroppedAttributes int                  `json:"dropped_attrs,omitempty"`
}

func traceIdValue(v trace.TraceID) *trace.TraceID {
	if !v.IsValid() {
		return nil
	}
	return &v
}

func traceFlagsValue(v trace.TraceFlags) *trace.TraceFlags {
	if v == 0 {
		return nil
	}
	return &v
}

func spanIdValue(v trace.SpanID) *trace.SpanID {
	if !v.IsValid() {
		return nil
	}
	return &v
}

func (enc *Encoder) record(src sdk.Record) record {

	out := record{

		Timestamp:         enc.opts.Timestamp(src.Timestamp()),
		ObservedTimestamp: enc.opts.Timestamp(src.ObservedTimestamp()),

		Severity:     src.Severity(),
		SeverityText: src.SeverityText(),
		Message:      logValue{src.Body()},

		TraceID:    traceIdValue(src.TraceID()),
		TraceFlags: traceFlagsValue(src.TraceFlags()),
		SpanID:     spanIdValue(src.SpanID()),

		// Attributes: make([]keyValue, 0, rec.AttributesLen()),

		Resource: Resource(
			src.Resource(),
		),
		Scope: instrumentationScope(
			src.InstrumentationScope(),
		),

		DroppedAttributes: src.DroppedAttributes(),
	}

	if out.SeverityText == "" {
		out.SeverityText = logv.Severity(out.Severity).String()
	}

	if n := src.AttributesLen(); n > 0 {
		out.Attributes = make(attributes, n)
		src.WalkAttributes(func(att log.KeyValue) bool {
			out.Attributes[att.Key] = logValue{att.Value}
			return true
		})
	}

	return out
}

func (enc *Encoder) Encode(rec sdk.Record) error {
	return enc.json.Encode(enc.record(rec))
}

func init() {
	codec.Register("json", NewCodec)
}
