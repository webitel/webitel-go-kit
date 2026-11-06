package consul

import (
	"github.com/hashicorp/consul/api"
	"github.com/webitel/webitel-go-kit/infra/discovery"
)

// WithServiceResolver returns an Option that sets the service resolver function of the Registry.
// The service resolver function is used to resolve the service instances for the given service name.
// The function takes a context.Context and a slice of *api.ServiceEntry as input parameters,
// and returns a slice of *discovery.ServiceInstance. If the service resolver function is nil,
// the default service resolver of the Registry is used.
func WithServiceResolver(fn ServiceResolver) discovery.Option[discovery.DiscoveryProvider] {
	return func(p discovery.DiscoveryProvider) {
		if r, ok := p.(*Registry); ok {
			if r.client != nil {
				r.client.resolver = fn
			}
		}
	}
}

// WithServiceChecks returns an Option that sets the service checks of the Registry.
// The service checks are used to check the health of the registered services.
// The function takes a variable number of *api.AgentServiceCheck as input parameters,
// and sets the service checks of the Registry to the given checks.
// If the service checks are nil, the default service checks of the Registry are used.
func WithServiceChecks(checks ...*api.AgentServiceCheck) discovery.Option[discovery.DiscoveryProvider] {
	return func(p discovery.DiscoveryProvider) {
		if r, ok := p.(*Registry); ok {
			if r.client != nil {
				r.client.serviceChecks = checks
			}
		}
	}
}

// WithReadiness returns an Option that sets the readiness verdict reported by
// the TTL heartbeat. On each tick the heartbeat sends pass when fn reports
// true, and critical with the error text as output otherwise. fn is called on
// the heartbeat goroutine, so it must return quickly; health.Registry.ReadyFunc
// reads a cached snapshot and qualifies. It only takes effect with the heartbeat
// enabled. If fn is nil, the heartbeat always reports pass.
func WithReadiness(fn func() (bool, error)) discovery.Option[discovery.DiscoveryProvider] {
	return func(p discovery.DiscoveryProvider) {
		if r, ok := p.(*Registry); ok {
			if r.client != nil {
				r.client.readiness = fn
			}
		}
	}
}
