package consul

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/webitel/webitel-go-kit/infra/discovery"
)

type listServiceFn func(ctx context.Context, service string, index uint64, passingOnly bool) ([]*discovery.ServiceInstance, uint64, error)

// interface guards to make sure that consul
// registry implement [DiscoveryProvider] interface
var (
	_ discovery.DiscoveryProvider = (*Registry)(nil)
)

type Registry struct {
	client            *Client
	kv                *kVClient
	enableHealthCheck bool
	registry          map[string]*serviceSet
	lock              sync.RWMutex
	timeout           time.Duration
}

// NewConsulRegistry creates a new Consul registry with the given address and logger.
// The returned registry is configured with the given address and logger.
// The registry is also configured with the following default values:
//   - enableHealthCheck: true
//   - timeout: 10 seconds
//   - healthCheckInterval: 10 seconds
//   - heartbeat: true
//   - deregisterCriticalServiceAfter: 600 seconds
//   - resolver: defaultResolver
//
// Parameters:
//   - addr: The address of the Consul agent.
//   - logger: The logger used for logging.
//
// Returns:
//   - *Registry: The created registry.
//   - error: An error object if any error occurs during the creation process, otherwise nil.
func NewConsulRegistry(addr string, logger discovery.Logger) (*Registry, error) {
	cfg := api.DefaultConfig()
	cfg.Address = addr
	apiClient, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if _, err := apiClient.Agent().NodeName(); err != nil {
		return nil, err
	}

	r := &Registry{
		registry:          make(map[string]*serviceSet),
		enableHealthCheck: true,
		timeout:           10 * time.Second,
		client: &Client{
			dc:                             SingleDatacenter,
			healthCheckInterval:            10,
			heartbeat:                      true,
			deregisterCriticalServiceAfter: 600,
			resolver:                       defaultResolver,
			client:                         apiClient,
			cancelers:                      make(map[string]*canceler),
			logger:                         logger,
		},
		kv: &kVClient{client: apiClient},
	}

	return r, nil
}

// #region Setters

// SetHealthCheck sets the enableHealthCheck flag of the Registry to the given value.
// If set to true, the registry will enable health checks for services registered with it.
func (r *Registry) SetHealthCheck(v bool) {
	r.enableHealthCheck = v
}

// SetTimeout sets the timeout for the registry.
// The timeout is used as the timeout for all Consul API calls made by the registry.
// If the timeout is zero, the default timeout of the registry is used.
func (r *Registry) SetTimeout(d time.Duration) {
	r.timeout = d
}

// SetDatacenter sets the data center of the Registry to the given value.
// The data center is used as the data center for all Consul API calls made by the registry.
// If the data center is not set, the default data center of the registry is used.
// Valid values for the data center are "SINGLE" and "MULTI".
// If the given data center is not valid, the data center is not updated.
func (r *Registry) SetDatacenter(dc string) {
	parsed := Datacenter(dc)
	if (parsed == SingleDatacenter || parsed == MultiDataCenter) && r.client != nil {
		r.client.dc = parsed
	} else if r.client != nil {
		r.client.dc = SingleDatacenter
	}
}

// SetHeartbeatEnabled sets the heartbeat flag of the Registry to the given value.
// If set to true, the registry will start a goroutine to update the TTL of the service registration periodically.
// If set to false, the goroutine is stopped and the TTL is no longer updated.
func (r *Registry) SetHeartbeatEnabled(enabled bool) {
	if r.client != nil {
		r.client.heartbeat = enabled
	}
}

// SetHealthCheckInterval sets the health check interval of the Registry to the given value.
// The health check interval is used as the interval for all health checks made by the registry.
// If the interval is zero, the default health check interval of the registry is used.
// The unit of the interval is seconds.
// If the interval is negative, the health check interval is disabled.
func (r *Registry) SetHealthCheckInterval(interval int) {
	if r.client != nil {
		r.client.healthCheckInterval = interval
	}
}

// SetDeregisterCriticalServiceAfter sets the deregister critical service after interval of the Registry to the given value.
// The deregister critical service after interval is used as the interval after which the registry will deregister critical services.
// If the interval is zero, the default deregister critical service after interval of the registry is used.
// The unit of the interval is seconds.
// If the interval is negative, the deregister critical service after interval is disabled.
func (r *Registry) SetDeregisterCriticalServiceAfter(interval int) {
	if r.client != nil {
		r.client.deregisterCriticalServiceAfter = interval
	}
}

// SetTags sets the tags of the Registry to the given tags.
// The tags are used as additional metadata for the registered services.
// If the tags are nil, the default tags of the Registry are used.
func (r *Registry) SetTags(tags ...string) {
	if r.client != nil {
		r.client.tags = tags
	}
}

//#endregion

//#region Public

// Register registers the given service instance with the Consul agent.
// The method takes a context.Context and a *discovery.ServiceInstance as input parameters.
// It enables health checks for the service if the enableHealthCheck field of the Registry is set to true.
// The method returns an error if the registration fails, otherwise nil.
func (r *Registry) Register(ctx context.Context, svc *discovery.ServiceInstance) error {
	return r.client.Register(ctx, svc, r.enableHealthCheck)
}

// Deregister deregisters the given service instance from the Consul agent.
// The method takes a context.Context and a *discovery.ServiceInstance as input parameters.
// It returns an error if the deregistration fails, otherwise nil.
func (r *Registry) Deregister(ctx context.Context, svc *discovery.ServiceInstance) error {
	return r.client.Deregister(ctx, svc.Id)
}

// GetService returns the list of service instances for the given service name.
// It takes a context.Context and a service name as input parameters.
// If the service is not resolved in the registry, it will try to get the service from the Consul agent.
// If the service is not found in the Consul agent, it will return an error.
// Otherwise, it will return the list of service instances.
func (r *Registry) GetService(ctx context.Context, serviceName string) ([]*discovery.ServiceInstance, error) {
	r.lock.RLock()         //+[R] lock
	defer r.lock.RUnlock() //-[R] lock

	var (
		getRemote = func() []*discovery.ServiceInstance {
			services, _, err := r.client.GetService(ctx, serviceName, 0, true)
			if err == nil && len(services) > 0 {
				return services
			}

			return nil
		}
		set = r.registry[serviceName]
	)

	if set == nil {
		if s := getRemote(); len(s) > 0 {
			return s, nil
		}
		return nil, fmt.Errorf("service %s not resolved in registry", serviceName)
	}

	ss, _ := set.services.Load().([]*discovery.ServiceInstance)
	if ss == nil {
		if s := getRemote(); len(s) > 0 {
			return s, nil
		}
		return nil, fmt.Errorf("service %s not resolved in registry", serviceName)
	}

	return ss, nil
}

// GetWatcher returns a Watcher for the given service name.
// The Watcher is a data structure that can be used to watch the service instances of the given service name.
// The Watcher is created by calling the GetService method of the Registry and resolves the service instances of the given service name.
// The GetWatcher method takes a context.Context and a service name as input parameters.
// If the context is canceled, it will return an error.
// If the service is not resolved in the registry, it will try to get the service from the Consul agent.
// If the service is not found in the Consul agent, it will return an error.
// Otherwise, it will return the Watcher.
func (r *Registry) GetWatcher(ctx context.Context, serviceName string) (discovery.Watcher, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var (
		set *serviceSet
		ok  bool
	)

	r.lock.Lock() //+[RW] lock
	if set, ok = r.registry[serviceName]; !ok {
		cancelCtx, cancel := context.WithCancel(context.Background())
		set = new(serviceSet)
		{
			set.registry = r
			set.watcher = make(map[*watcher]struct{})
			set.services = new(atomic.Value)
			set.serviceName = serviceName
			set.ctx = cancelCtx
			set.cancel = cancel
		}

		r.registry[serviceName] = set
	}

	set.ref.Add(1)
	r.lock.Unlock() //-[RW] lock

	watcher := new(watcher)
	{
		watcher.event = make(chan struct{}, 1)
		watcher.ctx, watcher.cancel = context.WithCancel(ctx)
		watcher.set = set
	}

	set.lock.Lock() //+[RW] lock
	set.watcher[watcher] = struct{}{}
	set.lock.Unlock() //-[RW] lock

	ss, _ := set.services.Load().([]*discovery.ServiceInstance)
	if len(ss) > 0 {
		select {
		case watcher.event <- struct{}{}:
		default:
		}
	}

	if !ok {
		if err := r.resolve(ctx, set); err != nil {
			return nil, err
		}
	}

	return watcher, nil
}

// ListServices returns a map of service name to list of service instances.
// The map is populated by iterating over the registry and appending
// the service instances of each service to the result map.
// The method is thread-safe and protected by a read-write lock.
func (r *Registry) ListServices() map[string][]*discovery.ServiceInstance {
	var (
		allServices = make(map[string][]*discovery.ServiceInstance)
	)

	r.lock.RLock()         //+[R] lock
	defer r.lock.RUnlock() //-[R] lock

	for name, set := range r.registry {
		var (
			services []*discovery.ServiceInstance
			ss, _    = set.services.Load().([]*discovery.ServiceInstance)
		)

		if ss != nil {
			services = append(services, ss...)
			allServices[name] = services
		}
	}

	return allServices
}

// KV returns the KVProvider interface associated with the registry.
// It is used to interact with the key-value store of the registry.
func (r *Registry) KV() discovery.KVProvider {
	return r.kv
}

//#endregion

//#region Private

// resolve resolves the service instances of the given service name by calling the
// GetService method of the Consul client. If the timeout field of the Registry is set,
// it will wrap the GetService method with a context.WithTimeout call. If the service
// instances are resolved successfully, it will broadcast the service instances to all
// registered watchers of the service set. The resolve method is run in a separate
// goroutine and will continuously resolve the service instances until the context is
// canceled.
func (r *Registry) resolve(ctx context.Context, ss *serviceSet) error {
	listServices := r.client.GetService
	if r.timeout > 0 {
		listServices = func(ctx context.Context, service string, index uint64, passingOnly bool) ([]*discovery.ServiceInstance, uint64, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
			defer cancel()

			return r.client.GetService(timeoutCtx, service, index, passingOnly)
		}
	}

	services, idx, err := listServices(ctx, ss.serviceName, 0, true)
	if err != nil {
		return err
	}
	if len(services) > 0 {
		ss.broadcast(services)
	}

	go r.watchServiceChange(ss, idx, listServices, services)

	return nil
}

// watchServiceChange continuously resolves the service instances of the given service name
// by calling the given listService function. If the listService function returns an error,
// it will sleep for one second and try again. If the context is canceled, it will stop
// watching for service changes. If the service instances are resolved successfully, it
// will broadcast the service instances to all registered watchers of the service set.
// The watchServiceChange method is run in a separate goroutine and will continuously watch
// for service changes until the context is canceled.
func (r *Registry) watchServiceChange(ss *serviceSet, lastIndex uint64, listService listServiceFn, services []*discovery.ServiceInstance) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tmpService, tmpIndex, err := listService(ss.ctx, ss.serviceName, lastIndex, true)
			if err != nil {
				if err := sleepCtx(ss.ctx, time.Second); err != nil {
					return
				}
				continue
			}

			if len(tmpService) != 0 && tmpIndex != lastIndex {
				services = tmpService
				ss.broadcast(services)
			}
			lastIndex = tmpIndex
		case <-ss.ctx.Done():
			return
		}
	}
}

// tryDelete tries to delete the given service set from the registry.
// It locks the registry, decrements the reference count of the service set,
// cancels the service set's context, and deletes the service set from the registry.
// If the reference count of the service set is not zero, it will return false.
// If the deletion is successful, it will return true.
// The tryDelete method is thread-safe and can be called concurrently from multiple goroutines.
func (r *Registry) tryDelete(ss *serviceSet) bool {
	r.lock.Lock()         //+[RW] lock
	defer r.lock.Unlock() //-[RW] lock

	if ss.ref.Add(-1) != 0 {
		return false
	}
	ss.cancel()
	delete(r.registry, ss.serviceName)

	return true
}

//#endregion
