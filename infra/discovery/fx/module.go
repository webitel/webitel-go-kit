package discoveryfx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	"github.com/webitel/webitel-go-kit/infra/discovery/consul"
	"github.com/webitel/webitel-go-kit/infra/health"
)

// DefaultTimeout bounds every Consul API call.
const DefaultTimeout = 30 * time.Second

// Config describes the Consul agent and the service instance to register.
// Zero optional fields take their defaults.
type Config struct {
	// ConsulAddr is the address of the Consul agent.
	ConsulAddr string
	// Name is the service name. It is required and also seeds the instance ID.
	Name    string
	Version string
	// Endpoints are URLs of the service, such as "grpc://10.0.0.1:8080". The
	// first one is the address Consul checks.
	Endpoints []string
	Metadata  map[string]string
	// Timeout defaults to DefaultTimeout.
	Timeout time.Duration
}

func (c Config) withDefaults() Config {
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}

	return c
}

type params struct {
	fx.In

	Log       *slog.Logger
	Lifecycle fx.Lifecycle
	Health    *health.Registry `optional:"true"`
}

// Module provides discovery.DiscoveryProvider and registers the instance on
// the application's start. It needs a *slog.Logger in the graph. With a
// *health.Registry in the graph the TTL heartbeat reports its readiness;
// without one it always reports pass.
//
// The instance is deregistered on stop. The hook is added before any root
// level fx.Invoke runs, so healthfx.Shutdown drains the node before it.
func Module(cfg Config) fx.Option {
	cfg = cfg.withDefaults()

	return fx.Module("discovery",
		fx.Provide(func(p params) (discovery.DiscoveryProvider, error) {
			return build(cfg, p)
		}),
		fx.Invoke(func(discovery.DiscoveryProvider) {}),
	)
}

func build(cfg Config, p params) (discovery.DiscoveryProvider, error) {
	if cfg.Name == "" {
		return nil, errors.New("discoveryfx: service name is required")
	}

	opts := []discovery.Option[discovery.DiscoveryProvider]{
		discovery.WithHeartbeat[discovery.DiscoveryProvider](true),
		discovery.WithTimeout[discovery.DiscoveryProvider](cfg.Timeout),
	}

	if p.Health != nil {
		opts = append(opts, consul.WithReadiness(p.Health.ReadyFunc()))
	}

	provider, err := discovery.DefaultFactory.CreateProvider(discovery.ProviderConsul, p.Log, cfg.ConsulAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("create discovery provider: %w", err)
	}

	si := &discovery.ServiceInstance{
		Id:        discovery.GenerateInstanceID(cfg.Name),
		Name:      cfg.Name,
		Version:   cfg.Version,
		Metadata:  cfg.Metadata,
		Endpoints: cfg.Endpoints,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := provider.Register(ctx, si); err != nil {
				return fmt.Errorf("register %s in discovery: %w", si.Id, err)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := provider.Deregister(ctx, si); err != nil {
				return fmt.Errorf("deregister %s from discovery: %w", si.Id, err)
			}

			return nil
		},
	})

	return provider, nil
}
