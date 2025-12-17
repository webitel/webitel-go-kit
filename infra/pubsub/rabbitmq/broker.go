package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
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
	BindExchange(ctx context.Context, destination, source, routingKey string, noWait bool, args amqp091.Table) error
	DeclareQueue(ctx context.Context, cfg *QueueConfig, exchange *ExchangeConfig, routingKey string) error
	BindQueue(queueName string, rk string, exchange string, noWait bool, args amqp091.Table) error
	Close() error
}

var _ Broker = (*Connection)(nil)

type Connection struct {
	cfg    *Config
	conn   *amqp091.Connection
	ch     *amqp091.Channel
	mu     sync.RWMutex
	done   chan struct{}
	logger Logger

	reconnecting atomic.Bool
}

func NewConnection(cfg *Config, logger Logger) (*Connection, error) {
	if logger == nil {
		logger = &NoopLogger{}
	}

	b := &Connection{
		cfg:    cfg,
		done:   make(chan struct{}),
		logger: logger,
	}

	if err := b.connect(); err != nil {
		return nil, fmt.Errorf("broker creation: %w", err)
	}

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

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	// close old resources
	if b.ch != nil && !b.ch.IsClosed() {
		_ = b.ch.Close()
	}
	if b.conn != nil && !b.conn.IsClosed() {
		_ = b.conn.Close()
	}

	b.conn = conn
	b.ch = ch

	go b.watchConn(conn)

	b.logger.Info("connected to RabbitMQ")
	return nil
}

func (b *Connection) watchConn(conn *amqp091.Connection) {
	notifyClose := conn.NotifyClose(make(chan *amqp091.Error, 1))

	select {
	case err := <-notifyClose:
		b.logger.Warn("rabbitmq connection closed", err)
		b.reconnect()
	case <-b.done:
		return
	}
}

func (b *Connection) reconnect() {
	if !b.reconnecting.CompareAndSwap(false, true) {
		return
	}
	defer b.reconnecting.Store(false)

	const maxRetry = 30 * time.Second
	retry := time.Second

	for {
		select {
		case <-b.done:
			return
		default:
			if err := b.connect(); err == nil {
				b.logger.Info("successfully reconnected to RabbitMQ")
				return
			}

			time.Sleep(retry)
			retry = minDuration(retry*2, maxRetry)
		}
	}
}

func (b *Connection) Channel(ctx context.Context) (*amqp091.Channel, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
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
		return err
	}

	if err := ch.ExchangeDeclare(
		cfg.Name,
		string(cfg.Type),
		cfg.Durable,
		cfg.AutoDelete,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationFailed, err)
	}

	return nil
}

func (b *Connection) BindExchange(
	ctx context.Context,
	destination, source, routingKey string,
	noWait bool,
	args amqp091.Table,
) error {
	ch, err := b.Channel(ctx)
	if err != nil {
		return err
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

func (b *Connection) DeclareQueue(
	ctx context.Context,
	cfg *QueueConfig,
	exchange *ExchangeConfig,
	routingKey string,
) error {
	ch, err := b.Channel(ctx)
	if err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		cfg.Name,
		cfg.Durable,
		cfg.AutoDelete,
		cfg.Exclusive,
		false,
		cfg.Arguments,
	); err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationFailed, err)
	}

	if exchange != nil && routingKey != "" {
		if err := ch.QueueBind(
			cfg.Name,
			routingKey,
			exchange.Name,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("bind queue: %w", err)
		}
	}

	return nil
}

func (b *Connection) BindQueue(queueName string, rk string, exchange string, noWait bool, args amqp091.Table) error {
	ch, err := b.Channel(context.Background())
	if err != nil {
		return fmt.Errorf("get channel for exchange declaration: %w", err)
	}

	err = ch.QueueBind(queueName, rk, exchange, noWait, args)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationFailed, err)
	}

	b.logger.Info("queue binded")
	return nil
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

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
