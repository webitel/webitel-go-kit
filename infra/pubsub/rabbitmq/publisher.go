package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrPublishFailed  = errors.New("rabbitmq message publish failed")
	ErrPublishTimeout = errors.New("rabbitmq publish confirmation timeout")
	ErrMessageNacked  = errors.New("rabbitmq message nacked by broker")
)

type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte, headers amqp.Table) error
	Close() error
}

var _ Publisher = (*MessagePublisher)(nil)

// PublisherConfig holds configuration for the publisher.
type PublisherConfig struct {
	MaxRetries          int
	ConfirmationTimeout time.Duration
}

// PublisherOption defines a function to modify PublisherConfig.
type PublisherOption func(*PublisherConfig)

// NewPublisherConfig creates a PublisherConfig and applies options.
// Validation is done after applying options.
func NewPublisherConfig(opts ...PublisherOption) (*PublisherConfig, error) {
	cfg := &PublisherConfig{
		MaxRetries:          3,               // default 3 retries
		ConfirmationTimeout: 5 * time.Second, // default 5s timeout
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.MaxRetries <= 0 {
		return nil, errors.New("publisher max retries must be > 0")
	}
	if cfg.ConfirmationTimeout <= 0 {
		return nil, errors.New("publisher confirmation timeout must be > 0")
	}

	return cfg, nil
}

// WithPublisherMaxRetries sets max retries for publishing a message.
func WithPublisherMaxRetries(retries int) PublisherOption {
	return func(c *PublisherConfig) {
		c.MaxRetries = retries
	}
}

// WithPublisherConfirmationTimeout sets confirmation timeout duration.
func WithPublisherConfirmationTimeout(timeout time.Duration) PublisherOption {
	return func(c *PublisherConfig) {
		c.ConfirmationTimeout = timeout
	}
}

type MessagePublisher struct {
	broker    *Connection
	exchange  *ExchangeConfig
	config    *PublisherConfig
	channel   *amqp.Channel
	confirmCh <-chan amqp.Confirmation
	mu        sync.Mutex
	logger    Logger
}

func NewPublisher(
	broker *Connection,
	exchange *ExchangeConfig,
	config *PublisherConfig,
	logger Logger,
) (*MessagePublisher, error) {
	ch, err := broker.Channel(context.Background())
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}

	if err := ch.Confirm(false); err != nil {
		return nil, fmt.Errorf("enable confirm mode: %w", err)
	}

	return &MessagePublisher{
		broker:    broker,
		exchange:  exchange,
		config:    config,
		channel:   ch,
		confirmCh: ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
		logger:    logger,
	}, nil
}

func (p *MessagePublisher) Publish(
	ctx context.Context,
	routingKey string,
	body []byte,
	headers amqp.Table,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for attempt := 0; attempt < p.config.MaxRetries; attempt++ {
		err := p.publishWithConfirmation(ctx, routingKey, body, headers)
		if err == nil {
			return nil
		}

		p.logger.Warn("publish attempt failed", "attempt", attempt+1, "error", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}

	return fmt.Errorf("%w after %d attempts", ErrPublishFailed, p.config.MaxRetries)
}

func (p *MessagePublisher) publishWithConfirmation(
	ctx context.Context,
	routingKey string,
	body []byte,
	headers amqp.Table,
) error {
	err := p.channel.Publish(
		p.exchange.Name,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Headers:     headers,
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	select {
	case confirm := <-p.confirmCh:
		if !confirm.Ack {
			return ErrMessageNacked
		}
		return nil
	case <-time.After(p.config.ConfirmationTimeout):
		return ErrPublishTimeout
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *MessagePublisher) Close() error {
	return p.channel.Close()
}
