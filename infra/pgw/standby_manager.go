package pgw

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/webitel-go-kit/pkg/safemap"
)

type standbyManagerConfig struct {
	PickStrategy                  PickStrategy
	UnhealthyStandbyRetryInterval time.Duration

	RetriesBeforeUnhealthy int
	RetryStrategy          RetryStrategy
	RetryStrategyBaseValue int

	HostHealthCheckInterval time.Duration
	HostHealthCheckTimeout  time.Duration

	ErrorParser       pgErrorParser
	MigrationVerifier MigrationVerifier
}

type standbyManager struct {
	config *standbyManagerConfig

	store *safemap.SafeMap[string, *Pool]

	unhealthyStore *safemap.SafeMap[string, *Pool]

	// ctx is canceled in Close so in-flight ConnectWithRetry calls stop.
	ctx    context.Context
	cancel context.CancelFunc

	closeOnce sync.Once
}

func (cfg *standbyManagerConfig) normalizeConfig() {
	if cfg.PickStrategy == nil {
		cfg.PickStrategy = DefaultStandbyPoolConfig.PickStrategy
	}
	if cfg.UnhealthyStandbyRetryInterval == 0 {
		cfg.UnhealthyStandbyRetryInterval = DefaultStandbyPoolConfig.UnhealthyReplicaRetryInterval
	}
	if cfg.RetriesBeforeUnhealthy == 0 {
		cfg.RetriesBeforeUnhealthy = DefaultStandbyPoolConfig.RetriesBeforeUnhealthy
	}
	if cfg.RetryStrategy == nil {
		cfg.RetryStrategy = DefaultStandbyPoolConfig.RetryStrategy
	}
	if cfg.RetryStrategyBaseValue == 0 {
		cfg.RetryStrategyBaseValue = DefaultStandbyPoolConfig.RetryStrategyBaseValue
	}
	if cfg.HostHealthCheckInterval == 0 {
		cfg.HostHealthCheckInterval = DefaultStandbyPoolConfig.HealthCheckInterval
	}
	if cfg.HostHealthCheckTimeout == 0 {
		cfg.HostHealthCheckTimeout = DefaultStandbyPoolConfig.HealthCheckTimeout
	}
}

func newStandbyManager(cfg standbyManagerConfig) (*standbyManager, error) {
	cfg.normalizeConfig()

	ctx, cancel := context.WithCancel(context.Background())
	manager := &standbyManager{
		config: &cfg,

		store:          safemap.New[string, *Pool](nil),
		unhealthyStore: safemap.New[string, *Pool](nil),

		ctx:    ctx,
		cancel: cancel,
	}

	go manager.monitorUnhealthy()

	return manager, nil
}

func (rm *standbyManager) Stats() (int, int) {
	return rm.store.Len(), rm.unhealthyStore.Len()
}

func (rm *standbyManager) Pick() *Pool {
	return rm.config.PickStrategy(rm.store)
}

func (rm *standbyManager) AddStandby(ctx context.Context, pool *pgxpool.Pool) error {
	host, err := newPool(pool, PoolConfig{
		HealthCheckInterval: rm.config.HostHealthCheckInterval,
		HealthCheckTimeout:  rm.config.HostHealthCheckTimeout,
		ErrorParser:         rm.config.ErrorParser,
		MigrationVerifier:   rm.config.MigrationVerifier,
	})
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		host.Close()
		return err
	}

	key := rm.buildMapKey(host)
	// Register before connect so Close can always find and shut the host down.
	rm.unhealthyStore.Set(key, host)
	if rm.isClosed() {
		rm.unhealthyStore.Remove(key)
		host.Close()
		return ErrClosed
	}

	go rm.connectStandby(key, host)
	return nil
}

func (rm *standbyManager) connectStandby(key string, host *Pool) {
	err := host.ConnectWithRetry(
		rm.ctx,
		rm.config.RetriesBeforeUnhealthy,
		rm.config.RetryStrategy,
		rm.config.RetryStrategyBaseValue,
	)
	if err != nil || rm.isClosed() {
		return
	}
	rm.promoteHealthy(key, host)
}

func (rm *standbyManager) monitorStateChange(key string, host *Pool) {
	ctx, cancel := context.WithCancel(rm.ctx)
	defer cancel()
	subscribedChan := host.SubscribeStateChange(ctx)
	for {
		select {
		case state, ok := <-subscribedChan:
			if !ok {
				return
			}
			switch state {
			case HostStateError:
				rm.moveToUnhealthy(key)
				return
			case HostStateClosed:
				rm.store.Remove(key)
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func (rm *standbyManager) moveToUnhealthy(key string) {
	if key == "" {
		return
	}

	host, found := rm.store.Get(key)
	if !found || host == nil {
		return
	}

	rm.store.Remove(key)
	rm.unhealthyStore.Set(key, host)
}

func (rm *standbyManager) monitorUnhealthy() {
	ticker := time.NewTicker(rm.config.UnhealthyStandbyRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.reconnectUnhealthy()
		case <-rm.ctx.Done():
			return
		}
	}
}

func (rm *standbyManager) reconnectUnhealthy() {
	if rm.isClosed() || rm.unhealthyStore.Len() == 0 {
		return
	}

	var wg sync.WaitGroup
	_ = rm.unhealthyStore.Range(func(key string, replica *Pool) error {
		wg.Add(1)
		go func(k string, host *Pool) {
			defer wg.Done()
			rm.retryUnhealthy(k, host)
		}(key, replica)
		return nil
	})
	wg.Wait()
}

func (rm *standbyManager) retryUnhealthy(key string, host *Pool) {
	if host == nil || host.GetState() == HostStateClosed {
		rm.unhealthyStore.Remove(key)
		return
	}

	err := host.ConnectWithRetry(
		rm.ctx,
		rm.config.RetriesBeforeUnhealthy,
		rm.config.RetryStrategy,
		rm.config.RetryStrategyBaseValue,
	)
	if err != nil || rm.isClosed() {
		return
	}
	rm.promoteHealthy(key, host)
}

// promoteHealthy moves a connected host into the healthy store.
// Store-then-recheck mirrors PoolManager.initPrimary so a concurrent Close
// cannot leave an open pool invisible to shutdown.
func (rm *standbyManager) promoteHealthy(key string, host *Pool) {
	current, found := rm.unhealthyStore.Get(key)
	if !found || current != host {
		return
	}
	rm.unhealthyStore.Remove(key)
	rm.store.Set(key, host)
	if rm.isClosed() {
		rm.store.Remove(key)
		host.Close()
		return
	}
	go rm.monitorStateChange(key, host)
}

func (rm *standbyManager) buildMapKey(host *Pool) string {
	return rm.buildMapKeyFromPool(host.pool)
}

func (rm *standbyManager) buildMapKeyFromPool(pool *pgxpool.Pool) string {
	return pool.Config().ConnString()
}

func (rm *standbyManager) Close() {
	rm.closeOnce.Do(rm.shutdown)
}

func (rm *standbyManager) isClosed() bool {
	return rm.ctx.Err() != nil
}

func (rm *standbyManager) shutdown() {
	rm.cancel()

	closeTrackedPools(rm.store)
	closeTrackedPools(rm.unhealthyStore)
}

func closeTrackedPools(store *safemap.SafeMap[string, *Pool]) {
	hosts, err := store.Copy()
	if err != nil {
		return
	}
	for key, host := range hosts {
		if host != nil {
			host.Close()
		}
		store.Remove(key)
	}
}
