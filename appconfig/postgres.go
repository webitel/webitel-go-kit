package appconfig

import (
	"database/sql"
	"time"
)

// Postgres holds PostgreSQL connection and connection-pool configuration.
//
// Env vars: POSTGRES_DSN, POSTGRES_MAX_OPEN_CONNS, POSTGRES_MAX_IDLE_CONNS,
// POSTGRES_CONN_MAX_IDLE_TIME, POSTGRES_CONN_MAX_LIFETIME.
type Postgres struct {
	DSN string `mapstructure:"dsn"`

	// Connection pool — zero values mean "use driver default".
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// ApplyToSQLDB applies non-zero pool settings to a *sql.DB.
// Call this immediately after sql.Open.
func (p Postgres) ApplyToSQLDB(db *sql.DB) {
	if p.MaxOpenConns > 0 {
		db.SetMaxOpenConns(p.MaxOpenConns)
	}
	if p.MaxIdleConns > 0 {
		db.SetMaxIdleConns(p.MaxIdleConns)
	}
	if p.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(p.ConnMaxIdleTime)
	}
	if p.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(p.ConnMaxLifetime)
	}
}
