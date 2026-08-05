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
	Publish(ctx context.Context, exchange string, routingKey string, body []byte, headers amqp.Table) error
	Close() error
}

var _ Publisher = (*MessagePublisher)(nil)

// PublisherConfig holds configuration for the publisher.
type PublisherConfig struct {
	MaxRetries          int
	ConfirmationTimeout time.Duration
	Confirmation        bool
}

// PublisherOption defines a function to modify PublisherConfig.
type PublisherOption func(*PublisherConfig)

// NewPublisherConfig creates a PublisherConfig and applies options.
// Validation is done after applying options.
func NewPublisherConfig(opts ...PublisherOption) (*PublisherConfig, error) {
	cfg := &PublisherConfig{
		MaxRetries:          3,               // default 3 retries
		ConfirmationTimeout: 5 * time.Second, // default 5s timeout
		Confirmation:        true,
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

// WithConfirmation sets need to use confirmation mode.
func WithConfirmation(conf bool) PublisherOption {
	return func(c *PublisherConfig) {
		c.Confirmation = conf
	}
}

type MessagePublisher struct {
	broker    *Connection
	config    *PublisherConfig
	channel   *amqp.Channel
	confirmCh <-chan amqp.Confirmation
	mu        sync.Mutex
	logger    Logger
}

func NewPublisher(
	broker *Connection,
	config *PublisherConfig,
	logger Logger,
) (*MessagePublisher, error) {
	p := &MessagePublisher{
		broker: broker,
		config: config,
		logger: logger,
	}

	if err := p.ensureChannel(); err != nil {
		return nil, fmt.Errorf("init channel: %w", err)
	}

	return p, nil
}

func (p *MessagePublisher) Publish(
	ctx context.Context,
	exchange string,
	routingKey string,
	body []byte,
	headers amqp.Table,
) error {
	for attempt := 0; attempt < p.config.MaxRetries; attempt++ {
		err := p.doPublish(ctx, exchange, routingKey, body, headers)
		if err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		p.logger.Warn("publish attempt failed", "attempt", attempt+1, "error", err)

		backoff := time.Duration(attempt+1) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return fmt.Errorf("%w after %d attempts", ErrPublishFailed, p.config.MaxRetries)
}

func (p *MessagePublisher) doPublish(
	ctx context.Context,
	exchange string,
	routingKey string,
	body []byte,
	headers amqp.Table,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.ensureChannel(); err != nil {
		return fmt.Errorf("ensure channel: %w", err)
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		Headers:     headers,
		Timestamp:   time.Now(),
	}

	err := p.channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		msg,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	if !p.config.Confirmation {
		return nil
	}

	var timeoutCh <-chan time.Time
	if p.config.ConfirmationTimeout > 0 {
		timer := time.NewTimer(p.config.ConfirmationTimeout)
		defer func() {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}()
		timeoutCh = timer.C
	}

	select {
	case confirm, ok := <-p.confirmCh:
		if !ok {
			return errors.New("confirm channel closed")
		}
		if !confirm.Ack {
			return ErrMessageNacked
		}
		return nil
	case <-timeoutCh:
		return ErrPublishTimeout
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *MessagePublisher) ensureChannel() error {
	if p.channel != nil && !p.channel.IsClosed() {
		return nil
	}

	ch, err := p.broker.Channel(context.Background())
	if err != nil {
		return fmt.Errorf("recreate channel: %w", err)
	}

	if p.config.Confirmation {
		if err := ch.Confirm(false); err != nil {
			_ = ch.Close()
			return fmt.Errorf("enable confirm mode: %w", err)
		}

		p.confirmCh = ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	} else {
		p.confirmCh = nil
	}

	p.channel = ch

	return nil
}

func (p *MessagePublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil && !p.channel.IsClosed() {
		return p.channel.Close()
	}

	return nil
}
