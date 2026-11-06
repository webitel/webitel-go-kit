// Package redaction provides a log record processor that redacts attributes
// declared sensitive by name.
package redaction

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/logging"
	"github.com/webitel/webitel-go-kit/pkg/logging/redact"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

const classUnknown redact.Class = math.MaxUint8

// Processor renders the values of sensitive attributes with the rules of
// [logging.RedactValue], including keys inside map values. Values that
// already look rendered by [redact.Rendered] are kept as is, unless the
// policy drops values of their class.
//
// It changes the record in place, so it must be registered before the
// processors that export records.
type Processor struct {
	classes map[attribute.Key]redact.Class
}

var _ sdklog.Processor = (*Processor)(nil)

// NewProcessor returns a Processor for classes, which maps attribute names to
// the sensitivity class "pii" or "secret". Values of an attribute with any
// other class are dropped.
func NewProcessor(classes map[string]string) *Processor {
	p := &Processor{
		classes: make(map[attribute.Key]redact.Class, len(classes)),
	}

	var unknown []string
	for key, name := range classes {
		c := parseClass(name)
		if c == classUnknown {
			unknown = append(unknown, key)
		}
		p.classes[attribute.Key(key)] = c
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		otel.Handle(fmt.Errorf("otel/sdk/log/redaction: unknown sensitivity class of %s, values are dropped", strings.Join(unknown, ", ")))
	}

	return p
}

func parseClass(name string) redact.Class {
	switch name {
	case "pii":
		return redact.ClassPII
	case "secret":
		return redact.ClassSecret
	default:
		return classUnknown
	}
}

// OnEmit redacts the attributes of r.
func (p *Processor) OnEmit(_ context.Context, r *sdklog.Record) error {
	found := false
	r.WalkAttributes(func(kv attribute.KeyValue) bool {
		found = p.needsRedaction(kv)
		return !found
	})
	if !found {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, r.AttributesLen())
	r.WalkAttributes(func(kv attribute.KeyValue) bool {
		if kv, keep := p.redact(kv); keep {
			attrs = append(attrs, kv)
		}
		return true
	})
	r.SetAttributes(attrs...)
	return nil
}

// Enabled returns false: the processor never makes a record enabled.
func (p *Processor) Enabled(context.Context, sdklog.EnabledParameters) bool {
	return false
}

func (p *Processor) Shutdown(context.Context) error {
	return nil
}

func (p *Processor) ForceFlush(context.Context) error {
	return nil
}

func (p *Processor) needsRedaction(kv attribute.KeyValue) bool {
	if _, ok := p.classes[kv.Key]; ok {
		return true
	}
	return p.needsRedactionValue(kv.Value)
}

func (p *Processor) needsRedactionValue(v attribute.Value) bool {
	switch v.Type() {
	case attribute.MAP:
		return slices.ContainsFunc(v.AsMap(), p.needsRedaction)
	case attribute.SLICE:
		return slices.ContainsFunc(v.AsSlice(), p.needsRedactionValue)
	default:
		return false
	}
}

func (p *Processor) redact(kv attribute.KeyValue) (attribute.KeyValue, bool) {
	c, ok := p.classes[kv.Key]
	if !ok {
		// this key does not require redaction
		kv.Value = p.redactNestedValue(kv.Value)
		return kv, true
	}

	v, keep := logging.RedactValue(c, kv.Value)
	return attribute.KeyValue{Key: kv.Key, Value: v}, keep
}

func (p *Processor) redactNestedValue(v attribute.Value) attribute.Value {
	switch v.Type() {
	case attribute.MAP:
		kvs := v.AsMap()
		if !slices.ContainsFunc(kvs, p.needsRedaction) {
			return v
		}
		out := make([]attribute.KeyValue, 0, len(kvs))
		for _, kv := range kvs {
			if kv, keep := p.redact(kv); keep {
				out = append(out, kv)
			}
		}
		return attribute.MapValue(out...)
	case attribute.SLICE:
		vs := v.AsSlice()
		if !slices.ContainsFunc(vs, p.needsRedactionValue) {
			return v
		}
		out := make([]attribute.Value, len(vs))
		for i, e := range vs {
			out[i] = p.redactNestedValue(e)
		}
		return attribute.SliceValue(out...)
	default:
		return v
	}
}
