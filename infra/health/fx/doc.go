// Package healthfx wires the health registry into an fx application: it builds
// the registry with the sd_notify and HTTP probe transports, starts it with the
// application, and drains it first on shutdown.
//
//	fx.New(
//		healthfx.Module(healthfx.Config{HTTPAddr: cfg.Health.Addr}),
//		// ... servers and dependencies ...
//		fx.Invoke(registerChecks),
//		healthfx.Shutdown(), // last, so the drain runs before any server stops
//	)
//
// The module provides *health.Registry, for checks and for the service
// discovery verdict. Extra health options, such as the metrics transport, go
// after the config:
//
//	healthfx.Module(cfg, health.WithTransport(otelhealth.NewTransport()))
package healthfx
