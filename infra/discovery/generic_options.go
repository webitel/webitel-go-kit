package discovery

import (
	"time"
)

type Option[T any] func(T)

// WithHealthCheck returns an Option that sets the enableHealthCheck flag of the
// Registry to the given value. If set to true, the registry will enable health
// checks for services registered with it.
func WithHealthCheck[T DefaultOptionConfigurable](enable bool) Option[T] {
	return func(r T) {
		r.SetHealthCheck(enable)
	}
}

// WithTimeout returns an Option that sets the timeout for the registry.
// The timeout is used as the timeout for all Consul API calls made by the registry.
// If the timeout is zero, the default timeout of the registry is used.
func WithTimeout[T DefaultOptionConfigurable](timeout time.Duration) Option[T] {
	return func(r T) {
		r.SetTimeout(timeout)
	}
}

// WithDatacenter returns an Option that sets the data center for the registry.
// The data center is used as the data center for all Consul API calls made by the registry.
// If the data center is not set, the default data center of the registry is used.
func WithDatacenter[T DefaultOptionConfigurable](dc string) Option[T] {
	return func(r T) {
		r.SetDatacenter(dc)
	}
}

// WithHeartbeat returns an Option that sets the heartbeat flag of the Registry to the given value.
// If set to true, the registry will start a goroutine to update the TTL of the service registration periodically.
// The TTL is updated every healthCheckInterval seconds. If the UpdateTTL call failed, the goroutine will sleep for a random
// duration between 1 and 5 seconds, and then retry to update the TTL or re-register the service if the last update TTL call failed.
// If the heartbeat flag is set to false, the goroutine is stopped and the TTL is no longer updated.
func WithHeartbeat[T DefaultOptionConfigurable](heartbeat bool) Option[T] {
	return func(r T) {
		r.SetHeartbeatEnabled(heartbeat)
	}
}

// WithHealthCheckInterval returns an Option that sets the health check interval of the Registry.
// The health check interval is used as the interval for all health checks made by the registry.
// If the interval is zero, the default health check interval of the registry is used.
func WithHealthCheckInterval[T DefaultOptionConfigurable](interval int) Option[T] {
	return func(r T) {
		r.SetHealthCheckInterval(interval)
	}
}

// WithDeregisterCriticalServiceAfter returns an Option that sets the deregister critical service after interval of the Registry.
// The deregister critical service after interval is used as the interval after which the registry will deregister critical services.
// If the interval is zero, the default deregister critical service after interval of the registry is used.
// The unit of the interval is seconds.
// If the interval is negative, the deregister critical service after interval is disabled.
func WithDeregisterCriticalServiceAfter[T DefaultOptionConfigurable](interval int) Option[T] {
	return func(r T) {
		r.SetDeregisterCriticalServiceAfter(interval)
	}
}

// WithTags returns an Option that sets the tags of the Registry.
// The tags are used as additional metadata for the registered services.
// The function takes a variable number of string as input parameters,
// and sets the tags of the Registry to the given tags.
// If the tags are nil, the default tags of the Registry are used.
func WithTags[T DefaultOptionConfigurable](tags ...string) Option[T] {
	return func(r T) {
		r.SetTags(tags...)
	}
}
