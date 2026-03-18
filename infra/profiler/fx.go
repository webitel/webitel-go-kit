package profiler

import "go.uber.org/fx"

func NewWithFx(lc fx.Lifecycle, config Config, logger Logger) *Profiler {
	p := New(config, logger)
	if p == nil {
		return nil
	}

	lc.Append(fx.Hook{OnStart: p.Start, OnStop: p.Stop})

	return p
}
