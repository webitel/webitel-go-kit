package otlp_provider

import "time"

type config struct {
	address             string
	requestTimeoutAfter time.Duration
}

type Option interface {
	apply(*config)
}

type optFunc func(c *config)

func (o optFunc) apply(config2 *config) {
	o(config2)
}

func WithAddress(address string) Option {
	return optFunc(func(c *config) {
		c.address = address
	})
}

func WithTimeout(timeoutAfter time.Duration) Option {
	return optFunc(func(c *config) {
		c.requestTimeoutAfter = timeoutAfter
	})
}
