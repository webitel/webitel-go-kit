// Package semconv defines Webitel's semantic-convention keys: the canonical
// attribute and field names shared across services for structured logging and
// telemetry, keeping field naming consistent between producers and consumers.
package semconv

// Application-level log attribute keys for correlating log records by request,
// identity, and originating component.
const (
	RequestIDKey = "request_id"
	UserIDKey    = "user_id"
	DomainIDKey  = "domain_id"
	ComponentKey = "component"
	ErrorKey     = "error"
)
