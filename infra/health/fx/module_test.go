package healthfx

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/infra/health"
)

func TestConfigWithDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   Config
		want Config
	}{
		{
			name: "zero",
			want: Config{
				StartTimeout: DefaultStartTimeout,
				DrainHold:    DefaultDrainHold,
				StopTimeout:  DefaultDrainHold + stopMargin,
			},
		},
		{
			name: "stop timeout follows a custom drain hold",
			in:   Config{DrainHold: time.Second},
			want: Config{
				StartTimeout: DefaultStartTimeout,
				DrainHold:    time.Second,
				StopTimeout:  time.Second + stopMargin,
			},
		},
		{
			name: "explicit values kept",
			in:   Config{HTTPAddr: "127.0.0.1:1", StartTimeout: 2 * time.Second, DrainHold: 3 * time.Second, StopTimeout: 4 * time.Second},
			want: Config{HTTPAddr: "127.0.0.1:1", StartTimeout: 2 * time.Second, DrainHold: 3 * time.Second, StopTimeout: 4 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.withDefaults(); got != tt.want {
				t.Errorf("withDefaults() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestModule_Lifecycle(t *testing.T) {
	addr := freeAddr(t)

	var (
		registry                *health.Registry
		drainedBeforeServerStop atomic.Bool
	)

	app := fx.New(
		fx.NopLogger,
		fx.Supply(slog.New(slog.NewTextHandler(io.Discard, nil))),
		Module(Config{HTTPAddr: addr, DrainHold: 300 * time.Millisecond}),
		fx.Invoke(func(lc fx.Lifecycle, h *health.Registry) {
			h.Critical("node", func(context.Context) error { return nil })

			lc.Append(fx.Hook{
				OnStop: func(context.Context) error {
					drainedBeforeServerStop.Store(h.Snapshot().Draining)

					return nil
				},
			})
		}),
		Shutdown(),
		fx.Populate(&registry),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		t.Fatalf("start app: %v", err)
	}

	waitReadyz(t, "http://"+addr+"/readyz")

	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("stop app: %v", err)
	}

	if !drainedBeforeServerStop.Load() {
		t.Error("server stop hook ran before the drain; Shutdown() must run first")
	}

	if got := registry.Snapshot().State; got != health.StateStopping {
		t.Errorf("state after stop = %s, want %s", got, health.StateStopping)
	}
}

func waitReadyz(t *testing.T, url string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for {
		resp, err := http.Get(url)
		if err == nil {
			if cerr := resp.Body.Close(); cerr != nil {
				t.Errorf("close body: %v", cerr)
			}

			if resp.StatusCode == http.StatusOK {
				return
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("GET %s never returned 200: last err %v", url, err)
		}

		time.Sleep(20 * time.Millisecond)
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick free port: %v", err)
	}

	addr := l.Addr().String()

	if err := l.Close(); err != nil {
		t.Fatalf("release free port: %v", err)
	}

	return addr
}
