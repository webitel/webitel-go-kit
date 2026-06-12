package appconfig

// Redis holds Redis connection configuration.
// Env vars: REDIS_ADDR, REDIS_PASSWORD, REDIS_DB.
type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}
