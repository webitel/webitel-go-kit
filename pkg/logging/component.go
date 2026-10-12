package logging

import "go.opentelemetry.io/otel/attribute"

// Component is the name of an OpenTelemetry instrumentation scope.
// Declare it only as constants. Each name creates a logger
// in the SDK, so names built from runtime values grow without bound and consume memory.
// Put such values into fields instead. Use "<service>.<subsystem>" for
// application code (e.g., "engine.dialer"), and the bare framework name
// for adapters("grpc"/"fx").
type Component string

const DefaultComponent Component = "github.com/webitel/webitel-go-kit/pkg/logging"

type Event interface {
	EventName() string
	Attributes() []attribute.KeyValue
}
