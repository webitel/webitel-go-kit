package consul

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/webitel/webitel-go-kit/infra/discovery"
)

type mockConsulClient struct {
	mock.Mock
}

func (m *mockConsulClient) Agent() *api.Agent {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*api.Agent)
}

type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) NodeName() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// Helper functions

func createTestRegistry(t *testing.T) *Registry {
	logger := new(mockLogger)
	logger.On("Debug", mock.Anything, mock.Anything).Maybe()
	logger.On("Info", mock.Anything, mock.Anything).Maybe()
	logger.On("Warn", mock.Anything, mock.Anything).Maybe()
	logger.On("Error", mock.Anything, mock.Anything).Maybe()

	client := &Client{
		dc:                             SingleDatacenter,
		healthCheckInterval:            10,
		heartbeat:                      true,
		deregisterCriticalServiceAfter: 600,
		resolver:                       defaultResolver,
		cancelers:                      make(map[string]*canceler),
		logger:                         logger,
	}

	registry := &Registry{
		registry:          make(map[string]*serviceSet),
		enableHealthCheck: true,
		timeout:           10 * time.Second,
		client:            client,
	}

	return registry
}

func createTestServiceInstance(id, name, address string, port int) *discovery.ServiceInstance {
	return &discovery.ServiceInstance{
		Id:   id,
		Name: name,
		Metadata: map[string]string{
			"test": "value",
		},
	}
}

func TestRegistry_SetHealthCheck(t *testing.T) {
	registry := createTestRegistry(t)

	tests := []struct {
		name  string
		value bool
	}{
		{"Enable health check", true},
		{"Disable health check", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry.SetHealthCheck(tt.value)
			assert.Equal(t, tt.value, registry.enableHealthCheck)
		})
	}
}

func TestRegistry_SetTimeout(t *testing.T) {
	registry := createTestRegistry(t)

	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"Set 5 seconds", 5 * time.Second},
		{"Set 30 seconds", 30 * time.Second},
		{"Set zero timeout", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry.SetTimeout(tt.timeout)
			assert.Equal(t, tt.timeout, registry.timeout)
		})
	}
}

func TestRegistry_SetDatacenter(t *testing.T) {
	registry := createTestRegistry(t)

	tests := []struct {
		name     string
		dc       string
		expected Datacenter
	}{
		{"Set single datacenter", "SINGLE", SingleDatacenter},
		{"Set multi datacenter", "MULTI", MultiDataCenter},
		{"Invalid datacenter keeps current", "INVALID", SingleDatacenter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry.SetDatacenter(tt.dc)
			assert.Equal(t, tt.expected, registry.client.dc)
		})
	}
}

func TestRegistry_SetDatacenter_NilClient(t *testing.T) {
	registry := &Registry{
		client: nil,
	}

	assert.NotPanics(t, func() {
		registry.SetDatacenter("SINGLE")
	})
}

func TestRegistry_SetHeartbeatEnabled(t *testing.T) {
	registry := createTestRegistry(t)

	tests := []struct {
		name    string
		enabled bool
	}{
		{"Enable heartbeat", true},
		{"Disable heartbeat", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry.SetHeartbeatEnabled(tt.enabled)
			assert.Equal(t, tt.enabled, registry.client.heartbeat)
		})
	}
}

func TestRegistry_SetHeartbeatEnabled_NilClient(t *testing.T) {
	registry := &Registry{
		client: nil,
	}

	assert.NotPanics(t, func() {
		registry.SetHeartbeatEnabled(true)
	})
}

func TestRegistry_SetHealthCheckInterval(t *testing.T) {
	registry := createTestRegistry(t)

	tests := []struct {
		name     string
		interval int
	}{
		{"Set 15 seconds", 15},
		{"Set 0 seconds", 0},
		{"Set negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry.SetHealthCheckInterval(tt.interval)
			assert.Equal(t, tt.interval, registry.client.healthCheckInterval)
		})
	}
}

func TestRegistry_SetDeregisterCriticalServiceAfter(t *testing.T) {
	registry := createTestRegistry(t)

	tests := []struct {
		name     string
		interval int
	}{
		{"Set 300 seconds", 300},
		{"Set 0 seconds", 0},
		{"Set negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry.SetDeregisterCriticalServiceAfter(tt.interval)
			assert.Equal(t, tt.interval, registry.client.deregisterCriticalServiceAfter)
		})
	}
}

func TestRegistry_SetTags(t *testing.T) {
	registry := createTestRegistry(t)

	tests := []struct {
		name string
		tags []string
	}{
		{"Set single tag", []string{"production"}},
		{"Set multiple tags", []string{"production", "v1.0", "api"}},
		{"Set empty tags", []string{}},
		{"Set nil tags", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry.SetTags(tt.tags...)
			assert.Equal(t, tt.tags, registry.client.tags)
		})
	}
}

func TestRegistry_GetService_FromCache(t *testing.T) {
	registry := createTestRegistry(t)
	ctx := context.Background()

	services := []*discovery.ServiceInstance{
		createTestServiceInstance("test-1", "test-service", "localhost", 8080),
	}

	set := &serviceSet{
		serviceName: "test-service",
		services:    new(atomic.Value),
	}
	set.services.Store(services)

	registry.lock.Lock()
	registry.registry["test-service"] = set
	registry.lock.Unlock()

	result, err := registry.GetService(ctx, "test-service")
	assert.NoError(t, err)
	assert.Equal(t, services, result)
}

func TestRegistry_GetWatcher_ExistingService(t *testing.T) {
	registry := createTestRegistry(t)
	ctx := context.Background()

	services := []*discovery.ServiceInstance{
		createTestServiceInstance("test-1", "test-service", "localhost", 8080),
	}

	set := &serviceSet{
		registry:    registry,
		serviceName: "test-service",
		services:    new(atomic.Value),
		watcher:     make(map[*watcher]struct{}),
	}
	set.services.Store(services)
	set.ctx, set.cancel = context.WithCancel(context.Background())
	defer set.cancel()

	registry.lock.Lock()
	registry.registry["test-service"] = set
	registry.lock.Unlock()

	watcher, err := registry.GetWatcher(ctx, "test-service")
	require.NoError(t, err)
	require.NotNil(t, watcher)

	assert.Equal(t, int32(1), set.ref.Load())
}

func TestRegistry_GetWatcher_ContextCanceled(t *testing.T) {
	registry := createTestRegistry(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	watcher, err := registry.GetWatcher(ctx, "test-service")
	assert.Error(t, err)
	assert.Nil(t, watcher)
}

func TestRegistry_ListServices_Empty(t *testing.T) {
	registry := createTestRegistry(t)

	result := registry.ListServices()
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestRegistry_ListServices_WithServices(t *testing.T) {
	registry := createTestRegistry(t)

	services1 := []*discovery.ServiceInstance{
		createTestServiceInstance("test-1", "service1", "localhost", 8080),
	}
	services2 := []*discovery.ServiceInstance{
		createTestServiceInstance("test-2", "service2", "localhost", 8081),
		createTestServiceInstance("test-3", "service2", "localhost", 8082),
	}

	set1 := &serviceSet{
		serviceName: "service1",
		services:    new(atomic.Value),
	}
	set1.services.Store(services1)

	set2 := &serviceSet{
		serviceName: "service2",
		services:    new(atomic.Value),
	}
	set2.services.Store(services2)

	registry.lock.Lock()
	registry.registry["service1"] = set1
	registry.registry["service2"] = set2
	registry.lock.Unlock()

	result := registry.ListServices()
	assert.Len(t, result, 2)
	assert.Len(t, result["service1"], 1)
	assert.Len(t, result["service2"], 2)
}

func TestRegistry_ListServices_IgnoresNilServices(t *testing.T) {
	registry := createTestRegistry(t)

	set1 := &serviceSet{
		serviceName: "service1",
		services:    new(atomic.Value),
	}

	registry.lock.Lock()
	registry.registry["service1"] = set1
	registry.lock.Unlock()

	result := registry.ListServices()
	assert.Empty(t, result)
}

func TestRegistry_ConcurrentGetService(t *testing.T) {
	registry := createTestRegistry(t)

	services := []*discovery.ServiceInstance{
		createTestServiceInstance("test-1", "test-service", "localhost", 8080),
	}

	set := &serviceSet{
		serviceName: "test-service",
		services:    new(atomic.Value),
	}
	set.services.Store(services)

	registry.lock.Lock()
	registry.registry["test-service"] = set
	registry.lock.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := registry.GetService(context.Background(), "test-service")
			assert.NoError(t, err)
			assert.NotNil(t, result)
		}()
	}

	wg.Wait()
}

func TestRegistry_ConcurrentSetters(t *testing.T) {
	registry := createTestRegistry(t)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			registry.SetHealthCheck(i%2 == 0)
			registry.SetTimeout(time.Duration(i) * time.Second)
			registry.SetHeartbeatEnabled(i%2 == 1)
			registry.SetHealthCheckInterval(i)
			registry.SetDeregisterCriticalServiceAfter(i * 10)
			registry.SetTags("tag1", "tag2")
		}(i)
	}

	wg.Wait()
}

func TestRegistry_ConcurrentListServices(t *testing.T) {
	registry := createTestRegistry(t)

	services := []*discovery.ServiceInstance{
		createTestServiceInstance("test-1", "test-service", "localhost", 8080),
	}

	set := &serviceSet{
		serviceName: "test-service",
		services:    new(atomic.Value),
	}
	set.services.Store(services)

	registry.lock.Lock()
	registry.registry["test-service"] = set
	registry.lock.Unlock()

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			result := registry.ListServices()
			assert.NotNil(t, result)
		})
	}

	wg.Wait()
}
