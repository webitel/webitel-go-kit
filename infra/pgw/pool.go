package pgw

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMigrationVerify = errors.New("pgw: migration unverified")
)

type PoolConfig struct {
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration

	ErrorParser       pgErrorParser
	MigrationVerifier MigrationVerifier
}

type pgErrorParser interface {
	ParsePgError(error) error
}

type Pool struct {
	pool  *pgxpool.Pool
	state poolState

	reconnectMu sync.Mutex

	closeOnce sync.Once
	closeChan chan struct{}

	config *PoolConfig
}

func newPool(pool *pgxpool.Pool, cfg PoolConfig) (*Pool, error) {
	if pool == nil {
		return nil, errors.New("pgw: pool must not be nil")
	}
	if cfg.ErrorParser == nil {
		return nil, errors.New("pgw: ErrorParser must not be nil")
	}
	if cfg.HealthCheckInterval == 0 {
		cfg.HealthCheckInterval = DefaultPrimaryPoolConfig.HealthCheckInterval
	}
	if cfg.HealthCheckTimeout == 0 {
		cfg.HealthCheckTimeout = DefaultPrimaryPoolConfig.HealthCheckTimeout
	}

	mode := pool.Config().ConnConfig.RuntimeParams["target_session_attrs"]

	node := &Pool{
		pool: pool,
		state: poolState{
			state:     HostStateConnecting,
			mode:      HostMode(mode),
			closeChan: make(chan struct{}),
		},
		closeChan: make(chan struct{}),
		config:    &cfg,
	}

	return node, nil
}

func (h *Pool) GetState() PoolState {
	return h.state.Get()
}

func (h *Pool) Connect(ctx context.Context) error {
	err := h.validateHealth(ctx)
	if err != nil {
		return err
	}

	err = h.verifyMigration(ctx)
	if err != nil {
		return err
	}

	go h.healthCheck()

	return nil
}

func (h *Pool) verifyMigration(ctx context.Context) error {
	if h.config.MigrationVerifier == nil {
		return nil
	}
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		h.state.Set(HostStateError)
		return err
	}
	defer conn.Release()
	err = h.config.MigrationVerifier(ctx, conn)
	if err != nil {
		h.state.Set(HostStateError)
		return errors.Join(ErrMigrationVerify, err)
	}
	return nil
}

func (h *Pool) ConnectWithRetry(maxAttempts int, strategy RetryStrategy, baseValue int) error {
	h.reconnectMu.Lock()
	defer h.reconnectMu.Unlock()

	var lastErr error
	for i := 1; i <= maxAttempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), h.config.HealthCheckTimeout)
		err := h.Connect(ctx)
		cancel()
		if err == nil {
			return nil
		}

		lastErr = err
		if i < maxAttempts {
			timeToWait := strategy(baseValue, i)
			select {
			case <-time.After(timeToWait):
			case <-h.closeChan:
				return errors.New("pool closed while waiting for reconnect")
			}
		}
	}
	return lastErr
}

func (h *Pool) validateHealth(ctx context.Context) error {
	err := h.pool.Ping(ctx)
	if err != nil {
		if h.state.Get() != HostStateError {
			h.state.Set(HostStateError)
		}
		return err
	}
	if h.state.Get() != HostStateConnected {
		h.state.Set(HostStateConnected)
	}
	return nil
}

func (h *Pool) Stat() *pgxpool.Stat {
	return h.pool.Stat()
}

func (h *Pool) SubscribeStateChange(ctx context.Context) <-chan PoolState {
	return h.state.SubscribeChange(ctx)
}

func (h *Pool) healthCheck() {
	ticker := time.NewTicker(h.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), h.config.HealthCheckTimeout)
			err := h.validateHealth(ctx)
			cancel()
			if err != nil {
				return
			}
		case <-h.closeChan:
			return
		}
	}
}

func (h *Pool) Close() {
	h.closeOnce.Do(func() {
		close(h.closeChan)

		h.pool.Close()

		h.state.Close()

	})
}

func (h *Pool) parseErr(err error) error {
	if err == nil {
		return nil
	}
	return h.config.ErrorParser.ParsePgError(err)
}

func (h *Pool) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	tag, err := h.pool.Exec(ctx, query, args...)
	return tag, h.parseErr(err)
}

func (h *Pool) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := h.pool.Query(ctx, query, args...)
	return rows, h.parseErr(err)
}

func (h *Pool) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return h.pool.QueryRow(ctx, query, args...)
}

func (h *Pool) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	tx, err := h.pool.BeginTx(ctx, opts)
	return tx, h.parseErr(err)
}

func (h *Pool) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := h.pool.Begin(ctx)
	return tx, h.parseErr(err)
}

func (h *Pool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	conn, err := h.pool.Acquire(ctx)
	return conn, h.parseErr(err)
}

func (h *Pool) AcquireAllIdle(ctx context.Context) []*pgxpool.Conn {
	return h.pool.AcquireAllIdle(ctx)
}

func (h *Pool) AcquireFunc(ctx context.Context, f func(*pgxpool.Conn) error) error {
	return h.pool.AcquireFunc(ctx, f)
}

func (h *Pool) CopyFrom(ctx context.Context, tableName pgx.Identifier, columns []string, rows pgx.Rows) (int64, error) {
	n, err := h.pool.CopyFrom(ctx, tableName, columns, rows)
	return n, h.parseErr(err)
}

func (h *Pool) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return h.pool.SendBatch(ctx, b)
}
