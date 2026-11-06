package logradapter

import "github.com/webitel/webitel-go-kit/pkg/logging"

type config struct {
	component logging.Component
	verbosity func(v int) logging.Level
}

type Option func(*config)

// WithComponent sets the instrumentation scope instead of [Component].
func WithComponent(c logging.Component) Option {
	return func(cfg *config) {
		cfg.component = c
	}
}

// WithVerbosity sets the level of a logr verbosity. The default maps V(n)
// to INFO-n, the same as [logr.FromSlogHandler].
func WithVerbosity(f func(v int) logging.Level) Option {
	return func(cfg *config) {
		if f == nil {
			return
		}
		cfg.verbosity = func(v int) (lvl logging.Level) {
			defer func() {
				if r := recover(); r != nil {
					// fallback to INFO
					lvl = logging.LevelInfo
					return
				}
			}()
			return f(v)
		}
	}
}
