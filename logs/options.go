package logs

import (
	"github.com/webitel/webitel-go-kit/logs/model"
	"github.com/webitel/wlog"
	"strings"
	"time"
)

type config struct {
	serviceInfo

	exporter string
	address  string
	logger   *wlog.Logger

	requestTimeout time.Duration
	logLevel       model.LogLevel
	fileFormat     string
	filePath       string
}

type serviceInfo struct {
	serviceName    string
	consulId       string
	serviceVersion string
	build          int
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func WithServiceName(service string) Option {
	return optionFunc(func(c *config) {
		c.serviceName = service
	})
}

func WithConsulId(id string) Option {
	return optionFunc(func(c *config) {
		c.consulId = id
	})
}

func WithBuild(version int) Option {
	return optionFunc(func(c *config) {
		c.build = version
	})
}

func WithAddress(address string) Option {
	return optionFunc(func(c *config) {
		c.address = address
	})
}

func WithLogLevel(level string) Option {
	return optionFunc(func(c *config) {
		parsedLevel := strings.ToLower(level)
		for s, logLevel := range model.Levels {
			if s == parsedLevel {
				c.logLevel = logLevel
			}
		}

	})
}

func WithExistingLogger(logger *wlog.Logger) Option {
	return optionFunc(func(c *config) {
		c.logger = logger
	})
}

func WithTimeout(timeoutAfter time.Duration) Option {
	return optionFunc(func(c *config) {
		c.requestTimeout = timeoutAfter
	})
}

func WithExporter(exporter string) Option {
	return optionFunc(func(c *config) {
		c.exporter = exporter
	})
}

func WithFileType(fileType string) Option {
	return optionFunc(func(c *config) {
		c.fileFormat = fileType
	})
}

func WithFilePath(filePath string) Option {
	return optionFunc(func(c *config) {
		c.filePath = filePath
	})
}

func WithServiceVersion(version string) Option {
	return optionFunc(func(c *config) {
		c.serviceVersion = version
	})
}
