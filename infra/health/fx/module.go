package healthfx

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/infra/health"
	healthhttp "github.com/webitel/webitel-go-kit/infra/health/http"
	"github.com/webitel/webitel-go-kit/infra/health/sdnotify"
)

const (
	// DefaultStartTimeout is how long sd_notify waits for the critical checks
	// before it reports READY=1 anyway. It must stay below the unit's
	// TimeoutStartSec.
	DefaultStartTimeout = 60 * time.Second
	// DefaultDrainHold keeps the node not-ready before shutdown continues. It
	// exceeds the 10s Consul TTL heartbeat, so discovery sees the node leave.
	DefaultDrainHold = 15 * time.Second
)

const stopMargin = 5 * time.Second

// Config tunes the module. Zero fields take their defaults.
type Config struct {
	// HTTPAddr is the listen address of a dedicated /livez /readyz /healthz
	// server. Empty disables it; mount the healthhttp handlers on an existing
	// router instead.
	HTTPAddr string
	// StartTimeout defaults to DefaultStartTimeout.
	StartTimeout time.Duration
	// DrainHold defaults to DefaultDrainHold.
	DrainHold time.Duration
	// StopTimeout bounds the drain on shutdown and defaults to DrainHold plus
	// five seconds.
	StopTimeout time.Duration
}

func (c Config) withDefaults() Config {
	if c.StartTimeout <= 0 {
		c.StartTimeout = DefaultStartTimeout
	}

	if c.DrainHold <= 0 {
		c.DrainHold = DefaultDrainHold
	}

	if c.StopTimeout <= 0 {
		c.StopTimeout = c.DrainHold + stopMargin
	}

	return c
}

type settings struct {
	stopTimeout time.Duration
}

// Module provides *health.Registry. It needs a *slog.Logger in the graph. The
// registry starts on the application's start with a context that outlives the
// start hook.
func Module(cfg Config) fx.Option {
	cfg = cfg.withDefaults()

	return fx.Module("health",
		fx.Provide(func(log *slog.Logger, lc fx.Lifecycle) (*health.Registry, settings) {
			return build(cfg, log, lc), settings{stopTimeout: cfg.StopTimeout}
		}),
	)
}

// Shutdown drains the registry when the application stops. fx runs stop hooks
// in reverse order, so place it after every server: the drain then runs before
// any of them stops.
func Shutdown() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, h *health.Registry, s settings) {
		lc.Append(fx.Hook{
			OnStop: func(context.Context) error {
				ctx, cancel := context.WithTimeout(context.Background(), s.stopTimeout)
				defer cancel()

				if err := h.Shutdown(ctx); err != nil {
					return fmt.Errorf("health shutdown: %w", err)
				}

				return nil
			},
		})
	})
}

func build(cfg Config, log *slog.Logger, lc fx.Lifecycle) *health.Registry {
	hcfg := health.DefaultConfig()
	hcfg.DrainHold = cfg.DrainHold

	h := health.New(hcfg, log,
		health.WithTransport(sdnotify.New(
			sdnotify.WithLogger(log),
			sdnotify.WithStartTimeout(cfg.StartTimeout),
		)),
		health.WithTransport(healthhttp.NewServer(cfg.HTTPAddr, healthhttp.WithLogger(log))),
	)

	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			if err := h.Start(ctx); err != nil {
				return fmt.Errorf("start health registry: %w", err)
			}

			return nil
		},
		OnStop: func(context.Context) error {
			cancel()

			return nil
		},
	})

	return h
}
