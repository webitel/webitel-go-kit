package consul

import "github.com/webitel/webitel-go-kit/infra/discovery"

// Registers the Consul discovery provider with the default factory.
// The provider is created with the given logger, address, and options.
// The options are applied to the provider in order.
func init() {
	discovery.DefaultFactory.Register(
		discovery.ProviderConsul,
		func(
			logger discovery.Logger,
			address string,
			options ...discovery.Option[discovery.DiscoveryProvider],
		) (discovery.DiscoveryProvider, error) {
			r, err := NewConsulRegistry(address, logger)
			if err != nil {
				return nil, err
			}

			for _, opt := range options {
				opt(r)
			}

			return r, nil
		},
	)
}
