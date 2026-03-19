package profiler

import (
	"context"

	"github.com/webitel/webitel-go-kit/pkg/logger"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    Config
	Logger    logger.Logger
}

func NewProfiler(p Params) *Profiler {
	prof := New(p.Logger, p.Config)
	if prof == nil {
		return nil
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return prof.Start()
		},
		OnStop: func(ctx context.Context) error {
			return prof.Stop(ctx)
		},
	})

	return prof
}

var Module = fx.Module(
	"profiler",
	fx.Provide(
		NewProfiler,
	),
)
