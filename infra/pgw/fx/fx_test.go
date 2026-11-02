package pgwfx

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/webitel/webitel-go-kit/infra/health"
	healthfx "github.com/webitel/webitel-go-kit/infra/health/fx"
	"github.com/webitel/webitel-go-kit/infra/pgw"
)

func TestConfigOptions(t *testing.T) {
	cfg := Config{
		ApplicationName: "svc",
		DSN:             "postgres://localhost/db",
		MaxConns:        8,
		MinConns:        2,
		StandbyDSN:      []string{"postgres://replica/db"},
	}

	opts := cfg.options(nil)
	if len(opts) != 3 {
		t.Fatalf("options() len = %d, want 3 (primary, app name, standby)", len(opts))
	}

	built := &pgw.Config{}
	for _, opt := range opts {
		opt(built)
	}

	if built.ApplicationName != "svc" {
		t.Errorf("ApplicationName = %q, want svc", built.ApplicationName)
	}
	if built.PrimaryConfig.DSN != cfg.DSN || built.PrimaryConfig.MaxConns != 8 || built.PrimaryConfig.MinConns != 2 {
		t.Errorf("PrimaryConfig = %+v, want DSN %q max 8 min 2", built.PrimaryConfig, cfg.DSN)
	}
	if len(built.StandbyConfig.DSN) != 1 || built.StandbyConfig.DSN[0] != cfg.StandbyDSN[0] {
		t.Errorf("StandbyConfig.DSN = %v, want %v", built.StandbyConfig.DSN, cfg.StandbyDSN)
	}
}

func TestConfigOptionsLeavesZeroLimits(t *testing.T) {
	opts := Config{DSN: "postgres://localhost/db"}.options(nil)
	built := &pgw.Config{}
	for _, opt := range opts {
		opt(built)
	}

	if built.PrimaryConfig.MaxConns != 0 || built.PrimaryConfig.MinConns != 0 {
		t.Fatalf("pool = max %d min %d, want zeros so pgw defaults apply",
			built.PrimaryConfig.MaxConns, built.PrimaryConfig.MinConns)
	}
}

func TestModule_RequiresDSN(t *testing.T) {
	app := fx.New(
		fx.NopLogger,
		Module(Config{}),
	)

	if err := app.Err(); err == nil || !strings.Contains(err.Error(), "DSN is required") {
		t.Fatalf("app.Err() = %v, want DSN required from pgw.New", err)
	}
}

func TestModule_ProvidesManagerBeforeStart(t *testing.T) {
	var got *pgw.PoolManager

	app := fx.New(
		fx.NopLogger,
		Module(Config{DSN: "postgres://localhost/db"}),
		fx.Populate(&got),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("app.Err() = %v, want nil (Start deferred to OnStart)", err)
	}
	if got == nil {
		t.Fatal("PoolManager was not provided")
	}
	if _, err := got.Primary(); !errors.Is(err, pgw.ErrUnreachable) {
		t.Fatalf("Primary() before Start = %v, want ErrUnreachable", err)
	}
}

func TestModule_LifecycleStartFails(t *testing.T) {
	dsn := "postgres://127.0.0.1:1/db?connect_timeout=1"
	app := fxtest.New(t,
		Module(Config{DSN: dsn}, pgw.WithPrimaryConfig(pgw.PrimaryConfig{
			DSN:                dsn,
			RetryAttempts:      1,
			HealthCheckTimeout: time.Second,
		})),
	)

	err := app.Start(context.Background())
	if err == nil {
		app.RequireStop()
		t.Fatal("Start() = nil, want start error")
	}
	if !strings.Contains(err.Error(), "start pgw pool") {
		t.Fatalf("Start() error = %v, want start pgw pool", err)
	}
}

func TestModule_RegisterHealthOptional(t *testing.T) {
	var registry *health.Registry

	app := fx.New(
		fx.NopLogger,
		fx.Supply(slog.New(slog.NewTextHandler(io.Discard, nil))),
		healthfx.Module(healthfx.Config{}),
		Module(Config{
			DSN:            "postgres://localhost/db",
			RegisterHealth: true,
		}),
		fx.Populate(&registry),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("app.Err() = %v, want nil", err)
	}
	if registry == nil {
		t.Fatal("health registry was not provided")
	}

	snap := registry.Snapshot()
	found := false
	for _, c := range snap.Checks {
		if c.System == health.SystemPostgreSQL {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("checks = %+v, want postgresql informational check", snap.Checks)
	}
}

func TestModule_RegisterHealthSkippedWithoutFlag(t *testing.T) {
	var registry *health.Registry

	app := fx.New(
		fx.NopLogger,
		fx.Supply(slog.New(slog.NewTextHandler(io.Discard, nil))),
		healthfx.Module(healthfx.Config{}),
		Module(Config{DSN: "postgres://localhost/db"}),
		fx.Populate(&registry),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("app.Err() = %v, want nil", err)
	}

	for _, c := range registry.Snapshot().Checks {
		if c.System == health.SystemPostgreSQL {
			t.Fatal("postgresql check registered without RegisterHealth")
		}
	}
}
