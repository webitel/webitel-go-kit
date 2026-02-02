package consul

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/webitel/webitel-go-kit/infra/discovery"
)

// # region Internal types
type Datacenter string
type ServiceResolver func(ctx context.Context, entries []*api.ServiceEntry) []*discovery.ServiceInstance
type canceler struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

//# endregion

// #region Constants
const (
	SingleDatacenter Datacenter = "SINGLE"
	MultiDataCenter  Datacenter = "MULTI"
)

const DeregisterNotFoundCode int = 404
const ServiceStr string = "service:" //service prefix str used in check and api calls
var EndpointSchemes []string = []string{"lan_ipv4", "wan_ipv4", "lan_ipv6", "wan_ipv6"}

var (
	TTLContextCanceledErr = errors.New("context canceled")
)

//#endregion

type Client struct {
	dc     Datacenter
	client *api.Client

	resolver                       ServiceResolver
	healthCheckInterval            int
	heartbeat                      bool
	deregisterCriticalServiceAfter int
	serviceChecks                  api.AgentServiceChecks
	tags                           []string
	logger                         discovery.Logger

	lock      sync.RWMutex
	cancelers map[string]*canceler
}

// #region Default resolver

// The function takes a context.Context and a slice of *api.ServiceEntry as input parameters.
// It iterates over the ServiceEntries and creates a ServiceInstance for each entry.
// The ServiceInstance includes the ID, name, metadata, version, and endpoints of the ServiceEntry.
// The function appends each ServiceInstance to the services slice and returns it.
func defaultResolver(_ context.Context, entries []*api.ServiceEntry) []*discovery.ServiceInstance {
	services := make([]*discovery.ServiceInstance, 0, len(entries))
	for _, entry := range entries {
		var (
			version   = versionBuilder(entry.Service.Tags)
			endpoints = grpcEndpointsBuilder(entry.Service.TaggedAddresses, entry.Service.Address, entry.Service.Port)
			svc       = &discovery.ServiceInstance{
				Id:        entry.Service.ID,
				Name:      entry.Service.Service,
				Metadata:  entry.Service.Meta,
				Version:   version,
				Endpoints: endpoints,
			}
		)

		services = append(services, svc)
	}

	return services
}

// versionBuilder takes a list of tags and returns the version string if the
// "version" tag is present. If the tag is not present, an empty string is
// returned.
func versionBuilder(tags []string) string {
	var version string
	for _, tag := range tags {
		ss := strings.SplitN(tag, "=", 2)
		if len(ss) == 2 && ss[0] == "version" {
			version = ss[1]
		}
	}
	return version
}

// grpcEndpointsBuilder takes a map of tagged addresses, an address, and a port and
// returns a slice of grpc endpoints. If the map contains any of the schemes
// in EndpointSchemes, those addresses are skipped. If the returned slice is empty
// and the address and port are not empty, a single grpc endpoint is added to the
// slice in the format "grpc://<address>:<port>".
func grpcEndpointsBuilder(taggedAddresses map[string]api.ServiceAddress, address string, port int) []string {
	endpoints := make([]string, 0)
	for scheme, addr := range taggedAddresses {
		if slices.Contains(EndpointSchemes, scheme) {
			continue
		}

		endpoints = append(endpoints, addr.Address)
	}

	shouldUseDefaultEndpoint := len(endpoints) == 0 && address != "" && port != 0
	if shouldUseDefaultEndpoint {
		endpoints = append(endpoints, fmt.Sprintf("grpc://%s:%d", address, port))
	}

	return endpoints
}

// #endregion Default resolver

//#region Get service

// GetService is a function that returns a list of service instances and
// query metadata for the given service name in a single or multiple
// data centers. It takes a context, a service name, an index, and a
// passingOnly flag as parameters. The returned list of service instances
// will have their metadata populated with the data center that each instance
// belongs to. The function will block until the given index is reached or
// the context is canceled. The function will return an error if there is an
// error communicating with the consul server.
func (c *Client) GetService(ctx context.Context, serviceName string, index uint64, passingOnly bool) ([]*discovery.ServiceInstance, uint64, error) {
	if c.dc == MultiDataCenter {
		return c.multiDCService(ctx, serviceName, index, passingOnly)
	}

	opts := (&api.QueryOptions{
		WaitIndex:  index,
		WaitTime:   time.Second * 55,
		Datacenter: string(c.dc),
	}).WithContext(ctx)

	if c.dc == SingleDatacenter {
		opts.Datacenter = ""
	}

	entries, meta, err := c.singleDCEntries(serviceName, "", passingOnly, opts)
	if err != nil {
		return nil, 0, err
	}

	return c.resolver(ctx, entries), meta.LastIndex, nil
}

// singleDCEntries is a function that returns a list of service entries and
// query metadata for the given service name and tag in a single data
// center. It takes a context, a service name, a tag, a passingOnly flag,
// and query options as parameters. The returned list of service entries
// will have their metadata updated according to the query options.
func (c *Client) singleDCEntries(serviceName, tag string, passingOnly bool, opts *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error) {
	return c.client.Health().Service(serviceName, tag, passingOnly, opts)
}

// multiDCService is a function that returns a list of service instances
// for the given service name across all datacenters. It takes a context,
// a service name, an index, and a passingOnly flag as parameters.
// The returned list of service instances will have their metadata populated
// with the data center that each instance belongs to. The function will
// block until the given index is reached or the context is canceled.
// The function will return an error if there is an error communicating with
// the consul server.
func (c *Client) multiDCService(ctx context.Context, service string, index uint64, passingOnly bool) ([]*discovery.ServiceInstance, uint64, error) {
	opts := (&api.QueryOptions{
		WaitIndex: index,
		WaitTime:  time.Second * 55,
	}).WithContext(ctx)

	dcs, err := c.client.Catalog().Datacenters()
	if err != nil {
		return nil, 0, err
	}

	var instances []*discovery.ServiceInstance
	for _, dc := range dcs {
		opts.Datacenter = dc

		e, m, err := c.singleDCEntries(service, "", passingOnly, opts)
		if err != nil {
			return nil, 0, err
		}

		inst := c.resolver(ctx, e)
		for _, in := range inst {
			if in.Metadata == nil {
				in.Metadata = make(map[string]string, 1)
				in.Metadata["dc"] = dc
			}

			instances = append(instances, in)
			opts.WaitIndex = m.LastIndex
		}
	}

	return instances, opts.WaitIndex, nil
}

//#endregion

// #region Register

// Register registers a service instance with the Consul agent.
//
// The method takes a context.Context, a *discovery.ServiceInstance, and a boolean flag for enabling health checks as input parameters.
// It builds a registration object using the buildRegistration method and applies the necessary checks using the applyChecks method.
// It prepares a canceler for the heartbeat and registers the service using the registerService method.
// If the heartbeat flag is set to true, it starts the heartbeat using the startHeartbeat method.
//
// Parameters:
//   - ctx: The context.Context object used for cancellation or timeouts.
//   - svc: The *discovery.ServiceInstance object representing the service instance to be registered.
//   - enableHealthChecks: A boolean flag indicating whether to enable health checks for the service.
//
// Returns:
//   - error: An error object if any error occurs during the registration process, otherwise nil.
func (c *Client) Register(ctx context.Context, svc *discovery.ServiceInstance, enableHealthChecks bool) error {
	//[A]gent [S]ervice [R]egistration
	asr, checkAddresses, err := c.buildRegistration(svc)
	if err != nil {
		return err
	}

	c.applyChecks(asr, svc, checkAddresses, enableHealthChecks)

	cc := c.prepareHeartbeat(svc.Id)

	if err := c.registerService(ctx, asr, cc); err != nil {
		return err
	}

	if c.heartbeat {
		c.startHeartbeat(asr, svc.Id, cc)
	}

	return nil
}

// buildRegistration creates a service registration object from the given service
// instance. The object is then used to register the service with the
// consul server.
//
// The returned addresses are the addresses that will be used for the
// health checks of the service.
func (c *Client) buildRegistration(svc *discovery.ServiceInstance) (*api.AgentServiceRegistration, []string, error) {
	var (
		addresses      = make(map[string]api.ServiceAddress)
		checkAddresses = make([]string, 0, len(svc.Endpoints))
	)

	for _, endpoint := range svc.Endpoints {
		raw, err := url.Parse(endpoint)
		if err != nil {
			return nil, nil, err
		}

		var (
			addr    = raw.Hostname()
			port, _ = strconv.ParseUint(raw.Port(), 10, 16)
		)

		checkAddresses = append(checkAddresses, net.JoinHostPort(addr, strconv.FormatUint(port, 10)))
		addresses[raw.Scheme] = api.ServiceAddress{
			Address: endpoint,
			Port:    int(port),
		}
	}

	tags := append([]string{fmt.Sprintf("version=%s", svc.Version)}, c.tags...)
	if len(c.tags) > 0 {
		tags = append(tags, c.tags...)
	}

	//[A]gent [S]ervice [R]egistration
	asr := new(api.AgentServiceRegistration)
	{
		asr.ID = svc.Id
		asr.Name = svc.Name
		asr.Meta = svc.Metadata
		asr.Tags = c.tags
		asr.TaggedAddresses = addresses
	}

	if len(checkAddresses) > 0 {
		host, portRaw, _ := net.SplitHostPort(checkAddresses[0])
		port, _ := strconv.ParseInt(portRaw, 10, 32)
		asr.Address = host
		asr.Port = int(port)
	}

	return asr, checkAddresses, nil
}

// applyChecks applies the necessary health checks and TTL checks to the given
// service registration object. If enableHealthChecks is true, TCP health checks
// and custom health checks are added to the registration object. If c.heartbeat
// is true, a TTL check is added to the registration object.
func (c *Client) applyChecks(asr *api.AgentServiceRegistration, svc *discovery.ServiceInstance, addresses []string, enableHealthChecks bool) {
	if enableHealthChecks {
		// TCP health checks
		for _, addr := range addresses {
			asr.Checks = append(asr.Checks, c.tcpCheck(addr))
		}

		// custom checks
		asr.Checks = append(asr.Checks, c.serviceChecks...)
	}

	// TTL checks
	if c.heartbeat {
		asr.Checks = append(asr.Checks, c.ttlCheck(svc))
	}
}

// tcpCheck creates a TCP health check for the given address.
// It returns a new AgentServiceCheck with the given address, the health check interval,
// the deregister critical service after interval, and a timeout of 5 seconds.
// The returned check can be used to register a service with consul.
func (c *Client) tcpCheck(addr string) *api.AgentServiceCheck {
	check := new(api.AgentServiceCheck)
	{
		check.TCP = addr
		check.Interval = fmt.Sprintf("%ds", c.healthCheckInterval)
		check.DeregisterCriticalServiceAfter = fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter)
		check.Timeout = "5s"
	}

	return check
}

// ttlCheck creates a TTL health check for the given service instance.
// It returns a new AgentServiceCheck with the given service ID, the health check interval,
// and the deregister critical service after interval. The returned check can be used
// to register a service with consul.
func (c *Client) ttlCheck(svc *discovery.ServiceInstance) *api.AgentServiceCheck {
	check := new(api.AgentServiceCheck)
	{
		check.CheckID = ServiceStr + svc.Id + ":ttl:1"
		check.TTL = fmt.Sprintf("%ds", c.healthCheckInterval*2)
		check.DeregisterCriticalServiceAfter = fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter)
	}
	return check
}

// prepareHeartbeat returns a canceller for the given service ID. It will cancel the previous
// heartbeat if it exists and start a new one. The returned canceller can be used to cancel the
// heartbeat. The heartbeat will be stopped when the returned canceller is canceled.
func (c *Client) prepareHeartbeat(serviceId string) *canceler {
	if !c.heartbeat {
		return nil
	}

	c.lock.Lock() //+[RW] lock
	prev, exists := c.cancelers[serviceId]
	c.lock.Unlock() //-[RW] lock

	if exists {
		prev.cancel()
		<-prev.done
	}

	var (
		ctx, cancel = context.WithCancel(context.Background())
		cc          = &canceler{
			ctx:    ctx,
			cancel: cancel,
			done:   make(chan struct{}),
		}
	)

	c.lock.Lock() //+[RW] lock
	c.cancelers[serviceId] = cc
	c.lock.Unlock() //-[RW] lock

	go func() {
		<-cc.done
		cc.cancel()

		c.lock.Lock() //+[RW] lock
		if c.cancelers[serviceId] == cc {
			delete(c.cancelers, serviceId)
		}
		c.lock.Unlock() //-[RW] lock
	}()

	return cc
}

// registerService registers the given service with the Consul agent. The
// serviceId is used to identify the service and is used to cancel the
// heartbeat. The cc is the canceller returned by prepareHeartbeat and
// should be used to stop the heartbeat. If the registration fails, the
// canceller will be canceled to stop the heartbeat.
func (c *Client) registerService(ctx context.Context, asr *api.AgentServiceRegistration, cc *canceler) error {
	err := c.client.Agent().ServiceRegisterOpts(asr, api.ServiceRegisterOpts{}.WithContext(ctx))
	if err != nil {
		if c.heartbeat {
			close(cc.done)
		}

		return err
	}

	return nil
}

// sendTTL updates the ttl of the service registration with the given serviceId.
// If the context is canceled or the UpdateTTL call failed with a context.DeadlineExceeded
// error, the function will cancel the heartbeat and deregister the service. Otherwise, it
// will return the error.
func (c *Client) sendTTL(ctx context.Context, serviceId string) error {
	err := c.client.Agent().UpdateTTLOpts(ServiceStr+serviceId, "pass", "pass", new(api.QueryOptions).WithContext(ctx))
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		_ = c.client.Agent().ServiceDeregister(serviceId)
		return TTLContextCanceledErr
	}
	return err
}

// retryRegister retries to register the given service with the Consul agent if the
// context is not canceled. If the context is canceled, it will deregister the
// service to avoid the service registration being stuck in the Consul agent.
// If the registration fails, it will log an error. Otherwise, it will log a
// warning message indicating that the re registration of the service occurred
// successfully.
func (c *Client) retryRegister(ctx context.Context, serviceId string, asr *api.AgentServiceRegistration) {
	if err := sleepCtx(ctx, time.Duration(rand.IntN(5))*time.Second); err != nil {
		_ = c.client.Agent().ServiceDeregister(serviceId)
		return
	}

	if err := c.client.Agent().ServiceRegisterOpts(asr, api.ServiceRegisterOpts{}.WithContext(ctx)); err != nil {
		c.logger.Error("[Consul] re-registry service failed", "err", err)
	} else {
		c.logger.Warn("[Consul] re registry of service occurred success")
	}

}

// startHeartbeat starts a goroutine to update the ttl of the service registration
// periodically. The goroutine will stop when the context is canceled or
// the deregister call failed with a context.Canceled or context.DeadlineExceeded
// error. If the UpdateTTL call failed, the goroutine will sleep for a random
// duration between 1 and 5 seconds, and then retry to update the ttl or
// re-register the service if the last update ttl call failed.
func (c *Client) startHeartbeat(asr *api.AgentServiceRegistration, serviceId string, cc *canceler) {
	go func() {
		defer close(cc.done)
		if err := c.client.Agent().UpdateTTL(ServiceStr+serviceId+":ttl:1", "pass", "pass"); err != nil {
			c.logger.Error("[Consul]update ttl heartbeat to consul failed!", "err", err)
		}

		ticker := time.NewTicker(time.Second * time.Duration(c.healthCheckInterval))
		defer ticker.Stop()

		for {
			select {
			case <-cc.ctx.Done():
				_ = c.client.Agent().ServiceDeregister(serviceId)
				return
			case <-ticker.C:
				if err := c.sendTTL(cc.ctx, serviceId); err != nil {
					if errors.Is(err, TTLContextCanceledErr) {
						return
					}

					c.retryRegister(cc.ctx, serviceId, asr)
				}
			}
		}
	}()
}

//#endregion

//#region Deregister

// Deregister will cancel the service registration and wait until the context is done.
// If the deregister call failed with a 404 status code, it will be ignored.
// The function will return an error if the deregister call failed with other status codes.
func (c *Client) Deregister(ctx context.Context, serviceId string) error {
	c.lock.RLock() //+R lock
	cc, ok := c.cancelers[serviceId]
	c.lock.RUnlock() //-R lock

	if ok {
		cc.cancel()
		<-cc.done
	}

	var (
		err error
		se  api.StatusError
	)

	err = c.client.Agent().ServiceDeregisterOpts(serviceId, new(api.QueryOptions).WithContext(ctx))
	if errors.As(err, &se) && se.Code == DeregisterNotFoundCode {
		err = nil
	}
	return err
}

//#endregion

// #region Helpers

// sleepCtx sleeps until the context is done or the timer expires.
// It returns an error if the context is done before the timer expires.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

//#endregion
