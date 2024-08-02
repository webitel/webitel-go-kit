package jsonl_provider

type config struct {
	logType     string
	filePath    string
	serviceName string
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

func WithFilePath(path string) Option {
	return optionFunc(func(c *config) {
		c.filePath = path
	})
}

func WithServiceName(name string) Option {
	return optionFunc(func(c *config) {
		c.serviceName = name
	})
}
