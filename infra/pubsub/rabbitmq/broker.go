package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

var (
	ErrConnectionFailed       = errors.New("rabbitmq connection failed")
	ErrConnectionNotAvailable = errors.New("rabbitmq connection not available")
	ErrDeclarationFailed      = errors.New("rabbitmq AMQP entity declaration failed")
)

type Broker interface {
	Channel(ctx context.Context) (*amqp091.Channel, error)
	DeclareExchange(ctx context.Context, cfg *ExchangeConfig) error
	DeclareQueue(ctx context.Context, cfg *QueueConfig, exchange *ExchangeConfig, routingKey string) error
	BindExchange(ctx context.Context, destination, source, routingKey string, noWait bool, args amqp091.Table) error
	Close() error
}

var _ Broker = (*Connection)(nil)

type Connection struct {
	cfg         *Config
	conn        *amqp091.Connection
	ch          *amqp091.Channel
	mu          sync.RWMutex
	done        chan struct{}
	reconnectCh chan struct{}
	logger      Logger
}

func NewConnection(cfg *Config, logger Logger) (*Connection, error) {
	if logger == nil {
		// by default no-operation logger
		logger = &NoopLogger{}
	}

	b := &Connection{
		cfg:         cfg,
		done:        make(chan struct{}),
		reconnectCh: make(chan struct{}, 1),
		logger:      logger,
	}

	if err := b.connect(); err != nil {
		return nil, fmt.Errorf("broker creation: %w", err)
	}

	go b.connectionWatcher()
	return b, nil
}

func (b *Connection) connect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	conn, err := amqp091.DialConfig(b.cfg.URL, amqp091.Config{
		Dial: amqp091.DefaultDial(b.cfg.ConnectTimeout),
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel after connection: %w", err)
	}

	b.conn = conn
	b.ch = channel
	b.logger.Info("connected to RabbitMQ")
	return nil
}

func (b *Connection) Channel(ctx context.Context) (*amqp091.Channel, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled while getting channel: %w", ctx.Err())
	default:
		if b.conn == nil || b.conn.IsClosed() || b.ch == nil || b.ch.IsClosed() {
			return nil, ErrConnectionNotAvailable
		}
		return b.ch, nil
	}
}

func (b *Connection) DeclareExchange(ctx context.Context, cfg *ExchangeConfig) error {
	ch, err := b.Channel(ctx)
	if err != nil {
		return fmt.Errorf("get channel for exchange declaration: %w", err)
	}

	err = ch.ExchangeDeclare(
		cfg.Name,
		string(cfg.Type),
		cfg.Durable,
		cfg.AutoDelete,
		false, // internal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationFailed, err)
	}

	return nil
}

func (b *Connection) BindExchange(ctx context.Context, source, destination, routingKey string, noWait bool, args amqp091.Table) error {
	ch, err := b.Channel(ctx)
	if err != nil {
		return fmt.Errorf("get channel for exchange bind: %w", err)
	}

	if err := ch.ExchangeBind(
		destination,
		routingKey,
		source,
		noWait,
		args,
	); err != nil {
		return fmt.Errorf("bind exchange: %w", err)
	}

	return nil
}

func (b *Connection) DeclareQueue(ctx context.Context, cfg *QueueConfig, exchange *ExchangeConfig, routingKey string) error {
	ch, err := b.Channel(ctx)
	if err != nil {
		return fmt.Errorf("get channel for queue declaration: %w", err)
	}

	_, err = ch.QueueDeclare(
		cfg.Name,
		cfg.Durable,
		cfg.AutoDelete,
		cfg.Exclusive,
		false, // noWait
		cfg.Arguments,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationFailed, err)
	}

	if exchange != nil && routingKey != "" {
		err = ch.QueueBind(
			cfg.Name,
			routingKey,
			exchange.Name,
			false, // noWait
			nil,   // args
		)
		if err != nil {
			return fmt.Errorf("bind queue: %w", err)
		}
	}

	return nil
}

func (b *Connection) connectionWatcher() {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("panic in connection watcher", fmt.Errorf("%v", r))
		}
	}()

	for {
		select {
		case <-b.done:
			b.logger.Info("connection watcher stopping due to done signal")
			return
		default:
			b.mu.RLock()
			conn := b.conn
			b.mu.RUnlock()

			if conn == nil {
				b.logger.Warn("connection watcher stopped: connection is nil")
				return
			}

			notifyClose := conn.NotifyClose(make(chan *amqp091.Error, 2))
			select {
			case err := <-notifyClose:
				if err != nil {
					b.logger.Error("connection closed", fmt.Errorf("%v", err))
					b.reconnect()
				}
			case <-b.done:
				b.logger.Info("connection watcher stopping due to done signal")
				return
			}
		}
	}
}

func (b *Connection) reconnect() {
	const maxRetryInterval = 30 * time.Second
	retryInterval := time.Second

	for i := 0; ; i++ {
		select {
		case <-b.done:
			b.logger.Info("reconnect stopped due to done signal")
			return
		default:
			if err := b.connect(); err == nil {
				b.logger.Info("successfully reconnected to RabbitMQ")
				return
			}

			if i%5 == 0 {
				b.logger.Warn("reconnection attempt failed",
					"attempt", i+1,
					"retry_interval", retryInterval,
				)
			}
			time.Sleep(retryInterval)
			retryInterval = minDuration(retryInterval*2, maxRetryInterval)
		}
	}
}

func (b *Connection) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	select {
	case <-b.done:
		return errors.New("broker already closed")
	default:
		close(b.done)
	}

	if b.ch != nil && !b.ch.IsClosed() {
		_ = b.ch.Close()
	}
	if b.conn != nil && !b.conn.IsClosed() {
		return b.conn.Close()
	}
	b.logger.Info("broker closed")
	return nil
}

// Helper function to get minimum of two durations
func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
