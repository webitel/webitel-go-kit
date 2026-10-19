package json

import (
	"encoding/json"
	"io"

	logv "github.com/webitel/webitel-go-kit/infra/otel/log"
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
	attribute.Value
}

// MarshalJSON implements a custom marshal function to encode attribute.Value.
func (v logValue) MarshalJSON() ([]byte, error) {
	var eval any
	switch v.Type() {
	case attribute.MAP:
		src := v.AsMap()
		dst := make(map[string]logValue, len(src))
		for _, att := range src {
			dst[string(att.Key)] = logValue{att.Value}
		}

		eval = dst
	case attribute.SLICE:
		src := v.AsSlice()
		dst := make([]logValue, 0, len(src))
		for _, e := range src {
			dst = append(dst, logValue{e})
		}

		eval = dst
	default:
		eval = v.AsInterface()
	}

	return json.Marshal(eval)
}

type attributes map[string]logValue

type Resource resource.Resource

func (e Resource) MarshalJSON() ([]byte, error) {
	src := resource.Resource(e)
	attr := src.Attributes()
	n := len(attr)
	if n == 0 {
		return nil, nil
	}
	obj := make(map[string]logValue, n)
	for _, att := range attr {
		obj[string(att.Key)] = logValue{att.Value}
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
			*src.Resource(),
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
		src.WalkAttributes(func(att attribute.KeyValue) bool {
			out.Attributes[string(att.Key)] = logValue{att.Value}
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
