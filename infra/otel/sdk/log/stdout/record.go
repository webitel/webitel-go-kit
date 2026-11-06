// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package stdout // import "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout"

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

// recordOTEL is a JSON-serializable representation of a Record.
// https://opentelemetry.io/docs/specs/otel/logs/data-model/#log-and-event-record-definition
type recordOTEL struct {
	Timestamp         *time.Time `json:",omitempty"`
	ObservedTimestamp *time.Time `json:",omitempty"`
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

func (e *Exporter) newRecordJSON(r sdklog.Record) recordOTEL {
	newRecord := recordOTEL{
		Severity:     r.Severity(),
		SeverityText: r.SeverityText(),
		Body:         r.Body(),

		TraceID:    r.TraceID(),
		SpanID:     r.SpanID(),
		TraceFlags: r.TraceFlags(),

		Attributes: make([]attribute.KeyValue, 0, r.AttributesLen()),

		Resource: r.Resource(),
		Scope:    r.InstrumentationScope(),

		DroppedAttributes: r.DroppedAttributes(),
	}

	r.WalkAttributes(func(kv attribute.KeyValue) bool {
		newRecord.Attributes = append(newRecord.Attributes, kv)
		return true
	})

	if e.timestamps {
		timestamp := r.Timestamp()
		newRecord.Timestamp = &timestamp

		observedTimestamp := r.ObservedTimestamp()
		newRecord.ObservedTimestamp = &observedTimestamp
	}

	return newRecord
}
