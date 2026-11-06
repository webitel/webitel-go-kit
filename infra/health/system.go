package health

// System is what a check probes. Values for shared infrastructure follow the
// upstream OpenTelemetry semantic conventions; any other value is allowed.
type System string

const (
	SystemPostgreSQL System = "postgresql"
	SystemRedis      System = "redis"
	SystemRabbitMQ   System = "rabbitmq"
	SystemGRPC       System = "grpc"
	SystemHTTP       System = "http"
)

// CheckOption configures one registration.
type CheckOption func(*checkOptions)

type checkOptions struct {
	name string
}

// WithName tells apart checks of one system. The name is a metric attribute:
// keep it static, never a host, an address or a tenant.
func WithName(name string) CheckOption {
	return func(o *checkOptions) { o.name = name }
}

// checkKey is unique within a registry: the system, or system/name.
func checkKey(system System, name string) string {
	if name == "" {
		return string(system)
	}

	return string(system) + "/" + name
}
