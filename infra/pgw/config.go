package pgw

import (
	"context"
	"math"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/webitel-go-kit/pkg/safemap"
)

type Config struct {
	ApplicationName string

	Tracer Tracer

	PrimaryConfig PrimaryConfig

	StandbyConfig StandbyConfig

	MigrationVerifier MigrationVerifier
}

type MigrationVerifier func(ctx context.Context, conn *pgxpool.Conn) error

type PrimaryConfig struct {
	DSN string

	MaxConns int
	MinConns int

	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration

	RetryAttempts          int
	RetryInterval          time.Duration
	RetryStrategy          RetryStrategy
	RetryStrategyBaseValue int
}

type StandbyConfig struct {
	DSN []string

	MaxConns int
	MinConns int

	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration

	RetriesBeforeUnhealthy int
	RetryStrategy          RetryStrategy
	RetryStrategyBaseValue int

	PickStrategy                  PickStrategy
	UnhealthyReplicaRetryInterval time.Duration
}

type ConfigOption func(*Config)

type PickStrategy func(*safemap.SafeMap[string, *Pool]) *Pool

var (
	RandomPickStrategy PickStrategy = func(activeConnections *safemap.SafeMap[string, *Pool]) *Pool {
		hosts := make([]*Pool, 0, activeConnections.Len())

		activeConnections.Range(func(s string, h *Pool) error {
			hosts = append(hosts, h)
			return nil
		})

		if len(hosts) == 0 {
			return nil
		}

		randomIndex := rand.IntN(len(hosts))
		return hosts[randomIndex]

	}
	LeastConnectionsPickStrategy PickStrategy = func(activeConnections *safemap.SafeMap[string, *Pool]) *Pool {
		var (
			minConnections int
			result         *Pool
		)

		activeConnections.Range(func(s string, h *Pool) error {
			load := h.Stat().TotalConns() - h.Stat().IdleConns()
			if minConnections == 0 || int(load) < minConnections {
				minConnections = int(load)
				result = h
			}
			return nil
		})

		return result

	}
)

type RetryStrategy func(baseValue int, coefficient int) time.Duration

var (
	RetryStrategyExponential RetryStrategy = func(a int, x int) time.Duration {
		// y = a^x
		if a <= 0 || a == 1 {
			a = 2
		}
		return time.Duration(math.Pow(float64(a), float64(x))) * time.Second
	}
	RetryStrategyLinear RetryStrategy = func(a int, x int) time.Duration {
		// y = a*x
		if a == 0 {
			a = 1
		}
		return time.Duration(a*x) * time.Second
	}
)

var (
	DefaultPrimaryPoolConfig = PrimaryConfig{
		HealthCheckInterval:    5 * time.Second,
		HealthCheckTimeout:     3 * time.Second,
		RetryAttempts:          5,
		RetryInterval:          5 * time.Second,
		RetryStrategy:          RetryStrategyLinear,
		RetryStrategyBaseValue: 2,
	}

	DefaultStandbyPoolConfig = StandbyConfig{
		HealthCheckInterval:           5 * time.Second,
		HealthCheckTimeout:            3 * time.Second,
		PickStrategy:                  RandomPickStrategy,
		UnhealthyReplicaRetryInterval: 30 * time.Minute,
		RetriesBeforeUnhealthy:        5,
		RetryStrategy:                 RetryStrategyLinear,
		RetryStrategyBaseValue:        2,
	}
)

func WithApplicationName(name string) ConfigOption {
	return func(c *Config) { c.ApplicationName = name }
}

func WithTracer(t Tracer) ConfigOption {
	return func(c *Config) { c.Tracer = t }
}

func WithPrimaryConfig(cfg PrimaryConfig) ConfigOption {
	return func(c *Config) { c.PrimaryConfig = cfg }
}

func WithStandbyConfig(cfg StandbyConfig) ConfigOption {
	return func(c *Config) { c.StandbyConfig = cfg }
}

func WithMigrationVerifier(verifier MigrationVerifier) ConfigOption {
	return func(c *Config) { c.MigrationVerifier = verifier }
}
