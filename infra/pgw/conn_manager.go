package pgw

import (
	"context"
	"errors"
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
)

type PoolManager struct {
	config *Config

	master *Pool

	standbyManager *standbyManager

	errorsManager *errorsManager

	closeChan chan struct{}
}

func (c *PoolManager) Primary() (*Pool, error) {
	if c.master == nil || c.master.GetState() != HostStateConnected {
		return nil, ErrUnreachable
	}

	return c.master, nil
}

func (c *PoolManager) Standby() (*Pool, error) {
	host := c.standbyManager.Pick()
	if host == nil {
		return nil, ErrUnreachable
	}

	return host, nil
}

func (c *PoolManager) StandbyPreferred() (*Pool, error) {
	host := c.standbyManager.Pick()
	if host == nil {
		return c.Primary()
	}

	return host, nil
}

func (e *PoolManager) RegisterUniqueViolation(constraintName string, processor ErrorProcessor) error {
	return e.errorsManager.RegisterUniqueViolation(constraintName, processor)
}

func (e *PoolManager) RegisterForeignKeyViolation(constraintName string, processor ErrorProcessor) error {
	return e.errorsManager.RegisterForeignKeyViolation(constraintName, processor)
}

func (e *PoolManager) RegisterCheckViolation(constraintName string, processor ErrorProcessor) error {
	return e.errorsManager.RegisterCheckViolation(constraintName, processor)
}

func (e *PoolManager) RegisterNotNullViolation(table string, column string, processor ErrorProcessor) error {
	return e.errorsManager.RegisterNotNullViolationProcessor(table, column, processor)
}

func (c *PoolManager) Close() {
	close(c.closeChan)
	if c.master != nil {
		c.master.Close()
	}
	c.standbyManager.Close()
}

func NewConnectionManager(ctx context.Context, options ...ConfigOption) (*PoolManager, error) {
	cfg := &Config{
		PrimaryConfig: DefaultMasterPoolConfig,
		StandbyConfig: DefaultReplicaPoolConfig,
	}

	for _, opt := range options {
		opt(cfg)
	}

	conn := &PoolManager{
		config:        cfg,
		closeChan:     make(chan struct{}),
		errorsManager: newErrorsManager(),
	}

	err := conn.initMaster(ctx, cfg)
	if err != nil {
		return nil, err
	}

	err = conn.initReplicas(ctx, cfg)
	if err != nil {
		conn.master.Close()
		return nil, err
	}

	return conn, nil
}

func (c *PoolManager) initMaster(ctx context.Context, config *Config) error {
	initPoolConfig := config.PrimaryConfig
	poolConfig, err := pgxpool.ParseConfig(initPoolConfig.DSN)
	if err != nil {
		return err
	}

	c.setPoolApplicationName(poolConfig, config.ApplicationName)
	if config.Tracer != nil {
		poolConfig.ConnConfig.Tracer = &pgxTracerAdapter{t: config.Tracer}
	}
	if initPoolConfig.MaxConns > 0 {
		poolConfig.MaxConns = int32(initPoolConfig.MaxConns)
	}
	if initPoolConfig.MinConns > 0 {
		poolConfig.MinConns = int32(initPoolConfig.MinConns)
	}
	poolConfig.BeforeConnect = func(ctx context.Context, cfg *pgx.ConnConfig) error {
		cfg.ValidateConnect = pgconn.ValidateConnectTargetSessionAttrsPrimary
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return err
	}

	defaultPool, err := newPool(pool, PoolConfig{
		HealthCheckInterval: initPoolConfig.HealthCheckInterval,
		HealthCheckTimeout:  initPoolConfig.HealthCheckTimeout,
		ErrorParser:         c.errorsManager,
		MigrationVerifier:   config.MigrationVerifier,
	})
	if err != nil {
		pool.Close()
		return err
	}

	err = defaultPool.ConnectWithRetry(
		initPoolConfig.RetryAttempts,
		initPoolConfig.RetryStrategy,
		initPoolConfig.RetryStrategyBaseValue)

	if err != nil {
		defaultPool.Close()
		return err
	}

	c.master = defaultPool
	go c.startMasterMonitor()

	return nil
}

func (c *PoolManager) startMasterMonitor() {
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := c.master.SubscribeStateChange(ctx)
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
			err := c.master.ConnectWithRetry(
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

func (c *PoolManager) initReplicas(ctx context.Context, config *Config) error {
	var (
		replicaConfig = config.StandbyConfig
	)
	manager, err := newStandbyManager(standbyManagerConfig{
		PickStrategy:                  replicaConfig.PickStrategy,
		UnhealthyStandbyRetryInterval: replicaConfig.UnhealthyReplicaRetryInterval,
		RetriesBeforeUnhealthy:        replicaConfig.RetriesBeforeUnhealthy,
		RetryStrategy:                 replicaConfig.RetryStrategy,
		RetryStrategyBaseValue:        replicaConfig.RetryStrategyBaseValue,
		HostHealthCheckInterval:       replicaConfig.HealthCheckInterval,
		HostHealthCheckTimeout:        replicaConfig.HealthCheckTimeout,
		ErrorParser:                   c.errorsManager,
		MigrationVerifier:             config.MigrationVerifier,
	})
	if err != nil {
		return err
	}
	c.standbyManager = manager

	if len(config.StandbyConfig.DSN) == 0 {
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

		err = c.standbyManager.AddStandby(ctx, pool)
		if err != nil {
			return err
		}
	}
	return nil

}

func (c *PoolManager) setPoolApplicationName(poolConfig *pgxpool.Config, applicationName string) {
	if len(applicationName) > 0 {
		poolConfig.ConnConfig.RuntimeParams[postgresApplicationNameKey] = applicationName
	}
}
