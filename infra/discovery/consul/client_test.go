package consul

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/webitel/webitel-go-kit/infra/discovery"
)

type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Info(format string, args ...any) {
	m.Called(format, args)
}

func (m *mockLogger) Warn(format string, args ...any) {
	m.Called(format, args)
}

func (m *mockLogger) Error(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *mockLogger) Debug(format string, args ...any) {
	m.Called(format, args)
}

func newTestClient() *Client {
	return &Client{
		dc:                             SingleDatacenter,
		resolver:                       defaultResolver,
		healthCheckInterval:            10,
		heartbeat:                      false,
		deregisterCriticalServiceAfter: 30,
		serviceChecks:                  api.AgentServiceChecks{},
		tags:                           []string{},
		logger:                         &mockLogger{},
		cancelers:                      make(map[string]*canceler),
	}
}

func TestVersionBuilder(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "version tag present",
			tags:     []string{"env=prod", "version=1.0.0", "region=us"},
			expected: "1.0.0",
		},
		{
			name:     "no version tag",
			tags:     []string{"env=prod", "region=us"},
			expected: "",
		},
		{
			name:     "empty tags",
			tags:     []string{},
			expected: "",
		},
		{
			name:     "version tag with equals in value",
			tags:     []string{"version=1.0.0=beta"},
			expected: "1.0.0=beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := versionBuilder(tt.tags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGrpcEndpointsBuilder(t *testing.T) {
	tests := []struct {
		name             string
		taggedAddresses  map[string]api.ServiceAddress
		address          string
		port             int
		expectedCount    int
		expectedContains string
	}{
		{
			name: "tagged addresses with custom scheme",
			taggedAddresses: map[string]api.ServiceAddress{
				"grpc": {Address: "grpc://localhost:8080", Port: 8080},
				"http": {Address: "http://localhost:8081", Port: 8081},
			},
			address:          "localhost",
			port:             9090,
			expectedCount:    2,
			expectedContains: "grpc://localhost:8080",
		},
		{
			name:             "no tagged addresses, use default",
			taggedAddresses:  map[string]api.ServiceAddress{},
			address:          "localhost",
			port:             9090,
			expectedCount:    1,
			expectedContains: "grpc://localhost:9090",
		},
		{
			name: "skip lan_ipv4 and wan_ipv4",
			taggedAddresses: map[string]api.ServiceAddress{
				"lan_ipv4": {Address: "10.0.0.1:8080", Port: 8080},
				"wan_ipv4": {Address: "20.0.0.1:8080", Port: 8080},
				"grpc":     {Address: "grpc://localhost:8080", Port: 8080},
			},
			address:          "localhost",
			port:             9090,
			expectedCount:    1,
			expectedContains: "grpc://localhost:8080",
		},
		{
			name:            "empty addresses and no default",
			taggedAddresses: map[string]api.ServiceAddress{},
			address:         "",
			port:            0,
			expectedCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grpcEndpointsBuilder(tt.taggedAddresses, tt.address, tt.port)
			assert.Len(t, result, tt.expectedCount)
			if tt.expectedContains != "" {
				assert.Contains(t, result, tt.expectedContains)
			}
		})
	}
}

func TestDefaultResolver(t *testing.T) {
	entries := []*api.ServiceEntry{
		{
			Service: &api.AgentService{
				ID:      "service-1",
				Service: "test-service",
				Tags:    []string{"version=1.0.0"},
				Meta:    map[string]string{"env": "prod"},
				Address: "localhost",
				Port:    8080,
				TaggedAddresses: map[string]api.ServiceAddress{
					"grpc": {Address: "grpc://localhost:8080", Port: 8080},
				},
			},
		},
		{
			Service: &api.AgentService{
				ID:              "service-2",
				Service:         "test-service",
				Tags:            []string{"version=2.0.0"},
				Meta:            map[string]string{"env": "dev"},
				Address:         "localhost",
				Port:            8081,
				TaggedAddresses: map[string]api.ServiceAddress{},
			},
		},
	}

	ctx := context.Background()
	result := defaultResolver(ctx, entries)

	assert.Len(t, result, 2)
	assert.Equal(t, "service-1", result[0].Id)
	assert.Equal(t, "test-service", result[0].Name)
	assert.Equal(t, "1.0.0", result[0].Version)
	assert.Equal(t, map[string]string{"env": "prod"}, result[0].Metadata)
	assert.Contains(t, result[0].Endpoints, "grpc://localhost:8080")

	assert.Equal(t, "service-2", result[1].Id)
	assert.Equal(t, "2.0.0", result[1].Version)
	assert.Contains(t, result[1].Endpoints, "grpc://localhost:8081")
}

func TestBuildRegistration(t *testing.T) {
	client := newTestClient()
	client.tags = []string{"env=prod"}

	svc := &discovery.ServiceInstance{
		Id:       "test-service-1",
		Name:     "test-service",
		Version:  "1.0.0",
		Metadata: map[string]string{"key": "value"},
		Endpoints: []string{
			"grpc://localhost:8080",
			"http://localhost:8081",
		},
	}

	asr, checkAddresses, err := client.buildRegistration(svc)

	require.NoError(t, err)
	assert.NotNil(t, asr)
	assert.Equal(t, "test-service-1", asr.ID)
	assert.Equal(t, "test-service", asr.Name)
	assert.Equal(t, map[string]string{"key": "value"}, asr.Meta)
	assert.Len(t, checkAddresses, 2)
	assert.Contains(t, checkAddresses, "localhost:8080")
	assert.Contains(t, checkAddresses, "localhost:8081")
	assert.Equal(t, "localhost", asr.Address)
	assert.Equal(t, 8080, asr.Port)
	assert.Len(t, asr.TaggedAddresses, 2)
}

func TestBuildRegistrationInvalidURL(t *testing.T) {
	client := newTestClient()

	svc := &discovery.ServiceInstance{
		Id:        "test-service-1",
		Name:      "test-service",
		Endpoints: []string{"://invalid-url"},
	}

	_, _, err := client.buildRegistration(svc)
	assert.Error(t, err)
}

func TestTcpCheck(t *testing.T) {
	client := newTestClient()
	client.healthCheckInterval = 15
	client.deregisterCriticalServiceAfter = 60

	check := client.tcpCheck("localhost:8080")

	assert.NotNil(t, check)
	assert.Equal(t, "localhost:8080", check.TCP)
	assert.Equal(t, "15s", check.Interval)
	assert.Equal(t, "60s", check.DeregisterCriticalServiceAfter)
	assert.Equal(t, "5s", check.Timeout)
}

func TestTtlCheck(t *testing.T) {
	client := newTestClient()
	client.healthCheckInterval = 15
	client.deregisterCriticalServiceAfter = 60

	svc := &discovery.ServiceInstance{
		Id:   "test-service-1",
		Name: "test-service",
	}

	check := client.ttlCheck(svc)

	assert.NotNil(t, check)
	assert.Equal(t, "service:test-service-1", check.CheckID)
	assert.Equal(t, "15s", check.TTL)
	assert.Equal(t, "60s", check.DeregisterCriticalServiceAfter)
}

func TestApplyChecks(t *testing.T) {
	tests := []struct {
		name                string
		enableHealthChecks  bool
		heartbeat           bool
		addresses           []string
		expectedChecksCount int
	}{
		{
			name:                "health checks enabled, no heartbeat",
			enableHealthChecks:  true,
			heartbeat:           false,
			addresses:           []string{"localhost:8080", "localhost:8081"},
			expectedChecksCount: 2,
		},
		{
			name:                "health checks enabled, with heartbeat",
			enableHealthChecks:  true,
			heartbeat:           true,
			addresses:           []string{"localhost:8080"},
			expectedChecksCount: 2,
		},
		{
			name:                "no health checks, with heartbeat",
			enableHealthChecks:  false,
			heartbeat:           true,
			addresses:           []string{"localhost:8080"},
			expectedChecksCount: 1,
		},
		{
			name:                "no checks at all",
			enableHealthChecks:  false,
			heartbeat:           false,
			addresses:           []string{"localhost:8080"},
			expectedChecksCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient()
			client.heartbeat = tt.heartbeat

			asr := &api.AgentServiceRegistration{}
			svc := &discovery.ServiceInstance{
				Id:   "test-service-1",
				Name: "test-service",
			}

			client.applyChecks(asr, svc, tt.addresses, tt.enableHealthChecks)

			assert.Len(t, asr.Checks, tt.expectedChecksCount)
		})
	}
}

func TestPrepareHeartbeatWithoutHeartbeat(t *testing.T) {
	client := newTestClient()
	client.heartbeat = false

	cc := client.prepareHeartbeat("service-1")
	assert.Nil(t, cc)
}

func TestSleepCtx(t *testing.T) {
	t.Run("sleep completes", func(t *testing.T) {
		ctx := context.Background()
		start := time.Now()
		err := sleepCtx(ctx, 50*time.Millisecond)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, duration, 50*time.Millisecond)
	})

	t.Run("context canceled before sleep", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := sleepCtx(ctx, 1*time.Second)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("context canceled during sleep", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		err := sleepCtx(ctx, 1*time.Second)
		duration := time.Since(start)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Less(t, duration, 200*time.Millisecond)
	})
}

func TestServiceInstanceBuildRegistrationWithEmptyEndpoints(t *testing.T) {
	client := newTestClient()

	svc := &discovery.ServiceInstance{
		Id:        "test-service-1",
		Name:      "test-service",
		Version:   "1.0.0",
		Endpoints: []string{},
	}

	asr, checkAddresses, err := client.buildRegistration(svc)

	require.NoError(t, err)
	assert.NotNil(t, asr)
	assert.Empty(t, checkAddresses)
	assert.Empty(t, asr.TaggedAddresses)
	assert.Equal(t, "", asr.Address)
	assert.Equal(t, 0, asr.Port)
}

func TestApplyChecksWithCustomChecks(t *testing.T) {
	client := newTestClient()
	client.serviceChecks = api.AgentServiceChecks{
		&api.AgentServiceCheck{
			HTTP:     "http://localhost:8080/health",
			Interval: "30s",
		},
	}

	asr := &api.AgentServiceRegistration{}
	svc := &discovery.ServiceInstance{
		Id:   "test-service-1",
		Name: "test-service",
	}

	client.applyChecks(asr, svc, []string{"localhost:8080"}, true)

	// Should have 1 TCP check + 1 custom check!
	assert.Len(t, asr.Checks, 2)

	// Verify custom check is included!
	hasHTTPCheck := false
	for _, check := range asr.Checks {
		if check.HTTP != "" {
			hasHTTPCheck = true
			break
		}
	}
	assert.True(t, hasHTTPCheck, "Custom HTTP check should be included")
}

func TestBuildRegistrationTagHandling(t *testing.T) {
	client := newTestClient()
	client.tags = []string{"env=prod", "region=us"}

	svc := &discovery.ServiceInstance{
		Id:        "test-service-1",
		Name:      "test-service",
		Version:   "1.0.0",
		Endpoints: []string{"grpc://localhost:8080"},
	}

	asr, _, err := client.buildRegistration(svc)

	require.NoError(t, err)
	assert.Contains(t, asr.Tags, "env=prod")
	assert.Contains(t, asr.Tags, "region=us")
}

func TestGrpcEndpointsBuilderAllSkippedSchemes(t *testing.T) {
	taggedAddresses := map[string]api.ServiceAddress{
		"lan_ipv4": {Address: "10.0.0.1:8080", Port: 8080},
		"wan_ipv4": {Address: "20.0.0.1:8080", Port: 8080},
		"lan_ipv6": {Address: "[::1]:8080", Port: 8080},
		"wan_ipv6": {Address: "[::2]:8080", Port: 8080},
	}

	result := grpcEndpointsBuilder(taggedAddresses, "localhost", 9090)

	assert.Len(t, result, 1)
	assert.Contains(t, result, "grpc://localhost:9090")
}

func BenchmarkVersionBuilder(b *testing.B) {
	tags := []string{"env=prod", "version=1.0.0", "region=us", "team=backend"}

	for b.Loop() {
		_ = versionBuilder(tags)
	}
}

func BenchmarkDefaultResolver(b *testing.B) {
	entries := make([]*api.ServiceEntry, 100)
	for i := range 100 {
		entries[i] = &api.ServiceEntry{
			Service: &api.AgentService{
				ID:      fmt.Sprintf("service-%d", i),
				Service: "test-service",
				Tags:    []string{"version=1.0.0"},
				Address: "localhost",
				Port:    8080 + i,
			},
		}
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = defaultResolver(ctx, entries)
	}
}

func BenchmarkGrpcEndpointsBuilder(b *testing.B) {
	taggedAddresses := map[string]api.ServiceAddress{
		"grpc": {Address: "grpc://localhost:8080", Port: 8080},
		"http": {Address: "http://localhost:8081", Port: 8081},
	}

	for b.Loop() {
		_ = grpcEndpointsBuilder(taggedAddresses, "localhost", 9090)
	}
}
