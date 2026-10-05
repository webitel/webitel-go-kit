package semconv

// OpenTelemetry resource attribute keys describing the service that produces
// telemetry, mirroring the OTel service.* resource conventions.
const (
	ServiceNameKey       = "service.name"
	ServiceVersionKey    = "service.version"
	ServiceInstanceIDKey = "service.instance.id"
	ServiceNamespaceKey  = "service.namespace"
)
