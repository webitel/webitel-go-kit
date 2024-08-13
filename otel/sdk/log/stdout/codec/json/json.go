package json

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/webitel/webitel-go-kit/otel/sdk/log/stdout/codec"
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

func newValue(v log.Value) value {
	return value{Value: v}
}

type value struct {
	log.Value
}

// MarshalJSON implements a custom marshal function to encode log.Value.
func (v value) MarshalJSON() ([]byte, error) {
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
		input := v.AsMap()
		object := make(map[string]value, len(input))
		for _, att := range input {
			object[att.Key] = newValue(att.Value)
		}

		eval = object
	case log.KindSlice:
		input := v.AsSlice()
		array := make([]value, 0, len(input))
		for _, e := range input {
			array = append(array, newValue(e))
		}

		eval = array
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

type attributes map[string]value

// record is a JSON-serializable representation of a record.
// https://opentelemetry.io/docs/specs/otel/logs/data-model/#log-and-event-record-definition
type record struct {
	Timestamp         *codec.Timestamp      `json:"date,omitempty"`
	ObservedTimestamp *codec.Timestamp      `json:"-"`
	Severity          log.Severity          `json:"-"`
	SeverityText      string                `json:"level,omitempty"`
	Message           value                 `json:"message,omitempty"`
	TraceID           *trace.TraceID        `json:"trace_id,omitempty"`
	TraceFlags        *trace.TraceFlags     `json:"trace_flags,omitempty"`
	SpanID            *trace.SpanID         `json:"span_id,omitempty"`
	Scope             instrumentation.Scope `json:"scope,omitempty"`
	Resource          *resource.Resource    `json:"resource,omitempty"`
	Attributes        attributes            `json:"attrs,omitempty"`
	DroppedAttributes int                   `json:"dropped_attrs,omitempty"`
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

func (enc *Encoder) record(rec sdk.Record) record {
	res := rec.Resource()
	row := record{

		Timestamp:         enc.opts.Timestamp(rec.Timestamp()),
		ObservedTimestamp: enc.opts.Timestamp(rec.ObservedTimestamp()),

		Severity:     rec.Severity(),
		SeverityText: rec.SeverityText(),
		Message:      newValue(rec.Body()),

		TraceID:    traceIdValue(rec.TraceID()),
		TraceFlags: traceFlagsValue(rec.TraceFlags()),
		SpanID:     spanIdValue(rec.SpanID()),

		// Attributes: make([]keyValue, 0, rec.AttributesLen()),
		Attributes: make(attributes, rec.AttributesLen()),

		Resource: &res,
		Scope:    rec.InstrumentationScope(),

		DroppedAttributes: rec.DroppedAttributes(),
	}

	rec.WalkAttributes(func(att log.KeyValue) bool {
		row.Attributes[att.Key] = newValue(att.Value)
		return true
	})

	return row
}

func (enc *Encoder) Encode(rec sdk.Record) error {
	return enc.json.Encode(enc.record(rec))
}

func init() {
	codec.Register("json", NewCodec)
}
