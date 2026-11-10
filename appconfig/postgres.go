package appconfig

import (
	"database/sql"
	"strings"
	"time"
)

// Postgres holds PostgreSQL connection and connection-pool configuration.
//
// Env vars: POSTGRES_DSN, POSTGRES_STANDBY_DSN, POSTGRES_MAX_OPEN_CONNS,
// POSTGRES_MAX_IDLE_CONNS, POSTGRES_CONN_MAX_IDLE_TIME, POSTGRES_CONN_MAX_LIFETIME.
type Postgres struct {
	DSN        string   `mapstructure:"dsn"`
	StandbyDSN []string `mapstructure:"standby_dsn"`

	// Connection pool — zero values mean "use driver default".
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// StandbyDSNs returns trimmed non-empty standby DSNs.
func (p Postgres) StandbyDSNs() []string {
	if len(p.StandbyDSN) == 0 {
		return nil
	}

	out := make([]string, 0, len(p.StandbyDSN))
	for _, dsn := range p.StandbyDSN {
		s := strings.TrimSpace(dsn)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
