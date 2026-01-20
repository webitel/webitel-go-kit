package grpc

import (
	"crypto/tls"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type ClientOption func(*clientOptions)

type clientOptions struct {
	target          string
	timeout         time.Duration
	tlsConfig       *tls.Config
	healthCheck     bool
	retryConfig     *RetryConfig
	poolConfig      PoolConfig
	interceptors    Interceptors
	dialOptions     []grpc.DialOption
	serviceConfig   *ServiceConfig
	keepaliveParams keepalive.ClientParameters
}

type RetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
}

type PoolConfig struct {
	InitialSize     int
	MaxSize         int
	IdleTimeout     time.Duration
	MaxLifeDuration time.Duration
}

type Interceptors struct {
	Unary  []grpc.UnaryClientInterceptor
	Stream []grpc.StreamClientInterceptor
}

func WithTarget(target string) ClientOption {
	return func(o *clientOptions) {
		o.target = target
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

func WithTLS(config *tls.Config) ClientOption {
	return func(o *clientOptions) {
		o.tlsConfig = config
	}
}

func WithHealthCheck() ClientOption {
	return func(o *clientOptions) {
		o.healthCheck = true
	}
}

func WithRetry(config RetryConfig) ClientOption {
	return func(o *clientOptions) {
		o.retryConfig = &config
	}
}

func WithPool(config PoolConfig) ClientOption {
	return func(o *clientOptions) {
		o.poolConfig = config
	}
}

func WithUnaryInterceptors(interceptors ...grpc.UnaryClientInterceptor) ClientOption {
	return func(o *clientOptions) {
		o.interceptors.Unary = append(o.interceptors.Unary, interceptors...)
	}
}

func WithStreamInterceptors(interceptors ...grpc.StreamClientInterceptor) ClientOption {
	return func(o *clientOptions) {
		o.interceptors.Stream = append(o.interceptors.Stream, interceptors...)
	}
}

func WithDialOptions(opts ...grpc.DialOption) ClientOption {
	return func(o *clientOptions) {
		o.dialOptions = append(o.dialOptions, opts...)
	}
}

func WithServiceConfig(config *ServiceConfig) ClientOption {
	return func(o *clientOptions) {
		o.serviceConfig = config
	}
}

func WithKeepalive(params keepalive.ClientParameters) ClientOption {
	return func(o *clientOptions) {
		o.keepaliveParams = params
	}
}

func WithLoadBalancing(policy string) ClientOption {
	return func(o *clientOptions) {
		if o.serviceConfig == nil {
			o.serviceConfig = &ServiceConfig{}
		}
		o.serviceConfig.LoadBalancingConfig = []map[string]any{
			{policy: map[string]any{}},
		}
	}
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

func AggressiveKeepalive() keepalive.ClientParameters {
	return keepalive.ClientParameters{
		Time:                5 * time.Second,
		Timeout:             2 * time.Second,
		PermitWithoutStream: true,
	}
}

func ConservativeKeepalive() keepalive.ClientParameters {
	return keepalive.ClientParameters{
		Time:                30 * time.Second,
		Timeout:             10 * time.Second,
		PermitWithoutStream: false,
	}
}
