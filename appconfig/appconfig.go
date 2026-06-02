// Package appconfig provides shared configuration building blocks for Webitel IM services:
// canonical struct definitions for common infrastructure (logging, Postgres, Redis, Consul,
// pubsub, TLS, profiler), section-based flag registration, and a Loader that wires
// pflag → viper → struct unmarshalling.
//
// Usage pattern:
//
//	loader := appconfig.NewLoader(appconfig.Sections{Log: true, Postgres: true, Redis: true})
//	loader.RegisterFlags(pflag.CommandLine)
//	pflag.String("service.id", "", "Service ID") // service-specific flags
//	pflag.Parse()
//	var cfg MyConfig
//	if err := loader.Load(pflag.CommandLine, &cfg); err != nil { ... }
package appconfig

// Sections declares which shared config blocks a CLI command needs.
// Only the declared sections have their flags registered, so a migrate
// command that sets {Log: true, Postgres: true} will never require
// --redis.addr or --consul.addr.
type Sections struct {
	Log      bool
	Postgres bool
	Redis    bool
	Consul   bool
	Pubsub   bool
	Profiler bool
}
