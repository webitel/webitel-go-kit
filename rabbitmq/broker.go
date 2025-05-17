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
	ErrConnectionFailed       = errors.New("connection failed")
	ErrConnectionNotAvailable = errors.New("connection not available")
	ErrChannelCreationFailed  = errors.New("channel creation failed")
	ErrDeclarationFailed      = errors.New("AMQP entity declaration failed")
)

type Broker interface {
	GetChannel(ctx context.Context) (*amqp.Channel, error)
	DeclareExchange(ctx context.Context) error
	DeclareQueue(ctx context.Context) error
	Close() error
}

var _ Broker = (*ConnectionBroker)(nil) // Ensure interface implementation

type ConnectionBroker struct {
	cfg         *RabbitConfig
	conn        *amqp091.Connection
	mu          sync.RWMutex
	done        chan struct{}
	reconnectCh chan struct{}
}

func NewConnectionBroker(cfg *RabbitConfig) (*ConnectionBroker, error) {
	b := &ConnectionBroker{
		cfg:         cfg,
		done:        make(chan struct{}),
		reconnectCh: make(chan struct{}, 1),
	}

	if err := b.connect(); err != nil {
		return nil, fmt.Errorf("failed to create broker: %w", err)
	}

	go b.connectionWatcher()
	return b, nil
}

func (b *ConnectionBroker) connect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	conn, err := amqp091.DialConfig(b.cfg.URL, amqp091.Config{
		Dial: amqp091.DefaultDial(b.cfg.ConnectTimeout),
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	b.conn = conn
	return nil
}

func (b *ConnectionBroker) GetChannel(ctx context.Context) (*amqp091.Channel, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled while getting channel: %w", ctx.Err())
	default:
		if b.conn == nil || b.conn.IsClosed() {
			return nil, ErrConnectionNotAvailable
		}

		ch, err := b.conn.Channel()
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrChannelCreationFailed, err)
		}

		if err := ch.Qos(b.cfg.PrefetchCount, 0, false); err != nil {
			err := ch.Close()
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("failed to set QoS: %w", err)
		}

		return ch, nil
	}
}

func (b *ConnectionBroker) DeclareExchange(ctx context.Context) error {
	ch, err := b.GetChannel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get channel for exchange declaration: %w", err)
	}
	defer func(ch *amqp091.Channel) {
		err := ch.Close()
		if err != nil {
			Logger().Error("failed to close channel", "error", err)
		}
	}(ch)

	err = ch.ExchangeDeclare(
		b.cfg.Exchange.Name,
		b.cfg.Exchange.Type,
		b.cfg.Exchange.Durable,
		b.cfg.Exchange.AutoDelete,
		false, // internal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationFailed, err)
	}

	return nil
}

func (b *ConnectionBroker) DeclareQueue(ctx context.Context) error {
	ch, err := b.GetChannel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get channel for queue declaration: %w", err)
	}
	defer func(ch *amqp091.Channel) {
		err := ch.Close()
		if err != nil {

		}
	}(ch)

	_, err = ch.QueueDeclare(
		b.cfg.Queue.Name,
		b.cfg.Queue.Durable,
		b.cfg.Queue.AutoDelete,
		b.cfg.Queue.Exclusive,
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeclarationFailed, err)
	}

	// Bind queue to exchange if routing key is specified
	if b.cfg.RoutingKey != "" {
		err = ch.QueueBind(
			b.cfg.Queue.Name,
			b.cfg.RoutingKey,
			b.cfg.Exchange.Name,
			false, // noWait
			nil,   // args
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue: %w", err)
		}
	}

	return nil
}

func (b *ConnectionBroker) connectionWatcher() {
	defer func() {
		if r := recover(); r != nil {
			Logger().Error("panic in connection watcher", "recover", r)
		}
	}()

	for {
		select {
		case <-b.done:
			return
		default:
			b.mu.RLock()
			conn := b.conn
			b.mu.RUnlock()

			if conn == nil {
				return
			}

			notifyClose := conn.NotifyClose(make(chan *amqp091.Error, 2))
			select {
			case err := <-notifyClose:
				if err != nil {
					Logger().Error("connection closed", "error", err)
					b.reconnect()
				}
			case <-b.done:
				return
			}
		}
	}
}

func (b *ConnectionBroker) reconnect() {
	const maxRetryInterval = 30 * time.Second
	retryInterval := time.Second

	for i := 0; ; i++ {
		select {
		case <-b.done:
			return
		default:
			if err := b.connect(); err == nil {
				Logger().Info("successfully reconnected to RabbitMQ")
				return
			}

			if i%5 == 0 { // Log every 5th attempt
				Logger().Warn("reconnection attempt failed",
					"attempt", i+1,
					"retry_interval", retryInterval,
				)
			}

			time.Sleep(retryInterval)
			retryInterval = min(retryInterval*2, maxRetryInterval)
		}
	}
}

func (b *ConnectionBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	select {
	case <-b.done:
		return errors.New("broker already closed")
	default:
		close(b.done)
	}

	if b.conn != nil && !b.conn.IsClosed() {
		if err := b.conn.Close(); err != nil {
			return fmt.Errorf("error closing connection: %w", err)
		}
	}

	return nil
}
