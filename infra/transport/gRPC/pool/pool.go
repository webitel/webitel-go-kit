package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
)

type (
	Factory            func() (*grpc.ClientConn, error)
	FactoryWithContext func(context.Context) (*grpc.ClientConn, error)

	Pool struct {
		clients         chan ClientConn
		factory         FactoryWithContext
		idleTimeout     time.Duration
		maxLifeDuration time.Duration
		lock            sync.RWMutex
		closed          atomic.Bool
	}

	ClientConn struct {
		*grpc.ClientConn

		pool          *Pool
		timeUsed      time.Time
		timeInitiated time.Time
		unhealthy     bool
	}
)

func New(factory Factory, init, capacity int, idleTimeout time.Duration, maxLifeDuration ...time.Duration) (*Pool, error) {
	return NewWithContext(context.Background(), func(ctx context.Context) (*grpc.ClientConn, error) { return factory() }, init, capacity, idleTimeout, maxLifeDuration...)
}

func NewWithContext(ctx context.Context, factory FactoryWithContext, init, capacity int, idleTimeout time.Duration, maxLifeDuration ...time.Duration) (*Pool, error) {
	if capacity <= 0 {
		capacity = 1
	}

	if init < 0 {
		init = 0
	}

	if init > capacity {
		init = capacity
	}

	p := &Pool{
		clients:     make(chan ClientConn, capacity),
		factory:     factory,
		idleTimeout: idleTimeout,
	}

	if len(maxLifeDuration) > 0 {
		p.maxLifeDuration = maxLifeDuration[0]
	}

	for range init {
		c, err := factory(ctx)
		if err != nil {
			return nil, err
		}

		p.clients <- ClientConn{
			ClientConn:    c,
			pool:          p,
			timeUsed:      time.Now(),
			timeInitiated: time.Now(),
		}
	}

	for range capacity - init {
		p.clients <- ClientConn{
			pool: p,
		}
	}

	return p, nil
}

func (p *Pool) getClients() chan ClientConn {
	p.lock.RLock()         // +[R] lock
	defer p.lock.RUnlock() // -[R] lock

	return p.clients
}

func (p *Pool) Close() {
	p.lock.Lock() // +[RW] lock

	clients := p.clients
	p.clients = nil

	p.lock.Unlock() // -[RW] lock

	if clients == nil {
		return
	}

	if p.closed.Swap(true) {
		return
	}

	close(clients)

	for client := range clients {
		if client.ClientConn == nil {
			continue
		}

		client.ClientConn.Close()
	}
}

func (p *Pool) IsClosed() bool {
	return p == nil || p.getClients() == nil
}

func (p *Pool) Get(ctx context.Context) (*ClientConn, error) {
	clients := p.getClients()
	if clients == nil {
		return nil, errors.New("[gRPC POOL] client pool is closed")
	}

	wrapper := ClientConn{
		pool: p,
	}

	select {
	case wrapper = <-clients:
		//All good!
	case <-ctx.Done():
		return nil, errors.New("[gRPC POOL] context deadline exceeded")
	}

	idleTimeout := p.idleTimeout
	if wrapper.ClientConn != nil && idleTimeout > 0 && wrapper.timeUsed.Add(idleTimeout).Before(time.Now()) {
		wrapper.ClientConn.Close()
		wrapper.ClientConn = nil
	}

	if wrapper.ClientConn == nil {
		conn, err := p.factory(ctx)
		if err != nil {
			wrapper.pool = p
			clients <- wrapper
			return nil, err
		}
		wrapper.ClientConn = conn
		wrapper.timeInitiated = time.Now()
	}

	wrapper.pool = p
	return &wrapper, nil
}

func (c *ClientConn) Unhealthy() {
	c.unhealthy = true
}

func (c *ClientConn) Close() error {
	if c == nil {
		return nil
	}

	if c.ClientConn == nil {
		return errors.New("[gRPC pool] already closed")
	}

	maxDuration := c.pool.maxLifeDuration
	if maxDuration > 0 && c.timeInitiated.Add(maxDuration).Before(time.Now()) {
		c.Unhealthy()
	}

	wrapper := ClientConn{
		pool:       c.pool,
		ClientConn: c.ClientConn,
		timeUsed:   time.Now(),
	}

	if c.unhealthy {
		wrapper.ClientConn.Close()
		wrapper.ClientConn = nil
	} else {
		wrapper.timeInitiated = c.timeInitiated
	}

	if c.pool.closed.Load() {
		c.ClientConn.Close()
		return nil
	}

	select {
	case c.pool.clients <- wrapper:
	default:
		c.ClientConn.Close()
	}

	c.ClientConn = nil

	return nil
}

func (p *Pool) Capacity() int {
	if p.IsClosed() {
		return 0
	}

	return cap(p.clients)
}

func (p *Pool) Available() int {
	if p.IsClosed() {
		return 0
	}

	return len(p.clients)
}
