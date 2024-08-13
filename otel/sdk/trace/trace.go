package trace

import (
	"context"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/webitel/webitel-go-kit/otel/internal"
	sdk "go.opentelemetry.io/otel/sdk/trace"
)

// Options constructor to build
type (
	Option  = sdk.TracerProviderOption
	Options func(ctx context.Context, dsn string) ([]Option, error)
)

var (
	regedit sync.Mutex
	// map[scheme]options
	registry = make(map[string]Options)
)

func Register(scheme string, ctor Options) {
	input := scheme
	scheme = strings.TrimSpace(scheme)
	scheme = strings.ToLower(scheme)
	if scheme != input {
		panic(errors.Errorf("otel/sdk/trace.Register( scheme: %q ); invalid name", scheme))
	}
	if scheme == "" {
		panic(errors.Errorf("otel/sdk/trace.Register( scheme: ? ); name required"))
	}
	if ctor == nil {
		panic(errors.Errorf("otel/sdk/trace.Register( scheme: %q ); not implemented", scheme))
	}

	regedit.Lock()
	defer regedit.Unlock()
	if _, exists := registry[scheme]; exists {
		panic(errors.Errorf("otel/sdk/trace.Register( scheme: %q ); duplicate name", scheme))
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
		return nil, errors.Errorf("otel/sdk/trace.Options( scheme: %q ); not registered", scheme)
	}
	return driver(ctx, dsn)
}

func NewProvider(ctx context.Context, dsn string, opts ...Option) (*sdk.TracerProvider, error) {
	schemeOpts, err := NewOptions(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return sdk.NewTracerProvider(append(schemeOpts, opts...)...), nil
}
