package log

import (
	"context"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/webitel/webitel-go-kit/otel/internal"
	sdk "go.opentelemetry.io/otel/sdk/log"
)

type (
	Option  = sdk.LoggerProviderOption
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
		panic(errors.Errorf("otel/sdk/log.Register( scheme: %q ); invalid name", scheme))
	}
	if scheme == "" {
		panic(errors.Errorf("otel/sdk/log.Register( scheme: ? ); name required"))
	}
	if ctor == nil {
		panic(errors.Errorf("otel/sdk/log.Register( scheme: %q ); not implemented", scheme))
	}

	regedit.Lock()
	defer regedit.Unlock()
	if _, exists := registry[scheme]; exists {
		panic(errors.Errorf("otel/sdk/log.Register( scheme: %q ); duplicate name", scheme))
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
		return nil, errors.Errorf("otel/sdk/log.Options( scheme: %q ); not registered", scheme)
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
