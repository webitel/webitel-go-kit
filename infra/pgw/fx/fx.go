// Package pgwfx wires pgw.PoolManager into an fx application.
package pgwfx

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/infra/health"
	"github.com/webitel/webitel-go-kit/infra/pgw"
)

// Config for Module.
type Config struct {
	ApplicationName string
	DSN             string
	MaxConns        int
	MinConns        int
	StandbyDSN      []string
	RegisterHealth  bool
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

// Module provides *pgw.PoolManager.
func Module(cfg Config, opts ...pgw.ConfigOption) fx.Option {
	return fx.Module("pgw",
		fx.Provide(func(p params) (*pgw.PoolManager, error) {
			return provide(cfg, p, opts)
		}),
		fx.Invoke(func(*pgw.PoolManager) {}),
	)
}

func provide(cfg Config, p params, opts []pgw.ConfigOption) (*pgw.PoolManager, error) {
	manager, err := pgw.New(cfg.options(opts)...)
	if err != nil {
		return nil, fmt.Errorf("create pgw pool manager: %w", err)
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := manager.Start(ctx); err != nil {
				manager.Close() // fx skips OnStop when this OnStart fails
				return fmt.Errorf("start pgw pool: %w", err)
			}

			return nil
		},
		OnStop: func(context.Context) error {
			manager.Close()

			return nil
		},
	})

	if cfg.RegisterHealth && p.Health != nil {
		p.Health.Informational(health.SystemPostgreSQL, func(context.Context) error {
			_, err := manager.Primary()

			return err
		})
	}

	return manager, nil
}
