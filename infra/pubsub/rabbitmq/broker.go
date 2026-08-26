package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"net"
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

// declaration recreates one AMQP entity on a channel.
type declaration func(ch *amqp091.Channel) error

type Connection struct {
	cfg    *Config
	conn   *amqp091.Connection
	ch     *amqp091.Channel
	mu     sync.RWMutex
	done   chan struct{}
	logger Logger

	reconnecting atomic.Bool

	declMu       sync.Mutex
	declOrder    []string
	declarations map[string]declaration
}

func NewConnection(cfg *Config, logger Logger) (*Connection, error) {
	if logger == nil {
		logger = &NoopLogger{}
	}

	b := &Connection{
		cfg:          cfg,
		done:         make(chan struct{}),
		logger:       logger,
		declarations: make(map[string]declaration),
	}

	if err := b.connect(); err != nil {
		if !cfg.LazyConnect {
			return nil, fmt.Errorf("broker creation: %w", err)
		}

		b.logger.Warn("rabbitmq is not available yet, connecting in the background", "err", err)

		go b.reconnect()
	}

	return b, nil
}

func (b *Connection) connect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	conn, err := amqp091.DialConfig(b.cfg.URL, amqp091.Config{Dial: b.dialer()})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	if err := b.replayDeclarations(ch); err != nil {
		b.logger.Warn("rabbitmq topology was not fully restored", "err", err)

		if ch, err = conn.Channel(); err != nil {
			_ = conn.Close()

			return fmt.Errorf("open channel: %w", err)
		}
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

			// Waiting on done as well, so Close does not leave this running.
			select {
			case <-b.done:
				return
			case <-time.After(retry):
			}

			retry = minDuration(retry*2, maxRetry)
		}
	}
}

// dialer bounds the handshake and, when configured, every later socket write:
// a broker applying TCP pushback cannot block a publish forever.
func (b *Connection) dialer() func(network, addr string) (net.Conn, error) {
	dial := amqp091.DefaultDial(b.cfg.ConnectTimeout)
	if b.cfg.WriteTimeout <= 0 {
		return dial
	}

	return func(network, addr string) (net.Conn, error) {
		conn, err := dial(network, addr)
		if err != nil {
			return nil, err
		}

		return &writeDeadlineConn{Conn: conn, timeout: b.cfg.WriteTimeout}, nil
	}
}

// writeDeadlineConn stamps a deadline on each write; reads stay unbounded
// because the AMQP heartbeat detects a silent peer.
type writeDeadlineConn struct {
	net.Conn

	timeout time.Duration
}

func (c *writeDeadlineConn) Write(b []byte) (int, error) {
	if err := c.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, err
	}

	return c.Conn.Write(b)
}

// remember stores a declaration under key so it can be replayed after a
// reconnect; declaring the same entity again replaces the stored one.
func (b *Connection) remember(key string, declare declaration) {
	b.declMu.Lock()
	defer b.declMu.Unlock()

	if _, ok := b.declarations[key]; !ok {
		b.declOrder = append(b.declOrder, key)
	}

	b.declarations[key] = declare
}

func (b *Connection) replayDeclarations(ch *amqp091.Channel) error {
	b.declMu.Lock()
	defer b.declMu.Unlock()

	for _, key := range b.declOrder {
		if err := b.declarations[key](ch); err != nil {
			return fmt.Errorf("restore %s: %w", key, err)
		}
	}

	if len(b.declOrder) > 0 {
		b.logger.Info("rabbitmq topology restored")
	}

	return nil
}

// declare applies a declaration and remembers it for later reconnects.
func (b *Connection) declare(ctx context.Context, key string, apply declaration) error {
	ch, err := b.Channel(ctx)
	if err != nil {
		return err
	}

	if err := apply(ch); err != nil {
		return err
	}

	b.remember(key, apply)

	return nil
}

// NewChannel opens a channel of its own, so a publisher never shares the
// channel the connection uses for declarations.
func (b *Connection) NewChannel(ctx context.Context) (*amqp091.Channel, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if b.conn == nil || b.conn.IsClosed() {
		return nil, ErrConnectionNotAvailable
	}

	ch, err := b.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	return ch, nil
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
	return b.declare(ctx, "exchange "+cfg.Name, func(ch *amqp091.Channel) error {
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
	})
}

func (b *Connection) BindExchange(
	ctx context.Context,
	destination, source, routingKey string,
	noWait bool,
	args amqp091.Table,
) error {
	key := fmt.Sprintf("exchange bind %s->%s/%s", source, destination, routingKey)

	return b.declare(ctx, key, func(ch *amqp091.Channel) error {
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
	})
}

func (b *Connection) DeclareQueue(
	ctx context.Context,
	cfg *QueueConfig,
	exchange *ExchangeConfig,
	routingKey string,
) error {
	key := fmt.Sprintf("queue %s->%s/%s", cfg.Name, exchangeName(exchange), routingKey)

	return b.declare(ctx, key, func(ch *amqp091.Channel) error {
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
	})
}

func exchangeName(cfg *ExchangeConfig) string {
	if cfg == nil {
		return ""
	}

	return cfg.Name
}

func (b *Connection) BindQueue(queueName string, rk string, exchange string, noWait bool, args amqp091.Table) error {
	key := fmt.Sprintf("queue bind %s->%s/%s", exchange, queueName, rk)

	err := b.declare(context.Background(), key, func(ch *amqp091.Channel) error {
		if err := ch.QueueBind(queueName, rk, exchange, noWait, args); err != nil {
			return fmt.Errorf("%w: %v", ErrDeclarationFailed, err)
		}

		return nil
	})
	if err != nil {
		return err
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
