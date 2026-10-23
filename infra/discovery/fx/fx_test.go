package discoveryfx

import (
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	"github.com/webitel/webitel-go-kit/infra/health"
	healthfx "github.com/webitel/webitel-go-kit/infra/health/fx"
)

func TestConfigWithDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   Config
		want time.Duration
	}{
		{name: "zero", want: DefaultTimeout},
		{name: "explicit value kept", in: Config{Timeout: time.Second}, want: time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.withDefaults().Timeout; got != tt.want {
				t.Errorf("withDefaults().Timeout = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestModule_RequiresName(t *testing.T) {
	agent := newAgent(t)

	app := fx.New(
		fx.NopLogger,
		fx.Supply(discardLogger()),
		Module(Config{ConsulAddr: agent.addr}),
	)

	if err := app.Err(); err == nil || !strings.Contains(err.Error(), "service name is required") {
		t.Fatalf("app.Err() = %v, want a missing name error", err)
	}
}

func TestModule_Lifecycle(t *testing.T) {
	agent := newAgent(t)

	cfg := Config{
		ConsulAddr: agent.addr,
		Name:       "im-test-service",
		Version:    "1.2.3",
		Endpoints:  []string{"grpc://127.0.0.1:8080"},
		Metadata:   map[string]string{"commit": "abc"},
	}

	var provider discovery.DiscoveryProvider

	app := fxtest.New(t,
		fx.Supply(discardLogger()),
		Module(cfg),
		fx.Populate(&provider),
	)

	if provider == nil {
		t.Fatal("DiscoveryProvider was not provided")
	}

	app.RequireStart()

	regs := agent.registrations()
	if len(regs) != 1 {
		t.Fatalf("registrations = %d, want 1", len(regs))
	}

	wantID := discovery.GenerateInstanceID(cfg.Name)
	if got := regs[0]; got.ID != wantID || got.Name != cfg.Name || !maps.Equal(got.Meta, cfg.Metadata) || got.Address != "127.0.0.1" || got.Port != 8080 {
		t.Errorf("registration = %+v, want ID %q, name %q, meta %v, 127.0.0.1:8080", got, wantID, cfg.Name, cfg.Metadata)
	}

	app.RequireStop()

	if got := agent.deregistrations(); !slices.Contains(got, wantID) {
		t.Errorf("deregistrations = %v, want %q", got, wantID)
	}
}

func TestModule_Readiness(t *testing.T) {
	tests := []struct {
		name   string
		health bool
		want   string
	}{
		// Without a registry the heartbeat always reports pass.
		{name: "without health registry", want: api.HealthPassing},
		// A registry with no checks is not ready.
		{name: "with health registry", health: true, want: api.HealthCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := newAgent(t)

			opts := []fx.Option{
				fx.Supply(discardLogger()),
				Module(Config{ConsulAddr: agent.addr, Name: "im-test-service", Endpoints: []string{"grpc://127.0.0.1:8080"}}),
			}

			if tt.health {
				opts = append(opts, healthfx.Module(healthfx.Config{}))
			}

			app := fxtest.New(t, opts...).RequireStart()
			defer app.RequireStop()

			if got := agent.waitTTL(t); got != tt.want {
				t.Errorf("TTL status = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestModule_DeregistersAfterDrain(t *testing.T) {
	tests := []struct {
		name  string
		order func(discovery, shutdown fx.Option) []fx.Option
	}{
		{
			name:  "module before shutdown",
			order: func(d, s fx.Option) []fx.Option { return []fx.Option{d, s} },
		},
		{
			name:  "module after shutdown",
			order: func(d, s fx.Option) []fx.Option { return []fx.Option{s, d} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := newAgent(t)

			var registry atomic.Pointer[health.Registry]

			agent.onDeregister = func() bool {
				h := registry.Load()

				return h != nil && h.Snapshot().Draining
			}

			opts := []fx.Option{
				fx.Supply(discardLogger()),
				healthfx.Module(healthfx.Config{DrainHold: 50 * time.Millisecond}),
				fx.Invoke(func(h *health.Registry) { registry.Store(h) }),
			}

			opts = append(opts, tt.order(
				Module(Config{ConsulAddr: agent.addr, Name: "im-test-service", Endpoints: []string{"grpc://127.0.0.1:8080"}}),
				healthfx.Shutdown(),
			)...)

			fxtest.New(t, opts...).RequireStart().RequireStop()

			drained := agent.drainedOnDeregister()
			if len(drained) == 0 {
				t.Fatal("instance was never deregistered")
			}

			for i, ok := range drained {
				if !ok {
					t.Errorf("deregister call %d ran before the drain", i)
				}
			}
		})
	}
}

// fakeAgent serves the Consul agent endpoints the provider calls.
type fakeAgent struct {
	addr         string
	onDeregister func() bool

	mu      sync.Mutex
	regs    []api.AgentServiceRegistration
	ttl     []string
	dereg   []string
	drained []bool
}

func newAgent(t *testing.T) *fakeAgent {
	t.Helper()

	a := new(fakeAgent)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/agent/self", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"Config":{"NodeName":"test-node"}}`)
	})
	mux.HandleFunc("PUT /v1/agent/service/register", func(_ http.ResponseWriter, r *http.Request) {
		var reg api.AgentServiceRegistration
		if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
			t.Errorf("decode registration: %v", err)
		}

		a.mu.Lock()
		a.regs = append(a.regs, reg)
		a.mu.Unlock()
	})
	mux.HandleFunc("PUT /v1/agent/check/update/", func(_ http.ResponseWriter, r *http.Request) {
		var upd struct{ Status string }
		if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
			t.Errorf("decode ttl update: %v", err)
		}

		a.mu.Lock()
		a.ttl = append(a.ttl, upd.Status)
		a.mu.Unlock()
	})
	mux.HandleFunc("PUT /v1/agent/service/deregister/{id}", func(_ http.ResponseWriter, r *http.Request) {
		drained := a.onDeregister != nil && a.onDeregister()

		a.mu.Lock()
		a.dereg = append(a.dereg, r.PathValue("id"))
		a.drained = append(a.drained, drained)
		a.mu.Unlock()
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	a.addr = strings.TrimPrefix(srv.URL, "http://")

	return a
}

func (a *fakeAgent) registrations() []api.AgentServiceRegistration {
	a.mu.Lock()
	defer a.mu.Unlock()

	return slices.Clone(a.regs)
}

func (a *fakeAgent) deregistrations() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	return slices.Clone(a.dereg)
}

func (a *fakeAgent) drainedOnDeregister() []bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return slices.Clone(a.drained)
}

// waitTTL returns the status of the first TTL update.
func (a *fakeAgent) waitTTL(t *testing.T) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for {
		a.mu.Lock()
		if len(a.ttl) > 0 {
			status := a.ttl[0]
			a.mu.Unlock()

			return status
		}
		a.mu.Unlock()

		if time.Now().After(deadline) {
			t.Fatal("no TTL update reached the agent")
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
