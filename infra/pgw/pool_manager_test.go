package pgw

import (
	"context"
	"errors"
	"strings"
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

func TestStart_Twice(t *testing.T) {
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
	if err := m.Start(context.Background()); !errors.Is(err, ErrStarted) {
		t.Fatalf("second Start() = %v, want ErrStarted", err)
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
