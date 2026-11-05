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

func TestNew_RequiresDSN(t *testing.T) {
	_, err := New()
	if err == nil || !strings.Contains(err.Error(), "DSN is required") {
		t.Fatalf("New() = %v, want DSN required error", err)
	}
}

func TestNew_PrimaryBeforeStart(t *testing.T) {
	m, err := New(WithPrimaryConfig(PrimaryConfig{DSN: "postgres://localhost/db"}))
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	t.Cleanup(m.Close)

	if _, err := m.Primary(); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("Primary() before Start = %v, want ErrUnreachable", err)
	}
	if _, err := m.Standby(); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("Standby() before Start = %v, want ErrUnreachable", err)
	}
}

func TestClose_BeforeStartIdempotent(t *testing.T) {
	m, err := New(WithPrimaryConfig(PrimaryConfig{DSN: "postgres://localhost/db"}))
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	m.Close()
	m.Close()
}

func TestStart_AfterClose(t *testing.T) {
	m, err := New(WithPrimaryConfig(PrimaryConfig{DSN: "postgres://localhost/db"}))
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	m.Close()

	if err := m.Start(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("Start() after Close = %v, want ErrClosed", err)
	}
}

func TestStart_FailedAllowsRetry(t *testing.T) {
	m, err := New(WithPrimaryConfig(PrimaryConfig{
		DSN:                "postgres://127.0.0.1:1/db?connect_timeout=1",
		RetryAttempts:      1,
		HealthCheckTimeout: 50 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	t.Cleanup(m.Close)

	if err := m.Start(context.Background()); err == nil {
		t.Fatal("first Start() = nil, want connect error")
	}
	// Failed Start must not stick as "started"; retry should dial again.
	if err := m.Start(context.Background()); err == nil || errors.Is(err, ErrStarted) {
		t.Fatalf("second Start() = %v, want connect error (not ErrStarted)", err)
	}
}

func TestConnectWithRetry_CanceledContext(t *testing.T) {
	poolCfg, err := pgxpool.ParseConfig("postgres://127.0.0.1:1/db?connect_timeout=1")
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}

	pgxPool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}

	host, err := newPool(pgxPool, PoolConfig{
		HealthCheckInterval: time.Hour,
		HealthCheckTimeout:  50 * time.Millisecond,
		ErrorParser:         newErrorsManager(),
	})
	if err != nil {
		t.Fatalf("newPool: %v", err)
	}
	t.Cleanup(host.Close)

	cancelCtx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err = host.ConnectWithRetry(cancelCtx, 5, RetryStrategyLinear, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ConnectWithRetry() = %v, want context.Canceled", err)
	}
}

func TestReconnectMasterLoop_CloseCancelsRetry(t *testing.T) {
	inRetry := make(chan struct{})
	release := make(chan struct{})
	var signalOnce sync.Once
	var releaseOnce sync.Once
	releaseRetry := func() {
		releaseOnce.Do(func() { close(release) })
	}
	t.Cleanup(releaseRetry)

	m, err := New(WithPrimaryConfig(PrimaryConfig{
		DSN:           "postgres://127.0.0.1:1/db?connect_timeout=1",
		RetryInterval: 10 * time.Millisecond,
		RetryAttempts: 2,
		RetryStrategy: func(int, int) time.Duration {
			signalOnce.Do(func() { close(inRetry) })
			<-release
			return time.Hour
		},
		RetryStrategyBaseValue: 1,
		HealthCheckInterval:    time.Hour,
		HealthCheckTimeout:     200 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	t.Cleanup(m.Close)

	host := newTestHost(t)
	m.master.Store(host)

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.reconnectMasterLoop()
	}()

	select {
	case <-inRetry:
	case <-time.After(3 * time.Second):
		t.Fatal("master reconnect did not start")
	}

	m.Close()
	releaseRetry()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("reconnectMasterLoop did not exit after Close")
	}
	if host.GetState() != HostStateClosed {
		t.Fatalf("state = %s, want %s", host.GetState(), HostStateClosed)
	}
	if m.master.Load() != nil {
		t.Fatal("master pointer still set after Close")
	}
	if !m.isClosed() {
		t.Fatal("manager not marked closed")
	}
}
