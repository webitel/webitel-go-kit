package wlog_provider

type config struct {
	logType string
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func WithLogType(logType string) Option {
	return optionFunc(func(c *config) {
		c.logType = logType
	})
}
