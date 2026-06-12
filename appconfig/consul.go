package appconfig

// Consul holds Consul service-discovery configuration.
// Env var: CONSUL_ADDR.
type Consul struct {
	Addr string `mapstructure:"addr"`
}
