package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrConsumerChannelClosed = errors.New("consumer channel closed")
	ErrConsumerStartFailed   = errors.New("consumer start failed")
	ErrShutdownTimeout       = errors.New("shutdown timeout")
)

type HandleFunc func(ctx context.Context, msg amqp.Delivery) error

type Consumer interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

var _ Consumer = (*MessageConsumer)(nil)

type MessageConsumer struct {
	broker    *ConnectionBroker
	cfg       *RabbitConfig
	handler   HandleFunc
	wg        sync.WaitGroup
	cancel    context.CancelFunc
	workerSem chan struct{}
}

func NewConsumer(
	broker *ConnectionBroker,
	cfg *RabbitConfig,
	handler HandleFunc,
) *MessageConsumer {
	return &MessageConsumer{
		broker:    broker,
		cfg:       cfg,
		handler:   handler,
		workerSem: make(chan struct{}, cfg.Consumer.MaxWorkers),
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
			ch, err := c.broker.GetChannel(ctx)
			if err != nil {
				Logger().Error("failed to get channel", "error", err)
				time.Sleep(retryDelay)
				retryDelay = min(retryDelay*2, c.cfg.Consumer.ReconnectDelay)
				continue
			}

			if err := c.consumeMessages(ctx, ch); err != nil {
				Logger().Error("consuming failed", "error", err)
				err := ch.Close()
				if err != nil {
					return
				}
				time.Sleep(retryDelay)
				retryDelay = min(retryDelay*2, c.cfg.Consumer.ReconnectDelay)
				continue
			}

			retryDelay = time.Second
		}
	}
}

func (c *MessageConsumer) consumeMessages(ctx context.Context, ch *amqp091.Channel) error {
	msgs, err := ch.Consume(
		c.cfg.Queue.Name,
		c.cfg.Consumer.Tag,
		false, // autoAck
		false, // exclusive
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

func (c *MessageConsumer) processMessage(ctx context.Context, msg amqp091.Delivery) {
	defer func() {
		<-c.workerSem
		c.wg.Done()
	}()

	processCtx, cancel := context.WithTimeout(ctx, c.cfg.Consumer.ProcessingTimeout)
	defer cancel()

	if err := c.handler(processCtx, msg); err != nil {
		Logger().Error("message handling failed", "error", err)
		if err := msg.Nack(false, !msg.Redelivered); err != nil {
			Logger().Error("failed to Nack message", "error", err)
		}
		return
	}

	if err := msg.Ack(false); err != nil {
		Logger().Error("failed to Ack message", "error", err)
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
