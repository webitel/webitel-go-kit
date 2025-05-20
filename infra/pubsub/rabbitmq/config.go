package rabbitmq

import (
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

// MQExchangeType represents RabbitMQ exchange types.
type MQExchangeType string

const (
	ExchangeTypeFanout MQExchangeType = "fanout"
	ExchangeTypeTopic  MQExchangeType = "topic"
	ExchangeTypeDirect MQExchangeType = "direct"
)

// Config holds the basic RabbitMQ connection parameters.
type Config struct {
	URL            string
	ConnectTimeout time.Duration
}

// NewConfig creates a new Config with URL and connect timeout validation.
func NewConfig(url string, opts ...ConfigOption) (*Config, error) {
	if url == "" {
		return nil, errors.New("rabbitmq config URL is required")
	}

	cfg := &Config{
		URL:            url,
		ConnectTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg, nil
}

type ConfigOption func(config *Config)

func WithConnectTimeout(timeout time.Duration) ConfigOption {
	return func(config *Config) {
		config.ConnectTimeout = timeout
	}
}

type ExchangeConfig struct {
	Name       string
	Type       MQExchangeType
	Durable    bool
	AutoDelete bool
}

func NewExchangeConfig(name string, exchangeType MQExchangeType, opts ...ExchangeOption) (*ExchangeConfig, error) {
	if name == "" {
		return nil, errors.New("rabbitmq config exchange name is required")
	}
	if exchangeType == "" {
		return nil, errors.New("rabbitmq config exchange type is required")
	}

	cfg := &ExchangeConfig{
		Name:       name,
		Type:       exchangeType,
		Durable:    true, // default value
		AutoDelete: false,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg, nil
}

type ExchangeOption func(*ExchangeConfig)

func WithDurable(durable bool) ExchangeOption {
	return func(c *ExchangeConfig) {
		c.Durable = durable
	}
}

func WithAutoDelete(autoDelete bool) ExchangeOption {
	return func(c *ExchangeConfig) {
		c.AutoDelete = autoDelete
	}
}

type QueueConfig struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	Arguments  amqp.Table
}

type QueueOption func(*QueueConfig)

func NewQueueConfig(name string, opts ...QueueOption) (*QueueConfig, error) {
	if name == "" {
		return nil, errors.New("rabbitmq config queue name is required")
	}

	cfg := &QueueConfig{
		Name:       name,
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		Arguments:  amqp.Table{},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg, nil
}

func WithQueueDurable(durable bool) QueueOption {
	return func(c *QueueConfig) {
		c.Durable = durable
	}
}

func WithQueueAutoDelete(autoDelete bool) QueueOption {
	return func(c *QueueConfig) {
		c.AutoDelete = autoDelete
	}
}

func WithQueueExclusive(exclusive bool) QueueOption {
	return func(c *QueueConfig) {
		c.Exclusive = exclusive
	}
}

func WithQueueArgument(key string, value interface{}) QueueOption {
	return func(c *QueueConfig) {
		if c.Arguments == nil {
			c.Arguments = amqp.Table{}
		}
		c.Arguments[key] = value
	}
}

func WithQueueTypeQuorum() QueueOption {
	return WithQueueArgument("x-queue-type", "quorum")
}
