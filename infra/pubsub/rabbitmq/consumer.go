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
	ErrConsumerChannelClosed = errors.New("rabbitmq consumer channel closed")
	ErrConsumerStartFailed   = errors.New("rabbitmq consumer start failed")
	ErrShutdownTimeout       = errors.New("rabbitmq shutdown timeout")
)

type HandleFunc func(ctx context.Context, msg amqp.Delivery) error

type Consumer interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

var _ Consumer = (*MessageConsumer)(nil)

// ConsumerConfig holds configuration for the consumer.
type ConsumerConfig struct {
	Tag               string
	MaxWorkers        int
	ReconnectDelay    time.Duration
	ProcessingTimeout time.Duration
}

// ConsumerOption defines a function to modify ConsumerConfig.
type ConsumerOption func(*ConsumerConfig)

// NewConsumerConfig creates a ConsumerConfig with required tag and applies options.
// Validation is done after applying options.
func NewConsumerConfig(tag string, opts ...ConsumerOption) (*ConsumerConfig, error) {
	if tag == "" {
		return nil, errors.New("tag is required")
	}

	cfg := &ConsumerConfig{
		Tag:               tag,
		MaxWorkers:        1,                // default 1 worker
		ReconnectDelay:    5 * time.Second,  // default 5s reconnect delay
		ProcessingTimeout: 30 * time.Second, // default 30s processing timeout
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.MaxWorkers <= 0 {
		return nil, errors.New("max workers must be > 0")
	}
	if cfg.ReconnectDelay <= 0 {
		return nil, errors.New("reconnect delay must be > 0")
	}
	if cfg.ProcessingTimeout <= 0 {
		return nil, errors.New("processing timeout must be > 0")
	}

	return cfg, nil
}

// WithConsumerMaxWorkers sets the number of concurrent workers for the consumer.
func WithConsumerMaxWorkers(workers int) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.MaxWorkers = workers
	}
}

// WithConsumerReconnectDelay sets delay before reconnect attempts.
func WithConsumerReconnectDelay(d time.Duration) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.ReconnectDelay = d
	}
}

// WithConsumerProcessingTimeout sets timeout for processing each message.
func WithConsumerProcessingTimeout(d time.Duration) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.ProcessingTimeout = d
	}
}

type MessageConsumer struct {
	broker    *Connection
	queue     *QueueConfig
	consumer  *ConsumerConfig
	handler   HandleFunc
	wg        sync.WaitGroup
	cancel    context.CancelFunc
	workerSem chan struct{}
	logger    Logger
}

func NewConsumer(
	broker *Connection,
	queueCfg *QueueConfig,
	consumerCfg *ConsumerConfig,
	handler HandleFunc,
	logger Logger,
) *MessageConsumer {
	return &MessageConsumer{
		broker:    broker,
		queue:     queueCfg,
		consumer:  consumerCfg,
		handler:   handler,
		logger:    logger,
		workerSem: make(chan struct{}, consumerCfg.MaxWorkers),
	}
}

func (c *MessageConsumer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.wg.Add(1)
	go c.consumeLoop(ctx)
	return nil
}

func (c *MessageConsumer) consumeLoop(ctx context.Context) {
	defer c.wg.Done()
	retryDelay := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
			ch, err := c.broker.Channel(ctx)
			if err != nil {
				c.logger.Error("get channel", err)
				time.Sleep(retryDelay)
				retryDelay = min(retryDelay*2, c.consumer.ReconnectDelay)
				continue
			}

			if err := c.consumeMessages(ctx, ch); err != nil {
				c.logger.Error("consuming failed", err)
				_ = ch.Close()
				time.Sleep(retryDelay)
				retryDelay = min(retryDelay*2, c.consumer.ReconnectDelay)
				continue
			}

			retryDelay = time.Second
		}
	}
}

func (c *MessageConsumer) consumeMessages(ctx context.Context, ch *amqp.Channel) error {
	msgs, err := ch.Consume(
		c.queue.Name,
		c.consumer.Tag,
		false, // autoAck
		c.queue.Exclusive,
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConsumerStartFailed, err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return ErrConsumerChannelClosed
			}

			select {
			case c.workerSem <- struct{}{}:
				c.wg.Add(1)
				go c.processMessage(ctx, msg)
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (c *MessageConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	defer func() {
		<-c.workerSem
		c.wg.Done()
	}()

	processCtx, cancel := context.WithTimeout(ctx, c.consumer.ProcessingTimeout)
	defer cancel()

	if err := c.handler(processCtx, msg); err != nil {
		c.logger.Error("message handling failed", err)
		if err := msg.Nack(false, !msg.Redelivered); err != nil {
			c.logger.Error("failed to Nack message", err)
		}
		return
	}

	if err := msg.Ack(false); err != nil {
		c.logger.Error("ack message", err)
	}
}

func (c *MessageConsumer) Shutdown(ctx context.Context) error {
	c.cancel()

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w: %v", ErrShutdownTimeout, ctx.Err())
	}
}
