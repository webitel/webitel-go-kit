package healthhttp

import "log/slog"

// Option configures a handler or a Server.
type Option func(*options)

type options struct {
	log *slog.Logger
}

// WithLogger sets the logger. A nil logger is ignored.
func WithLogger(log *slog.Logger) Option {
	return func(o *options) {
		if log == nil {
			return
		}

		o.log = log
	}
}

// newOptions applies opts over a discarding logger: a library must not write to
// stderr uninvited, so the default is never slog.Default.
func newOptions(opts []Option) options {
	o := options{log: slog.New(slog.DiscardHandler)}
	for _, opt := range opts {
		opt(&o)
	}

	return o
}
