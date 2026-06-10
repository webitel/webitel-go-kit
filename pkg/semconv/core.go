package semconv

// Core log record field keys: the canonical JSON field names used when
// encoding OpenTelemetry log records to stdout.
const (
	TimestampKey  = "date"
	LevelKey      = "level"
	MessageKey    = "message"
	TraceIDKey    = "trace_id"
	SpanIDKey     = "span_id"
	TraceFlagsKey = "trace_flags"
)
