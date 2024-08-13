package otel

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
	}
	enc.json = json.NewEncoder(out)
	if enc.opts.PrettyPrint != "" {
		enc.json.SetIndent(
			"", enc.opts.PrettyPrint,
		)
	}
	return enc
}

type value struct {
	log.Value
}

func newValue(v log.Value) value {
	return value{Value: v}
}

// MarshalJSON implements a custom marshal function to encode log.Value.
func (v value) MarshalJSON() ([]byte, error) {
	var jsonVal struct {
		Type  string
		Value interface{}
	}
	jsonVal.Type = v.Kind().String()

	switch v.Kind() {
	case log.KindString:
		jsonVal.Value = v.AsString()
	case log.KindInt64:
		jsonVal.Value = v.AsInt64()
	case log.KindFloat64:
		jsonVal.Value = v.AsFloat64()
	case log.KindBool:
		jsonVal.Value = v.AsBool()
	case log.KindBytes:
		jsonVal.Value = v.AsBytes()
	case log.KindMap:
		m := v.AsMap()
		values := make([]keyValue, 0, len(m))
		for _, kv := range m {
			values = append(values, keyValue{
				Key:   kv.Key,
				Value: newValue(kv.Value),
			})
		}

		jsonVal.Value = values
	case log.KindSlice:
		s := v.AsSlice()
		values := make([]value, 0, len(s))
		for _, e := range s {
			values = append(values, newValue(e))
		}

		jsonVal.Value = values
	case log.KindEmpty:
		jsonVal.Value = nil
	default:
		return nil, errors.New("invalid Kind")
	}

	return json.Marshal(jsonVal)
}

type keyValue struct {
	Key   string
	Value value
}

// record is a JSON-serializable representation of a record.
// https://opentelemetry.io/docs/specs/otel/logs/data-model/#log-and-event-record-definition
// Keep sync with: https://github.com/open-telemetry/opentelemetry-go/blob/main/exporters/stdout/stdoutlog/record.go#L94
type record struct {
	Timestamp         *codec.Timestamp `json:",omitempty"`
	ObservedTimestamp *codec.Timestamp `json:",omitempty"`
	Severity          log.Severity
	SeverityText      string
	Body              value
	Attributes        []keyValue
	TraceID           trace.TraceID
	SpanID            trace.SpanID
	TraceFlags        trace.TraceFlags
	Resource          *resource.Resource
	Scope             instrumentation.Scope
	DroppedAttributes int
}

func (enc *Encoder) record(rec sdk.Record) record {
	res := rec.Resource()
	row := record{

		Timestamp:         enc.opts.Timestamp(rec.Timestamp()),
		ObservedTimestamp: enc.opts.Timestamp(rec.ObservedTimestamp()),

		Severity:     rec.Severity(),
		SeverityText: rec.SeverityText(),
		Body:         newValue(rec.Body()),

		TraceID:    rec.TraceID(),
		SpanID:     rec.SpanID(),
		TraceFlags: rec.TraceFlags(),

		Attributes: make([]keyValue, 0, rec.AttributesLen()),

		Resource: &res,
		Scope:    rec.InstrumentationScope(),

		DroppedAttributes: rec.DroppedAttributes(),
	}

	rec.WalkAttributes(func(kv log.KeyValue) bool {
		row.Attributes = append(row.Attributes, keyValue{
			Key:   kv.Key,
			Value: newValue(kv.Value),
		})
		return true
	})

	return row
}

func (enc *Encoder) Encode(rec sdk.Record) error {
	return enc.json.Encode(enc.record(rec))
}

func init() {
	codec.Register("otel", NewCodec)
}
