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

	closeChan chan struct{}
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

	manager := &standbyManager{
		config: &cfg,

		store:          safemap.New[string, *Pool](nil),
		unhealthyStore: safemap.New[string, *Pool](nil),

		closeChan: make(chan struct{}),
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

	go func() {
		key := rm.buildMapKeyFromPool(pool)

		err := host.ConnectWithRetry(
			rm.config.RetriesBeforeUnhealthy,
			rm.config.RetryStrategy,
			rm.config.RetryStrategyBaseValue)

		if err != nil {
			rm.unhealthyStore.Set(key, host)
			return
		}

		rm.store.Set(key, host)

		go rm.monitorStateChange(key, host)
	}()

	return nil

}

func (rm *standbyManager) monitorStateChange(key string, host *Pool) {
	ctx, cancel := context.WithCancel(context.Background())
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

		case <-rm.closeChan:
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
			if rm.unhealthyStore.Len() == 0 {
				continue
			}

			var (
				wg                = sync.WaitGroup{}
				needMoveToHealthy = safemap.New[string, *Pool](nil)
			)

			rm.unhealthyStore.Range(func(key string, replica *Pool) error {
				wg.Add(1)
				go func(host *Pool) {
					defer wg.Done()
					if host.GetState() == HostStateClosed {
						rm.unhealthyStore.Remove(key)
						return
					}
					err := host.ConnectWithRetry(
						rm.config.RetriesBeforeUnhealthy,
						rm.config.RetryStrategy,
						rm.config.RetryStrategyBaseValue)

					if err != nil {
						// TODO: log error
						return
					}
					needMoveToHealthy.Set(rm.buildMapKey(host), host)
				}(replica)

				return nil
			})
			wg.Wait()

			needMoveToHealthy.Range(func(key string, host *Pool) error {
				go func(k string, h *Pool) {
					go rm.monitorStateChange(k, host)

					rm.unhealthyStore.Remove(k)
					rm.store.Set(k, h)
				}(key, host)

				return nil
			})

		case <-rm.closeChan:
			return
		}

	}
}

func (rm *standbyManager) buildMapKey(host *Pool) string {
	return rm.buildMapKeyFromPool(host.pool)
}

func (rm *standbyManager) buildMapKeyFromPool(pool *pgxpool.Pool) string {
	return pool.Config().ConnString()
}

func (rm *standbyManager) Close() {
	close(rm.closeChan)

	_ = rm.store.Range(func(s string, h *Pool) error {
		h.Close()
		return nil
	})
	_ = rm.unhealthyStore.Range(func(s string, h *Pool) error {
		h.Close()
		return nil
	})

	// TODO: track and  log error
}
