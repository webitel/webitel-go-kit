package rabbitmq

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

var (
	ErrInvalidConfig = errors.New("invalid configuration")
)

type RabbitConfig struct {
	URL            string `mapstructure:"url"`
	Exchange       ExchangeConfig
	Queue          QueueConfig
	Consumer       ConsumerConfig
	Publisher      PublisherConfig
	PrefetchCount  int           `mapstructure:"prefetch"`
	RoutingKey     string        `mapstructure:"routing_key"`
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
}

type ExchangeConfig struct {
	Name       string `mapstructure:"name"`
	Type       string `mapstructure:"type"`
	Durable    bool   `mapstructure:"durable"`
	AutoDelete bool   `mapstructure:"auto_delete"`
}

type QueueConfig struct {
	Name       string `mapstructure:"name"`
	Durable    bool   `mapstructure:"durable"`
	AutoDelete bool   `mapstructure:"auto_delete"`
	Exclusive  bool   `mapstructure:"exclusive"`
}

type ConsumerConfig struct {
	Tag               string        `mapstructure:"tag"`
	ReconnectDelay    time.Duration `mapstructure:"reconnect_delay"`
	MaxWorkers        int           `mapstructure:"max_workers"`
	ProcessingTimeout time.Duration `mapstructure:"processing_timeout"`
}

type PublisherConfig struct {
	ConfirmationTimeout time.Duration `mapstructure:"confirmation_timeout"`
	MaxRetries          int           `mapstructure:"max_retries"`
}

func LoadFromEnv() (*RabbitConfig, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("RABBITMQ")

	setDefaults(v)

	var cfg RabbitConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	return &cfg, validateConfig(&cfg)
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("exchange.type", "direct")
	v.SetDefault("exchange.durable", true)
	v.SetDefault("queue.durable", true)
	v.SetDefault("consumer.tag", "rabbitmq-consumer")
	v.SetDefault("consumer.reconnect_delay", 5*time.Second)
	v.SetDefault("consumer.max_workers", 10)
	v.SetDefault("consumer.processing_timeout", 30*time.Second)
	v.SetDefault("publisher.confirmation_timeout", 5*time.Second)
	v.SetDefault("publisher.max_retries", 3)
	v.SetDefault("routing_key", "")
	v.SetDefault("prefetch", 10)
	v.SetDefault("connect_timeout", 30*time.Second)
}

func validateConfig(cfg *RabbitConfig) error {
	if cfg.URL == "" {
		return fmt.Errorf("%w: missing URL", ErrInvalidConfig)
	}
	if cfg.Exchange.Name == "" {
		return fmt.Errorf("%w: missing exchange name", ErrInvalidConfig)
	}
	if cfg.Queue.Name == "" {
		return fmt.Errorf("%w: missing queue name", ErrInvalidConfig)
	}
	return nil
}
