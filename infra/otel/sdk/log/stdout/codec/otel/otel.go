package otel

import (
	"encoding/json"
	"io"

	"github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout/codec"
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
	}
	enc.json = json.NewEncoder(out)
	if enc.opts.PrettyPrint != "" {
		enc.json.SetIndent(
			"", enc.opts.PrettyPrint,
		)
	}
	return enc
}

// record is a JSON-serializable representation of a record.
// https://opentelemetry.io/docs/specs/otel/logs/data-model/#log-and-event-record-definition
// Keep sync with: https://github.com/open-telemetry/opentelemetry-go/blob/main/exporters/stdout/stdoutlog/record.go#L94
type record struct {
	Timestamp         *codec.Timestamp `json:",omitempty"`
	ObservedTimestamp *codec.Timestamp `json:",omitempty"`
	Severity          log.Severity
	SeverityText      string
	Body              attribute.Value
	Attributes        []attribute.KeyValue
	TraceID           trace.TraceID
	SpanID            trace.SpanID
	TraceFlags        trace.TraceFlags
	Resource          *resource.Resource
	Scope             instrumentation.Scope
	DroppedAttributes int
}

func (enc *Encoder) record(src sdk.Record) record {
	out := record{

		Timestamp:         enc.opts.Timestamp(src.Timestamp()),
		ObservedTimestamp: enc.opts.Timestamp(src.ObservedTimestamp()),

		Severity:     src.Severity(),
		SeverityText: src.SeverityText(),
		Body:         src.Body(),

		TraceID:    src.TraceID(),
		SpanID:     src.SpanID(),
		TraceFlags: src.TraceFlags(),

		Attributes: make([]attribute.KeyValue, 0, src.AttributesLen()),

		Resource: src.Resource(),
		Scope:    src.InstrumentationScope(),

		DroppedAttributes: src.DroppedAttributes(),
	}

	src.WalkAttributes(func(att attribute.KeyValue) bool {
		out.Attributes = append(out.Attributes, att)
		return true
	})

	return out
}

func (enc *Encoder) Encode(rec sdk.Record) error {
	return enc.json.Encode(enc.record(rec))
}

func init() {
	codec.Register("otel", NewCodec)
}
