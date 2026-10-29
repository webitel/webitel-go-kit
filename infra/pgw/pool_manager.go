package pgw

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	postgresApplicationNameKey = "application_name"
)

var (
	ErrUnreachable = errors.New("the resource is unreachable now")
	ErrClosed      = errors.New("pgw: pool manager is closed")
	ErrStarted     = errors.New("pgw: pool manager is already started")
)

type PoolManager struct {
	config *Config

	master atomic.Pointer[Pool]

	standbyManager *standbyManager

	errorsManager *errorsManager

	closeChan chan struct{}
	closeOnce sync.Once
	started   atomic.Bool
}

// Primary returns the primary pool.
// It returns an error if the primary pool is not connected.
func (c *PoolManager) Primary() (*Pool, error) {
	master := c.master.Load()
	if master == nil || master.GetState() != HostStateConnected {
		return nil, ErrUnreachable
	}

	return master, nil
}

// Standby returns the standby pool.
// It returns an error if the standby pool is not connected.
func (c *PoolManager) Standby() (*Pool, error) {
	host := c.standbyManager.Pick()
	if host == nil {
		return nil, ErrUnreachable
	}

	return host, nil
}

// StandbyPreferred returns the standby pool if it is connected, otherwise it returns the primary pool.
// It returns an error if the primary pool is not connected.
func (c *PoolManager) StandbyPreferred() (*Pool, error) {
	host := c.standbyManager.Pick()
	if host == nil {
		return c.Primary()
	}

	return host, nil
}

// RegisterUniqueViolation registers a unique violation error processor for the given constraint name.
func (e *PoolManager) RegisterUniqueViolation(constraintName string, processor ErrorProcessor) error {
	return e.errorsManager.RegisterUniqueViolation(constraintName, processor)
}

// RegisterForeignKeyViolation registers a foreign key violation error processor for the given constraint name.
func (e *PoolManager) RegisterForeignKeyViolation(constraintName string, processor ErrorProcessor) error {
	return e.errorsManager.RegisterForeignKeyViolation(constraintName, processor)
}

// RegisterCheckViolation registers a check violation error processor for the given constraint name.
func (e *PoolManager) RegisterCheckViolation(constraintName string, processor ErrorProcessor) error {
	return e.errorsManager.RegisterCheckViolation(constraintName, processor)
}

// RegisterNotNullViolation registers a not null violation error processor for the given table and column.
func (e *PoolManager) RegisterNotNullViolation(table string, column string, processor ErrorProcessor) error {
	return e.errorsManager.RegisterNotNullViolationProcessor(table, column, processor)
}

// Close closes the pool manager and all its pools. It is safe to call more than once.
func (c *PoolManager) Close() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
		if master := c.master.Swap(nil); master != nil {
			master.Close()
		}
		if c.standbyManager != nil {
			c.standbyManager.Close()
		}
	})
}

// New builds a PoolManager without connecting.
func New(options ...ConfigOption) (*PoolManager, error) {
	cfg := &Config{
		PrimaryConfig: DefaultPrimaryPoolConfig,
		StandbyConfig: DefaultStandbyPoolConfig,
	}

	for _, opt := range options {
		opt(cfg)
	}

	if cfg.PrimaryConfig.DSN == "" {
		return nil, errors.New("primary config DSN is required")
	}

	cfg.PrimaryConfig = mergePrimaryPoolConfigWithDefault(cfg.PrimaryConfig)
	cfg.StandbyConfig = mergeStandbyPoolConfigWithDefault(cfg.StandbyConfig)

	errorsMgr := newErrorsManager()
	standby, err := newStandbyManager(standbyManagerConfig{
		PickStrategy:                  cfg.StandbyConfig.PickStrategy,
		UnhealthyStandbyRetryInterval: cfg.StandbyConfig.UnhealthyReplicaRetryInterval,
		RetriesBeforeUnhealthy:        cfg.StandbyConfig.RetriesBeforeUnhealthy,
		RetryStrategy:                 cfg.StandbyConfig.RetryStrategy,
		RetryStrategyBaseValue:        cfg.StandbyConfig.RetryStrategyBaseValue,
		HostHealthCheckInterval:       cfg.StandbyConfig.HealthCheckInterval,
		HostHealthCheckTimeout:        cfg.StandbyConfig.HealthCheckTimeout,
		ErrorParser:                   errorsMgr,
		MigrationVerifier:             cfg.MigrationVerifier,
	})
	if err != nil {
		return nil, err
	}

	return &PoolManager{
		config:         cfg,
		closeChan:      make(chan struct{}),
		errorsManager:  errorsMgr,
		standbyManager: standby,
	}, nil
}

// NewPoolManager is New followed by Start.
func NewPoolManager(ctx context.Context, options ...ConfigOption) (*PoolManager, error) {
	m, err := New(options...)
	if err != nil {
		return nil, err
	}
	if err := m.Start(ctx); err != nil {
		m.Close()
		return nil, err
	}

	return m, nil
}

// Start dials the primary and standbys.
// Returns ErrStarted on a second call, ErrClosed after Close.
func (c *PoolManager) Start(ctx context.Context) error {
	if c.isClosed() {
		return ErrClosed
	}
	if !c.started.CompareAndSwap(false, true) {
		return ErrStarted
	}

	if err := c.initPrimary(ctx, c.config); err != nil {
		return err
	}

	if err := c.addStandbys(ctx, c.config); err != nil {
		if master := c.master.Swap(nil); master != nil {
			master.Close()
		}
		return err
	}

	return nil
}

func mergePrimaryPoolConfigWithDefault(primaryConfig PrimaryConfig) PrimaryConfig {
	mergedPoolConfig := DefaultPrimaryPoolConfig
	if primaryConfig.DSN != "" {
		mergedPoolConfig.DSN = primaryConfig.DSN
	}
	if primaryConfig.HealthCheckInterval > 0 {
		mergedPoolConfig.HealthCheckInterval = primaryConfig.HealthCheckInterval
	}
	if primaryConfig.HealthCheckTimeout > 0 {
		mergedPoolConfig.HealthCheckTimeout = primaryConfig.HealthCheckTimeout
	}
	if primaryConfig.RetryAttempts > 0 {
		mergedPoolConfig.RetryAttempts = primaryConfig.RetryAttempts
	}
	if primaryConfig.RetryInterval > 0 {
		mergedPoolConfig.RetryInterval = primaryConfig.RetryInterval
	}
	if primaryConfig.RetryStrategy != nil {
		mergedPoolConfig.RetryStrategy = primaryConfig.RetryStrategy
	}
	if primaryConfig.RetryStrategyBaseValue > 0 {
		mergedPoolConfig.RetryStrategyBaseValue = primaryConfig.RetryStrategyBaseValue
	}
	if primaryConfig.MaxConns > 0 {
		mergedPoolConfig.MaxConns = primaryConfig.MaxConns
	}
	if primaryConfig.MinConns > 0 {
		mergedPoolConfig.MinConns = primaryConfig.MinConns
	}
	return mergedPoolConfig
}

func mergeStandbyPoolConfigWithDefault(config StandbyConfig) StandbyConfig {
	mergedPoolConfig := DefaultStandbyPoolConfig
	if len(config.DSN) > 0 {
		mergedPoolConfig.DSN = config.DSN
	}
	if config.HealthCheckInterval > 0 {
		mergedPoolConfig.HealthCheckInterval = config.HealthCheckInterval
	}
	if config.HealthCheckTimeout > 0 {
		mergedPoolConfig.HealthCheckTimeout = config.HealthCheckTimeout
	}
	if config.RetriesBeforeUnhealthy > 0 {
		mergedPoolConfig.RetriesBeforeUnhealthy = config.RetriesBeforeUnhealthy
	}
	if config.RetryStrategy != nil {
		mergedPoolConfig.RetryStrategy = config.RetryStrategy
	}
	if config.RetryStrategyBaseValue > 0 {
		mergedPoolConfig.RetryStrategyBaseValue = config.RetryStrategyBaseValue
	}
	if config.MaxConns > 0 {
		mergedPoolConfig.MaxConns = config.MaxConns
	}
	if config.MinConns > 0 {
		mergedPoolConfig.MinConns = config.MinConns
	}
	return mergedPoolConfig
}

func (c *PoolManager) initPrimary(ctx context.Context, config *Config) error {
	if config == nil {
		return errors.New("primary config is required to run pgw")
	}

	primaryConfig := config.PrimaryConfig

	pool, err := c.buildPrimaryPgxPool(ctx, config)
	if err != nil {
		return err
	}

	defaultPool, err := newPool(pool, PoolConfig{
		HealthCheckInterval: primaryConfig.HealthCheckInterval,
		HealthCheckTimeout:  primaryConfig.HealthCheckTimeout,
		ErrorParser:         c.errorsManager,
		MigrationVerifier:   config.MigrationVerifier,
	})
	if err != nil {
		pool.Close()
		return err
	}

	err = defaultPool.ConnectWithRetry(
		ctx,
		primaryConfig.RetryAttempts,
		primaryConfig.RetryStrategy,
		primaryConfig.RetryStrategyBaseValue)

	if err != nil {
		defaultPool.Close()
		return err
	}

	c.master.Store(defaultPool)
	if c.isClosed() {
		if master := c.master.Swap(nil); master != nil {
			master.Close()
		}
		return ErrClosed
	}
	go c.startMasterMonitor()

	return nil
}

func (c *PoolManager) isClosed() bool {
	select {
	case <-c.closeChan:
		return true
	default:
		return false
	}
}

func (c *PoolManager) buildPrimaryPgxPool(ctx context.Context, config *Config) (*pgxpool.Pool, error) {
	primaryConfig := config.PrimaryConfig

	poolConfig, err := pgxpool.ParseConfig(primaryConfig.DSN)
	if err != nil {
		return nil, err
	}

	c.setPoolApplicationName(poolConfig, config.ApplicationName)
	if config.Tracer != nil {
		poolConfig.ConnConfig.Tracer = &pgxTracerAdapter{t: config.Tracer}
	}
	if primaryConfig.MaxConns > 0 {
		poolConfig.MaxConns = int32(primaryConfig.MaxConns)
	}
	if primaryConfig.MinConns > 0 {
		poolConfig.MinConns = int32(primaryConfig.MinConns)
	}
	poolConfig.BeforeConnect = func(ctx context.Context, cfg *pgx.ConnConfig) error {
		cfg.ValidateConnect = pgconn.ValidateConnectTargetSessionAttrsPrimary
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func (c *PoolManager) startMasterMonitor() {
	go func() {
		master := c.master.Load()
		if master == nil {
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		ch := master.SubscribeStateChange(ctx)
		defer cancel()
		for {
			select {
			case state, ok := <-ch:
				if !ok {
					return
				}
				if state == HostStateError {
					c.reconnectMasterLoop()
					return
				}
			case <-c.closeChan:
				return
			}
		}
	}()
}

func (c *PoolManager) reconnectMasterLoop() {
	masterCfg := c.config.PrimaryConfig
	for {
		select {
		case <-c.closeChan:
			return
		case <-time.After(masterCfg.RetryInterval):
			master := c.master.Load()
			if master == nil {
				return
			}
			err := master.ConnectWithRetry(
				context.Background(),
				masterCfg.RetryAttempts,
				masterCfg.RetryStrategy,
				masterCfg.RetryStrategyBaseValue)
			if err == nil {
				go c.startMasterMonitor()
				return
			}
		}
	}
}

func (c *PoolManager) addStandbys(ctx context.Context, config *Config) error {
	replicaConfig := config.StandbyConfig
	if len(replicaConfig.DSN) == 0 {
		return nil
	}

	for _, dsn := range replicaConfig.DSN {
		poolConfig, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			return err
		}

		c.setPoolApplicationName(poolConfig, config.ApplicationName)
		if config.Tracer != nil {
			poolConfig.ConnConfig.Tracer = &pgxTracerAdapter{t: config.Tracer}
		}
		if replicaConfig.MaxConns > 0 {
			poolConfig.MaxConns = int32(replicaConfig.MaxConns)
		}
		if replicaConfig.MinConns > 0 {
			poolConfig.MinConns = int32(replicaConfig.MinConns)
		}
		poolConfig.BeforeConnect = func(ctx context.Context, cfg *pgx.ConnConfig) error {
			cfg.ValidateConnect = pgconn.ValidateConnectTargetSessionAttrsReadOnly
			return nil
		}
		poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, "SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY")
			return err
		}

		pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			return err
		}

		if err := c.standbyManager.AddStandby(ctx, pool); err != nil {
			return fmt.Errorf("add standby: %w", err)
		}
	}
	return nil
}

func (c *PoolManager) setPoolApplicationName(poolConfig *pgxpool.Config, applicationName string) {
	if len(applicationName) > 0 {
		poolConfig.ConnConfig.RuntimeParams[postgresApplicationNameKey] = applicationName
	}
}
