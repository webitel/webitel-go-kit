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
	ErrPublishFailed     = errors.New("rabbitmq message publish failed")
	ErrPublishTimeout    = errors.New("rabbitmq publish confirmation timeout")
	ErrMessageNacked     = errors.New("rabbitmq message nacked by broker")
	ErrMessageUnroutable = errors.New("rabbitmq message matched no queue")
)

// confirmBuffer holds confirmations and returns that arrive while a publish is
// still being accounted for.
const confirmBuffer = 16

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
	Persistent          bool
	Mandatory           bool
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

// WithPersistentDelivery marks messages persistent, so a durable queue keeps
// them across a broker restart.
func WithPersistentDelivery(persistent bool) PublisherOption {
	return func(c *PublisherConfig) {
		c.Persistent = persistent
	}
}

// WithMandatory reports a message that matched no queue as an error instead of
// letting the broker discard it silently.
func WithMandatory(mandatory bool) PublisherOption {
	return func(c *PublisherConfig) {
		c.Mandatory = mandatory
	}
}

type MessagePublisher struct {
	broker    *Connection
	config    *PublisherConfig
	channel   *amqp.Channel
	confirmCh <-chan amqp.Confirmation
	returnCh  <-chan amqp.Return
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

	if err := p.ensureChannel(); err != nil && !broker.cfg.LazyConnect {
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
	var lastErr error

	for attempt := 0; attempt < p.config.MaxRetries; attempt++ {
		err := p.doPublish(ctx, exchange, routingKey, body, headers)
		if err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		if errors.Is(err, ErrMessageUnroutable) {
			return err
		}

		lastErr = err

		p.logger.Warn("publish attempt failed", "attempt", attempt+1, "error", err)

		backoff := time.Duration(attempt+1) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return fmt.Errorf("%w after %d attempts: %w", ErrPublishFailed, p.config.MaxRetries, lastErr)
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

	if p.config.Persistent {
		msg.DeliveryMode = amqp.Persistent
	}

	tag := p.channel.GetNextPublishSeqNo()

	err := p.channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		p.config.Mandatory,
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

	return p.awaitConfirm(ctx, tag, timeoutCh)
}

// awaitConfirm waits for the broker to confirm the publish carrying tag,
// skipping confirmations left over by an earlier publish that gave up.
func (p *MessagePublisher) awaitConfirm(ctx context.Context, tag uint64, timeoutCh <-chan time.Time) error {
	for {
		select {
		case confirm, ok := <-p.confirmCh:
			if !ok {
				return errors.New("confirm channel closed")
			}

			if confirm.DeliveryTag < tag {
				continue
			}

			if !confirm.Ack {
				return ErrMessageNacked
			}

			return p.routed()
		case <-timeoutCh:
			return ErrPublishTimeout
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// routed reports an unroutable publish. The broker sends the return before the
// confirmation of the same message, so by now it is already buffered.
func (p *MessagePublisher) routed() error {
	if !p.config.Mandatory {
		return nil
	}

	select {
	case ret := <-p.returnCh:
		return fmt.Errorf("%w: %s (%d)", ErrMessageUnroutable, ret.ReplyText, ret.ReplyCode)
	default:
		return nil
	}
}

func (p *MessagePublisher) ensureChannel() error {
	if p.channel != nil && !p.channel.IsClosed() {
		return nil
	}

	ch, err := p.broker.NewChannel(context.Background())
	if err != nil {
		return fmt.Errorf("recreate channel: %w", err)
	}

	if p.config.Confirmation {
		if err := ch.Confirm(false); err != nil {
			_ = ch.Close()
			return fmt.Errorf("enable confirm mode: %w", err)
		}

		p.confirmCh = ch.NotifyPublish(make(chan amqp.Confirmation, confirmBuffer))
	} else {
		p.confirmCh = nil
	}

	if p.config.Mandatory {
		p.returnCh = ch.NotifyReturn(make(chan amqp.Return, confirmBuffer))
	} else {
		p.returnCh = nil
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
