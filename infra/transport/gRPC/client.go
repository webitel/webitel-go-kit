package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/webitel/webitel-go-kit/infra/transport/gRPC/pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type (
	ClientFactory[T any] func(*grpc.ClientConn) T

	Client[T any] struct {
		pool    *pool.Pool
		factory ClientFactory[T]
		opts    *clientOptions
	}

	ServiceConfig struct {
		LoadBalancingConfig []map[string]any   `json:"loadBalancingConfig,omitempty"`
		HealthCheckConfig   *HealthCheckConfig `json:"healthCheckConfig,omitempty"`
		RetryPolicy         *RetryPolicy       `json:"retryPolicy,omitempty"`
	}

	HealthCheckConfig struct {
		ServiceName string `json:"serviceName"`
	}

	RetryPolicy struct {
		MaxAttempts          int      `json:"maxAttempts"`
		InitialBackoff       string   `json:"initialBackoff"`
		MaxBackoff           string   `json:"maxBackoff"`
		BackoffMultiplier    float64  `json:"backoffMultiplier"`
		RetryableStatusCodes []string `json:"retryableStatusCodes"`
	}

	PoolStats struct {
		Capacity    int
		Available   int
		InUse       int
		Utilization float64
	}
)

var (
	defaultKeepaliveParams = keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	defaultServiceConfig = &ServiceConfig{
		LoadBalancingConfig: []map[string]any{
			{"round_robin": map[string]any{}},
		},
	}
	defaultPoolConfig = PoolConfig{
		InitialSize:     5,
		MaxSize:         20,
		IdleTimeout:     5 * time.Minute,
		MaxLifeDuration: 30 * time.Minute,
	}

	defaultTimeout = 30 * time.Second
)

func NewClient[T any](
	ctx context.Context,
	factory ClientFactory[T],
	opts ...ClientOption,
) (*Client[T], error) {
	options := &clientOptions{
		timeout:         defaultTimeout,
		poolConfig:      defaultPoolConfig,
		serviceConfig:   defaultServiceConfig,
		keepaliveParams: defaultKeepaliveParams,
	}

	for _, opt := range opts {
		opt(options)
	}

	if err := validateOptions(options); err != nil {
		return nil, fmt.Errorf("[gRPC client] invalid options: %w", err)
	}

	client := &Client[T]{
		factory: factory,
		opts:    options,
	}

	poolFactory := client.createPoolFactory()

	p, err := pool.NewWithContext(
		ctx,
		poolFactory,
		options.poolConfig.InitialSize,
		options.poolConfig.MaxSize,
		options.poolConfig.IdleTimeout,
		options.poolConfig.MaxLifeDuration,
	)
	if err != nil {
		return nil, fmt.Errorf("[gRPC client] failed to create connection pool: %w", err)
	}

	client.pool = p

	return client, nil
}

func (c *Client[T]) createPoolFactory() func(context.Context) (*grpc.ClientConn, error) {
	return func(ctx context.Context) (*grpc.ClientConn, error) {
		dialOpts := c.buildDialOptions()

		conn, err := grpc.NewClient(c.opts.target, dialOpts...)
		if err != nil {
			return nil, fmt.Errorf("[gRPC client] failed to dial %s: %w", c.opts.target, err)
		}

		return conn, nil
	}
}

func (c *Client[T]) buildDialOptions() []grpc.DialOption {
	opts := make([]grpc.DialOption, 0)

	if c.opts.tlsConfig != nil {
		opts = append(opts, grpc.WithTransportCredentials(
			credentials.NewTLS(c.opts.tlsConfig),
		))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if serviceConfig := c.buildServiceConfig(); serviceConfig != "" {
		opts = append(opts, grpc.WithDefaultServiceConfig(serviceConfig))
	}

	opts = append(opts, grpc.WithKeepaliveParams(c.opts.keepaliveParams))

	if len(c.opts.interceptors.Unary) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(c.opts.interceptors.Unary...))
	}
	if len(c.opts.interceptors.Stream) > 0 {
		opts = append(opts, grpc.WithChainStreamInterceptor(c.opts.interceptors.Stream...))
	}

	opts = append(opts, c.opts.dialOptions...)

	return opts
}

func (c *Client[T]) buildServiceConfig() string {
	if c.opts.serviceConfig == nil {
		return ""
	}

	config := c.opts.serviceConfig

	if c.opts.healthCheck && config.HealthCheckConfig == nil {
		config.HealthCheckConfig = &HealthCheckConfig{
			ServiceName: "",
		}
	}

	if c.opts.retryConfig != nil && config.RetryPolicy == nil {
		config.RetryPolicy = &RetryPolicy{
			MaxAttempts:       c.opts.retryConfig.MaxAttempts,
			InitialBackoff:    c.opts.retryConfig.InitialBackoff.String(),
			MaxBackoff:        c.opts.retryConfig.MaxBackoff.String(),
			BackoffMultiplier: c.opts.retryConfig.BackoffMultiplier,
			RetryableStatusCodes: []string{
				codes.Unavailable.String(),
				codes.DeadlineExceeded.String(),
			},
		}
	}

	data, err := json.Marshal(config)
	if err != nil {
		return ""
	}

	return string(data)
}

func (c *Client[T]) Execute(ctx context.Context, fn func(T) error) error {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return fmt.Errorf("[gRPC client] failed to get connection from pool: %w", err)
	}
	defer conn.Close()

	api := c.factory(conn.ClientConn)

	err = fn(api)

	if err != nil && isConnectionError(err) {
		conn.Unhealthy()
	}

	return err
}

func (c *Client[T]) GetAPI(ctx context.Context) (T, func() error, error) {
	var zero T

	conn, err := c.pool.Get(ctx)
	if err != nil {
		return zero, nil, fmt.Errorf("[gRPC client] failed to get connection: %w", err)
	}

	api := c.factory(conn.ClientConn)

	cleanup := func() error {
		return conn.Close()
	}

	return api, cleanup, nil
}

func (c *Client[T]) Close() error {
	if c.pool != nil {
		c.pool.Close()
	}
	return nil
}

func (c *Client[T]) Stats() PoolStats {
	if c.pool == nil || c.pool.IsClosed() {
		return PoolStats{}
	}

	capacity := c.pool.Capacity()
	available := c.pool.Available()

	return PoolStats{
		Capacity:    capacity,
		Available:   available,
		InUse:       capacity - available,
		Utilization: float64(capacity-available) / float64(capacity) * 100,
	}
}

func validateOptions(opts *clientOptions) error {
	if opts.target == "" {
		return fmt.Errorf("target is required")
	}

	return nil
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	code := st.Code()

	return slices.Contains([]codes.Code{
		codes.Unavailable,
		codes.Canceled,
		codes.Aborted,
	}, code)
}
