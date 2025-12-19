//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	_ "github.com/webitel/webitel-go-kit/infra/discovery/consul"
	"go.uber.org/goleak"
)

type testLogger struct {
	t *testing.T
}

func (l *testLogger) Info(msg string, args ...any) { l.t.Logf("[INFO] "+msg, args...) }
func (l *testLogger) Warn(msg string, args ...any) { l.t.Logf("[WARN] "+msg, args...) }
func (l *testLogger) Error(msg string, err error, args ...any) {
	l.t.Logf("[ERROR] %v: "+msg, err)
}

// TestConsulE2E tests the Consul discovery provider in an end-to-end scenario.
// It registers a service, gets the service after registration, lists all services,
// watches service updates, and deregisters the service.
func TestConsulE2E(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})
	ctx := context.Background()

	consulContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "hashicorp/consul:1.15",
			ExposedPorts: []string{"8500/tcp"},
			WaitingFor:   wait.ForHTTP("/v1/status/leader").WithPort("8500/tcp"),
			Cmd:          []string{"agent", "-dev", "-client", "0.0.0.0"},
		},
		Started: true,
	})
	require.NoError(t, err)
	defer consulContainer.Terminate(ctx)

	host, err := consulContainer.Host(ctx)
	require.NoError(t, err)
	mappedPort, err := consulContainer.MappedPort(ctx, "8500")
	require.NoError(t, err)

	endpoint := fmt.Sprintf("%s:%s", host, mappedPort.Port())
	t.Logf("Consul endpoint: %s", endpoint)
	require.NoError(t, err)

	logger := &testLogger{t: t}
	provider, err := discovery.DefaultFactory.CreateProvider(
		discovery.ProviderConsul,
		logger,
		endpoint,
		discovery.WithHealthCheck[discovery.DiscoveryProvider](false),
		discovery.WithTimeout[discovery.DiscoveryProvider](2*time.Second),
	)
	require.NoError(t, err)

	svc := &discovery.ServiceInstance{
		Id:        "node-1",
		Name:      "order-service",
		Version:   "v1.0.0",
		Endpoints: []string{"http://127.0.0.1:8080"},
	}

	t.Run("Register service", func(t *testing.T) {
		err := provider.Register(ctx, svc)
		assert.NoError(t, err)
	})

	t.Run("Get service after registration", func(t *testing.T) {
		assert.Eventually(t, func() bool {
			instances, err := provider.GetService(ctx, svc.Name)
			return err == nil && len(instances) > 0 && instances[0].Id == svc.Id
		}, 5*time.Second, 500*time.Millisecond)
	})

	t.Run("List all services", func(t *testing.T) {
		watcher, err := provider.GetWatcher(ctx, svc.Name)
		require.NoError(t, err)
		defer watcher.Stop()

		all := provider.ListServices()
		assert.Contains(t, all, svc.Name)
		assert.Equal(t, svc.Id, all[svc.Name][0].Id)
	})

	t.Run("Watch service updates", func(t *testing.T) {
		watcher, err := provider.GetWatcher(ctx, "order-service-1")
		require.NoError(t, err)
		defer watcher.Stop()

		svc2 := &discovery.ServiceInstance{
			Id:   "node-2",
			Name: "order-service-1",
		}

		go func() {
			time.Sleep(1 * time.Second)
			_ = provider.Register(ctx, svc2)
		}()

		instances, err := watcher.Next()
		assert.NoError(t, err)

		assert.NotEmpty(t, instances)

		provider.Deregister(ctx, svc2)
	})

	t.Run("KV Operations Flow", func(t *testing.T) {
		testKey := "configs/payment-gate/timeout"
		testValue := []byte("30s")

		parsedProvider := provider.KV()
		err := parsedProvider.PutToKV(ctx, testKey, testValue)
		assert.NoError(t, err, "Failed to put value to KV")

		val, err := parsedProvider.GetFromKV(ctx, testKey)
		assert.NoError(t, err, "Failed to get value from KV")
		assert.Equal(t, testValue, val)

		newValue := []byte("60s")
		err = parsedProvider.PutToKV(ctx, testKey, newValue)
		assert.NoError(t, err)

		val, err = parsedProvider.GetFromKV(ctx, testKey)
		assert.NoError(t, err)
		assert.Equal(t, newValue, val)

		err = parsedProvider.DeleteFromKV(ctx, testKey)
		assert.NoError(t, err)

		_, err = parsedProvider.GetFromKV(ctx, testKey)
		assert.Error(t, err, "Key should be deleted and return error on Get")
	})

	t.Run("KV Watcher updates", func(t *testing.T) {
		parsedProvider := provider.KV()
		watchKey := "dynamic/feature-flag"
		initialValue := []byte("false")
		updatedValue := []byte("true")

		err := parsedProvider.PutToKV(ctx, watchKey, initialValue)
		require.NoError(t, err)

		kvWatcher := parsedProvider.GetKVWatcher(ctx, watchKey)
		defer kvWatcher.Stop()

		go func() {
			time.Sleep(1 * time.Second)
			_ = parsedProvider.PutToKV(ctx, watchKey, updatedValue)
		}()

		val, err := kvWatcher.Next()
		assert.NoError(t, err)
		assert.Equal(t, updatedValue, val)
	})

	t.Run("Deregister service", func(t *testing.T) {
		err := provider.Deregister(ctx, svc)
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			instances, _ := provider.GetService(ctx, svc.Name)
			for _, i := range instances {
				if i.Id == svc.Id {
					return false
				}
			}
			return true
		}, 5*time.Second, 500*time.Millisecond)
	})
}
