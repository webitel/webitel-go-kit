// Package pgwfx wires a pgw.PoolManager into an fx application: it opens the
// pool on start, closes it on stop, and can register an informational health
// check from the primary pool state (no SELECT 1). The module provides *Pool.
package pgwfx

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/infra/health"
	"github.com/webitel/webitel-go-kit/infra/pgw"
)

// Config tunes the module. DSN is required. Zero pool limits leave pgw defaults.
type Config struct {
	// ApplicationName sets PostgreSQL application_name on every connection.
	ApplicationName string
	// DSN is the primary database URL.
	DSN string
	// MaxConns maps to pgw PrimaryConfig.MaxConns. Zero keeps the pgw default.
	MaxConns int
	// MinConns maps to pgw PrimaryConfig.MinConns. Zero keeps the pgw default.
	MinConns int
	// StandbyDSN lists read-only replica URLs. Empty disables standbys.
	StandbyDSN []string
	// RegisterHealth registers Informational(SystemPostgreSQL) when a
	// *health.Registry is in the graph. The check uses Primary() pool state.
	RegisterHealth bool
}

func (c Config) validate() error {
	if c.DSN == "" {
		return errors.New("pgwfx: DSN is required")
	}

	return nil
}

func (c Config) options(extra []pgw.ConfigOption) []pgw.ConfigOption {
	primary := pgw.PrimaryConfig{DSN: c.DSN}
	if c.MaxConns > 0 {
		primary.MaxConns = c.MaxConns
	}
	if c.MinConns > 0 {
		primary.MinConns = c.MinConns
	}

	opts := []pgw.ConfigOption{
		pgw.WithPrimaryConfig(primary),
	}
	if c.ApplicationName != "" {
		opts = append(opts, pgw.WithApplicationName(c.ApplicationName))
	}
	if len(c.StandbyDSN) > 0 {
		opts = append(opts, pgw.WithStandbyConfig(pgw.StandbyConfig{DSN: c.StandbyDSN}))
	}

	return append(opts, extra...)
}

type params struct {
	fx.In

	Lifecycle fx.Lifecycle
	Health    *health.Registry `optional:"true"`
}

// Module provides *Pool. It dials the primary during OnStart and closes it
// during OnStop — the same lifecycle as webitel-ai-hub / webitel-kb
// postgres.Module. With RegisterHealth and a *health.Registry in the graph it
// registers an informational PostgreSQL check based on primary pool state.
func Module(cfg Config, opts ...pgw.ConfigOption) fx.Option {
	return fx.Module("pgw",
		fx.Provide(func(p params) (*Pool, error) {
			return provide(cfg, p, opts)
		}),
		fx.Invoke(func(*Pool) {}),
	)
}

// Pool owns the pgw pool manager.
//
// Ready fails with pgw.ErrUnreachable until OnStart connects the primary.
// A later outage makes Primary return the same error while pgw reconnects.
//
// mu guards the manager pointer so Close can take ownership once: health
// checks may still call Ready while shutdown runs Close.
type Pool struct {
	mu      sync.RWMutex
	manager *pgw.PoolManager
}

func provide(cfg Config, p params, opts []pgw.ConfigOption) (*Pool, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	db := &Pool{}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return db.open(ctx, cfg, opts)
		},
		OnStop: func(context.Context) error {
			db.close()

			return nil
		},
	})

	if cfg.RegisterHealth && p.Health != nil {
		p.Health.Informational(health.SystemPostgreSQL, db.Ready)
	}

	return db, nil
}

func (p *Pool) open(ctx context.Context, cfg Config, opts []pgw.ConfigOption) error {
	manager, err := pgw.NewPoolManager(ctx, cfg.options(opts)...)
	if err != nil {
		return fmt.Errorf("open pgw pool: %w", err)
	}

	p.mu.Lock()
	p.manager = manager
	p.mu.Unlock()

	return nil
}

func (p *Pool) close() {
	p.mu.Lock()
	manager := p.manager
	p.manager = nil
	p.mu.Unlock()

	if manager == nil {
		return
	}

	manager.Close()
}

// Manager returns the live *pgw.PoolManager, or nil before OnStart / after OnStop.
func (p *Pool) Manager() *pgw.PoolManager {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.manager
}

// Primary returns the read-write pool, or pgw.ErrUnreachable when it is down.
func (p *Pool) Primary() (*pgw.Pool, error) {
	manager := p.Manager()
	if manager == nil {
		return nil, pgw.ErrUnreachable
	}

	return manager.Primary()
}

// Ready reports whether the primary pool is in the connected state.
// Prefer this over a round-trip ping: pgw already health-checks the pool.
func (p *Pool) Ready(context.Context) error {
	_, err := p.Primary()

	return err
}
