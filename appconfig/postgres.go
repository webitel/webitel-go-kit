package appconfig

// Postgres holds PostgreSQL connection configuration.
// Env var: POSTGRES_DSN.
type Postgres struct {
	DSN string `mapstructure:"dsn"`
}
