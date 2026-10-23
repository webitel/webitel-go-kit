// Package discoveryfx wires service discovery into an fx application: it
// creates the Consul provider, registers the service instance on start and
// deregisters it on stop. With a *health.Registry in the graph the TTL
// heartbeat reports its readiness verdict.
//
//	fx.New(
//		healthfx.Module(healthfx.Config{HTTPAddr: cfg.Health.Addr}),
//		discoveryfx.Module(discoveryfx.Config{
//			ConsulAddr: cfg.Consul.Addr,
//			Name:       model.ServiceName,
//			Version:    model.Version,
//			Endpoints:  []string{"grpc://" + cfg.Service.Addr},
//		}),
//		// ... servers and dependencies ...
//		healthfx.Shutdown(), // last, so the drain runs before Deregister
//	)
//
// The module provides discovery.DiscoveryProvider.
package discoveryfx
