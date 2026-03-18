package profiler

import (
	"context"

	"go.uber.org/fx"
)

func NewWithFx(lc fx.Lifecycle, config Config, logger Logger) *Profiler {
	p := New(config, logger)
	if p == nil {
		return nil
	}

	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return p.Start()
	}, OnStop: p.Stop})

	return p
}
