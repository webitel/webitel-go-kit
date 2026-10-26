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

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{name: "empty DSN", cfg: Config{}, wantErr: "DSN is required"},
		{name: "ok", cfg: Config{DSN: "postgres://localhost/db"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

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
		t.Fatalf("app.Err() = %v, want a missing DSN error", err)
	}
}

func TestPool_ReadyBeforeOpen(t *testing.T) {
	db := &Pool{}

	err := db.Ready(context.Background())
	if !errors.Is(err, pgw.ErrUnreachable) {
		t.Fatalf("Ready() error = %v, want ErrUnreachable", err)
	}
}

func TestPool_CloseBeforeOpen(t *testing.T) {
	(&Pool{}).close()
}

func TestModule_ProvidesPool(t *testing.T) {
	var got *Pool

	app := fx.New(
		fx.NopLogger,
		Module(Config{DSN: "postgres://invalid:1/db"}),
		fx.Populate(&got),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("app.Err() = %v, want nil (open deferred to OnStart)", err)
	}
	if got == nil {
		t.Fatal("Pool was not provided")
	}
	if got.Manager() != nil {
		t.Fatal("Manager() before Start = non-nil, want nil")
	}
}

func TestModule_LifecycleOpenFails(t *testing.T) {
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
		t.Fatal("Start() = nil, want open error")
	}
	if !strings.Contains(err.Error(), "open pgw pool") {
		t.Fatalf("Start() error = %v, want open pgw pool", err)
	}
}

func TestModule_RegisterHealthOptional(t *testing.T) {
	var registry *health.Registry

	app := fx.New(
		fx.NopLogger,
		fx.Supply(slog.New(slog.NewTextHandler(io.Discard, nil))),
		healthfx.Module(healthfx.Config{}),
		Module(Config{
			DSN:            "postgres://invalid:1/db",
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

	// Check is registered; before Start it reports unreachable via pool state.
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
		Module(Config{DSN: "postgres://invalid:1/db"}),
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

func TestPool_PrimaryBeforeOpen(t *testing.T) {
	_, err := (&Pool{}).Primary()
	if !errors.Is(err, pgw.ErrUnreachable) {
		t.Fatalf("Primary() error = %v, want ErrUnreachable", err)
	}
}
