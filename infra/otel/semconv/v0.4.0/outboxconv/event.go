// Code generated from semantic convention specification. DO NOT EDIT.

// Copyright (c) 2026 Webitel
// SPDX-License-Identifier: MIT

package outboxconv

import "go.opentelemetry.io/otel/attribute"

// ErrorTypeAttr is a value of the "error.type" attribute. Describes a class of
// error the operation ended with.
type ErrorTypeAttr string

var (
	// ErrorTypeOther is a fallback error value to be used when the instrumentation
	// doesn't define a custom value.
	ErrorTypeOther ErrorTypeAttr = "_OTHER"
)

// RelayFailedEvent is the "webitel.outbox.relay.failed" event. This event
// represents a failed attempt of the outbox relay to publish an outbox event to
// the broker.
type RelayFailedEvent struct {
	attrs []attribute.KeyValue
}

// RelayFailedOption sets an optional attribute of RelayFailedEvent.
type RelayFailedOption struct {
	kv attribute.KeyValue
}

// NewRelayFailedEvent returns a new RelayFailedEvent.
//
// errorType is the "error.type" attribute. Describes a class of error the
// operation ended with.
//
// Optional attributes are set with RelayFailedWith* options.
func NewRelayFailedEvent(
	errorType ErrorTypeAttr,
	opts ...RelayFailedOption,
) RelayFailedEvent {
	attrs := make([]attribute.KeyValue, 0, 1+len(opts))
	attrs = append(attrs, attribute.String("error.type", string(errorType)))
	for _, o := range opts {
		attrs = append(attrs, o.kv)
	}
	return RelayFailedEvent{attrs: attrs}
}

// EventName returns the semantic convention name of the event.
func (RelayFailedEvent) EventName() string {
	return "webitel.outbox.relay.failed"
}

// Attributes returns the attributes of the event.
func (e RelayFailedEvent) Attributes() []attribute.KeyValue {
	return e.attrs
}

// RelayFailedWithMessagingDestinationName sets the optional
// "messaging.destination.name" attribute. The message destination name.
func RelayFailedWithMessagingDestinationName(val string) RelayFailedOption {
	return RelayFailedOption{attribute.String("messaging.destination.name", val)}
}
