package sdnotify

import (
	"log/slog"
	"time"
)

// Option configures a Notifier.
type Option func(*options)

type options struct {
	log          *slog.Logger
	writeTimeout time.Duration
	poll         time.Duration
	startTimeout time.Duration
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

// WithWriteTimeout bounds one datagram write. Default a second; non-positive is ignored.
func WithWriteTimeout(d time.Duration) Option {
	return func(o *options) {
		if d <= 0 {
			return
		}

		o.writeTimeout = d
	}
}

// WithPollInterval sets how often the registry is read. Default a second;
// non-positive is ignored.
func WithPollInterval(d time.Duration) Option {
	return func(o *options) {
		if d <= 0 {
			return
		}

		o.poll = d
	}
}

// WithStartTimeout sends READY=1 with STATUS=starting degraded when readiness
// has not arrived within d, so a unit does not hang in activating. Off by
// default. Keep d below the unit's TimeoutStartSec.
func WithStartTimeout(d time.Duration) Option {
	return func(o *options) {
		if d <= 0 {
			return
		}

		o.startTimeout = d
	}
}

// newOptions applies opts over a discarding logger: a library must not write to
// stderr uninvited.
func newOptions(opts []Option) options {
	o := options{
		log:          slog.New(slog.DiscardHandler),
		writeTimeout: time.Second,
		poll:         time.Second,
	}
	for _, opt := range opts {
		opt(&o)
	}

	return o
}
