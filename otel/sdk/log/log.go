package log

import (
	"context"
	"fmt"
	"strings"
	"sync"

	// "github.com/pkg/errors"
	"github.com/webitel/webitel-go-kit/otel/internal"
	"go.opentelemetry.io/otel"
	sdk "go.opentelemetry.io/otel/sdk/log"
)

type (
	// Option applies a configuration option value to an otel/sdk/log.LoggerProvider
	Option = sdk.LoggerProviderOption
	// Option to build an otel/sdk/log.LoggerProvider
	Options func(ctx context.Context, dsn string) ([]Option, error)
)

var (
	regedit  sync.Mutex
	registry = make(map[string]Options)
)

func Register(scheme string, ctor Options) {
	input := scheme
	scheme = strings.TrimSpace(scheme)
	scheme = strings.ToLower(scheme)
	if scheme != input {
		otel.Handle(fmt.Errorf("otel/sdk/log.Register( scheme: %q ); invalid name", scheme))
	}
	if scheme == "" {
		otel.Handle(fmt.Errorf("otel/sdk/log.Register( scheme: ? ); name required"))
	}
	if ctor == nil {
		otel.Handle(fmt.Errorf("otel/sdk/log.Register( scheme: %q ); not implemented", scheme))
	}

	regedit.Lock()
	defer regedit.Unlock()
	if _, exists := registry[scheme]; exists {
		otel.Handle(fmt.Errorf("otel/sdk/log.Register( scheme: %q ); duplicate name", scheme))
	}
	registry[scheme] = ctor
}

func NewOptions(ctx context.Context, dsn string) ([]Option, error) {
	scheme, _, err := internal.GetScheme(dsn)
	if err != nil {
		return nil, err
	}
	scheme = strings.ToLower(scheme)
	regedit.Lock()
	driver := registry[scheme]
	regedit.Unlock()
	if driver == nil {
		// return nil, fmt.Errorf("otel/sdk/log.Options( scheme: %q ); not registered", scheme)
		return nil, fmt.Errorf("scheme %s: unknown", scheme)
	}
	return driver(ctx, dsn)
}

func NewProvider(ctx context.Context, dsn string, opts ...Option) (*sdk.LoggerProvider, error) {
	driverOpts, err := NewOptions(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return sdk.NewLoggerProvider(append(driverOpts, opts...)...), nil
}
