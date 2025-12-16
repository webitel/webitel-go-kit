package discovery

import "fmt"

type ProviderFactory[T any] func(logger Logger, url string, options ...Option[T]) (T, error)
type DiscoveryProviderType string

const (
	ProviderConsul DiscoveryProviderType = "consul"
)

var DefaultFactory = NewFactory[DiscoveryProvider]()

type Factory[T any] struct {
	providers map[DiscoveryProviderType]ProviderFactory[T]
}

// NewFactory returns a new instance of the Factory, which is used to register provider factories.
// The returned factory can then be used to create instances of the registered providers.
// The factory is thread-safe and can be used concurrently by multiple goroutines.
func NewFactory[T any]() *Factory[T] {
	return &Factory[T]{
		providers: make(map[DiscoveryProviderType]ProviderFactory[T]),
	}
}

// Register registers a provider factory for the given provider type.
// The registered factory is used to create instances of the provider when calling CreateProvider.
// The factory is thread-safe and can be used concurrently by multiple goroutines.
func (f *Factory[T]) Register(providerType DiscoveryProviderType, factory ProviderFactory[T]) {
	f.providers[providerType] = factory
}

// CreateProvider creates an instance of the given provider type.
// The provider is created by calling the registered provider factory with the given logger, address and options.
// If the provider type is not registered, an error is returned.
// The returned provider is thread-safe and can be used concurrently by multiple goroutines.
func (f *Factory[T]) CreateProvider(providerType DiscoveryProviderType, logger Logger, address string, options ...Option[T]) (T, error) {
	factory, exists := f.providers[providerType]
	if !exists {
		var zero T
		return zero, fmt.Errorf("unsupported discovery provider: %s", providerType)
	}

	return factory(logger, address, options...)
}
