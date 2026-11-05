package pgw

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAddStandby_RegistersBeforeConnect(t *testing.T) {
	inRetry := make(chan struct{})
	release := make(chan struct{})
	var signalOnce sync.Once
	var releaseOnce sync.Once
	releaseRetry := func() {
		releaseOnce.Do(func() { close(release) })
	}
	t.Cleanup(releaseRetry)

	rm := newTestStandbyManager(t, standbyManagerConfig{
		UnhealthyStandbyRetryInterval: time.Hour,
		RetriesBeforeUnhealthy:        2,
		RetryStrategy: func(int, int) time.Duration {
			signalOnce.Do(func() { close(inRetry) })
			<-release
			return time.Hour
		},
		RetryStrategyBaseValue:  1,
		HostHealthCheckInterval: time.Hour,
		HostHealthCheckTimeout:  200 * time.Millisecond,
		ErrorParser:             newErrorsManager(),
	})

	if err := rm.AddStandby(context.Background(), newTestPgxPool(t)); err != nil {
		t.Fatalf("AddStandby() = %v", err)
	}

	select {
	case <-inRetry:
	case <-time.After(3 * time.Second):
		t.Fatal("connect retry did not start")
	}

	host := onlyUnhealthyHost(t, rm)
	if host.GetState() == HostStateClosed {
		t.Fatal("host closed while still owned by the manager")
	}
	healthy, unhealthy := rm.Stats()
	if healthy != 0 || unhealthy != 1 {
		t.Fatalf("Stats() during retry = (%d, %d), want (0, 1)", healthy, unhealthy)
	}

	rm.Close()
	releaseRetry()

	waitFor(t, 3*time.Second, func() bool {
		healthy, unhealthy = rm.Stats()
		return healthy == 0 && unhealthy == 0 && host.GetState() == HostStateClosed
	})
}

func TestAddStandby_FailedConnectStaysUnhealthy(t *testing.T) {
	rm := newTestStandbyManager(t, standbyManagerConfig{
		UnhealthyStandbyRetryInterval: time.Hour,
		RetriesBeforeUnhealthy:        1,
		RetryStrategy:                 func(int, int) time.Duration { return time.Millisecond },
		RetryStrategyBaseValue:        1,
		HostHealthCheckInterval:       time.Hour,
		HostHealthCheckTimeout:        200 * time.Millisecond,
		ErrorParser:                   newErrorsManager(),
	})

	if err := rm.AddStandby(context.Background(), newTestPgxPool(t)); err != nil {
		t.Fatalf("AddStandby() = %v", err)
	}

	var host *Pool
	waitFor(t, 3*time.Second, func() bool {
		healthy, unhealthy := rm.Stats()
		if healthy != 0 || unhealthy != 1 {
			return false
		}
		host = onlyUnhealthyHost(t, rm)
		return true
	})
	if host.GetState() == HostStateClosed {
		t.Fatal("host closed while tracked as unhealthy")
	}

	rm.Close()
	if host.GetState() != HostStateClosed {
		t.Fatalf("state = %s, want %s", host.GetState(), HostStateClosed)
	}
	healthy, unhealthy := rm.Stats()
	if healthy != 0 || unhealthy != 0 {
		t.Fatalf("Stats() after Close = (%d, %d), want (0, 0)", healthy, unhealthy)
	}
}

func TestAddStandby_AfterClose(t *testing.T) {
	rm := newTestStandbyManager(t, fastStandbyConfig())
	rm.Close()

	// Host never enters a store on this path, so assert via the underlying pgx pool.
	pool := newTestPgxPool(t)
	err := rm.AddStandby(context.Background(), pool)
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("AddStandby() after Close = %v, want ErrClosed", err)
	}
	if !pgxPoolClosed(pool) {
		t.Fatal("pool left open after AddStandby on closed manager")
	}
	healthy, unhealthy := rm.Stats()
	if healthy != 0 || unhealthy != 0 {
		t.Fatalf("Stats() = (%d, %d), want (0, 0)", healthy, unhealthy)
	}
}

func TestAddStandby_CanceledCallerContext(t *testing.T) {
	rm := newTestStandbyManager(t, fastStandbyConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Host never enters a store on this path, so assert via the underlying pgx pool.
	pool := newTestPgxPool(t)
	err := rm.AddStandby(ctx, pool)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("AddStandby() = %v, want context.Canceled", err)
	}
	if !pgxPoolClosed(pool) {
		t.Fatal("pool left open after canceled AddStandby")
	}
	healthy, unhealthy := rm.Stats()
	if healthy != 0 || unhealthy != 0 {
		t.Fatalf("Stats() = (%d, %d), want (0, 0)", healthy, unhealthy)
	}
}

func TestAddStandby_NilPool(t *testing.T) {
	rm := newTestStandbyManager(t, fastStandbyConfig())

	err := rm.AddStandby(context.Background(), nil)
	if err == nil {
		t.Fatal("AddStandby(nil) = nil, want error")
	}
}

func TestClose_StandbyManagerIdempotent(t *testing.T) {
	rm := newTestStandbyManager(t, fastStandbyConfig())
	rm.Close()
	rm.Close()
}

func TestPromoteHealthy_RacesClose(t *testing.T) {
	for range 50 {
		rm := newTestStandbyManager(t, fastStandbyConfig())
		host := newTestHost(t)
		key := rm.buildMapKey(host)
		rm.unhealthyStore.Set(key, host)

		started := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-started
			rm.Close()
		}()
		go func() {
			defer wg.Done()
			<-started
			rm.promoteHealthy(key, host)
		}()
		close(started)
		wg.Wait()

		if host.GetState() != HostStateClosed {
			t.Fatal("host left open after Close raced with promoteHealthy")
		}
		healthy, unhealthy := rm.Stats()
		if healthy != 0 || unhealthy != 0 {
			t.Fatalf("Stats() = (%d, %d), want (0, 0)", healthy, unhealthy)
		}
	}
}

func TestRetryUnhealthy_CloseDuringRetryDoesNotPromote(t *testing.T) {
	inRetry := make(chan struct{})
	release := make(chan struct{})
	var signalOnce sync.Once
	var releaseOnce sync.Once
	releaseRetry := func() {
		releaseOnce.Do(func() { close(release) })
	}
	t.Cleanup(releaseRetry)

	rm := newTestStandbyManager(t, standbyManagerConfig{
		UnhealthyStandbyRetryInterval: 10 * time.Millisecond,
		RetriesBeforeUnhealthy:        2,
		RetryStrategy: func(int, int) time.Duration {
			signalOnce.Do(func() { close(inRetry) })
			<-release
			return time.Hour
		},
		RetryStrategyBaseValue:  1,
		HostHealthCheckInterval: time.Hour,
		HostHealthCheckTimeout:  200 * time.Millisecond,
		ErrorParser:             newErrorsManager(),
	})

	host := newTestHost(t)
	rm.unhealthyStore.Set(rm.buildMapKey(host), host)

	select {
	case <-inRetry:
	case <-time.After(3 * time.Second):
		t.Fatal("unhealthy retry did not start")
	}

	rm.Close()
	releaseRetry()

	waitFor(t, 3*time.Second, func() bool {
		healthy, unhealthy := rm.Stats()
		return healthy == 0 && unhealthy == 0 && host.GetState() == HostStateClosed
	})
}

func newTestStandbyManager(t *testing.T, cfg standbyManagerConfig) *standbyManager {
	t.Helper()
	rm, err := newStandbyManager(cfg)
	if err != nil {
		t.Fatalf("newStandbyManager() = %v", err)
	}
	t.Cleanup(rm.Close)
	return rm
}

func fastStandbyConfig() standbyManagerConfig {
	return standbyManagerConfig{
		UnhealthyStandbyRetryInterval: time.Hour,
		RetriesBeforeUnhealthy:        1,
		RetryStrategy:                 func(int, int) time.Duration { return time.Millisecond },
		RetryStrategyBaseValue:        1,
		HostHealthCheckInterval:       time.Hour,
		HostHealthCheckTimeout:        200 * time.Millisecond,
		ErrorParser:                   newErrorsManager(),
	}
}

func newTestPgxPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	poolCfg, err := pgxpool.ParseConfig("postgres://127.0.0.1:1/db?connect_timeout=1")
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newTestHost(t *testing.T) *Pool {
	t.Helper()
	host, err := newPool(newTestPgxPool(t), PoolConfig{
		HealthCheckInterval: time.Hour,
		HealthCheckTimeout:  200 * time.Millisecond,
		ErrorParser:         newErrorsManager(),
	})
	if err != nil {
		t.Fatalf("newPool: %v", err)
	}
	t.Cleanup(host.Close)
	return host
}

func onlyUnhealthyHost(t *testing.T, rm *standbyManager) *Pool {
	t.Helper()
	var host *Pool
	_ = rm.unhealthyStore.Range(func(_ string, h *Pool) error {
		if host != nil {
			t.Fatal("expected exactly one unhealthy host")
		}
		host = h
		return nil
	})
	if host == nil {
		t.Fatal("expected one unhealthy host")
	}
	return host
}

// pgxPoolClosed is only for paths where AddStandby closes the host before
// it is published to a store, so tests have no *Pool handle.
func pgxPoolClosed(pool *pgxpool.Pool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	conn, err := pool.Acquire(ctx)
	if err == nil {
		conn.Release()
		return false
	}
	return strings.Contains(err.Error(), "closed pool")
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
