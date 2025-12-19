package discovery

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"github.com/webitel/webitel-go-kit/infra/discovery"
// 	// Import the consul package to register the provider in the DefaultFactory
// 	_ "github.com/webitel/webitel-go-kit/infra/discovery/consul"
// )

// func main() {
// 	// Root context that reacts to system signals for graceful shutdown
// 	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
// 	defer stop()

// 	logger := &simpleLogger{}
// 	consulAddr := "127.0.0.1:8500" // Should be passed via env in real scenarios

// 	// 1. Initialize the Discovery Provider via the Factory.
// 	// This approach follows the Dependency Inversion Principle,
// 	// allowing easy replacement of the discovery backend.
// 	provider, err := discovery.DefaultFactory.CreateProvider(
// 		discovery.ProviderConsul,
// 		logger,
// 		consulAddr,
// 		discovery.WithTimeout[discovery.DiscoveryProvider](3*time.Second),
// 		discovery.WithHealthCheck[discovery.DiscoveryProvider](true),
// 	)
// 	if err != nil {
// 		log.Fatalf("failed to create discovery provider: %v", err)
// 	}

// 	// 2. Define the current service instance metadata
// 	service := &discovery.ServiceInstance{
// 		Id:        "order-api-1",
// 		Name:      "order-service",
// 		Version:   "1.0.0",
// 		Endpoints: []string{"http://192.168.1.10:8080"},
// 		Metadata: map[string]string{
// 			"env": "production",
// 		},
// 	}

// 	// 3. Register the service in the registry
// 	if err := provider.Register(ctx, service); err != nil {
// 		log.Fatalf("failed to register service: %v", err)
// 	}
// 	fmt.Printf("Service %s registered at %s\n", service.Name, consulAddr)

// 	// 4. Demonstrate Key-Value Store usage (Dynamic Configuration)
// 	// We cast the provider to KVProvider interface to access KV features
// 	kv := provider.KV()
//  configKey := "config/order-service/maintenance-mode"

// // Set initial value
// 	_ = kv.PutToKV(ctx, configKey, []byte("false"))
// 	fmt.Println("Initial configuration set in KV")

// 		// Start a watcher in a separate goroutine to react to config changes
// 	go watchConfiguration(ctx, kv, configKey)
// 	}

// 	// 5. Discover other services (e.g., finding 'payment-service')
// 	go func() {
// 		ticker := time.NewTicker(10 * time.Second)
// 		for {
// 			select {
// 			case <-ticker.C:
// 				instances, err := provider.GetService(ctx, "payment-service")
// 				if err != nil {
// 					fmt.Printf("! Discovery error: %v\n", err)
// 					continue
// 				}
// 				fmt.Printf("Discovered %d instances of payment-service\n", len(instances))
// 			case <-ctx.Done():
// 				return
// 			}
// 		}
// 	}()

// 	// Start monitoring a dependency service in the background
// 	go watchRemoteService(ctx, provider, "payment-service")

// 	// Wait for termination signal
// 	fmt.Println("Service is running. Press CTRL+C to stop.")
// 	<-ctx.Done()

// 	// 6. Graceful Shutdown: Deregister the service
// 	fmt.Println("\nShutting down...")
// 	if err := provider.Deregister(context.Background(), service); err != nil {
// 		fmt.Printf("! Deregistration failed: %v\n", err)
// 	} else {
// 		fmt.Println("Service deregistered. Goodbye!")
// 	}
// }

// // watchConfiguration demonstrates how to handle real-time updates using Watchers.
// func watchConfiguration(ctx context.Context, kv discovery.KVProvider, key string) {
// 	watcher := kv.GetKVWatcher(ctx, key)
// 	defer watcher.Stop()

// 	fmt.Printf("Watching for changes on: %s\n", key)
// 	for {
// 		val, err := watcher.Next()
// 		if err != nil {
// 			if ctx.Err() != nil {
// 				return // Context canceled
// 			}
// 			fmt.Printf("! Watcher error: %v\n", err)
// 			time.Sleep(time.Second)
// 			continue
// 		}
// 		fmt.Printf("CONFIG UPDATE: %s = %s\n", key, string(val))
// 	}
// }

// // Example of watching for service topography changes (Service Discovery Watcher)
// func watchRemoteService(ctx context.Context, provider discovery.DiscoveryProvider, serviceName string) {
// 	// 1. Get a watcher for the target service.
// 	// This watcher will receive updates whenever nodes are added, removed, or changed.
// 	watcher, err := provider.GetWatcher(ctx, serviceName)
// 	if err != nil {
// 		log.Printf("[ERROR] failed to create watcher for %s: %v", serviceName, err)
// 		return
// 	}
// 	defer watcher.Stop()

// 	fmt.Printf("Monitoring topography changes for: %s\n", serviceName)

// 	for {
// 		// 2. Next() blocks until the service instances change in Consul.
// 		// It returns the full updated list of healthy instances.
// 		instances, err := watcher.Next()
// 		if err != nil {
// 			if ctx.Err() != nil {
// 				return // Context canceled (graceful shutdown)
// 			}
// 			log.Printf("[ERROR] service watcher error: %v", err)
// 			time.Sleep(2 * time.Second) // Backoff before retry
// 			continue
// 		}

// 		// 3. Logic to update local load balancer or connection pool
// 		fmt.Printf("TOPOLOGY UPDATE [%s]: %d nodes active\n", serviceName, len(instances))
// 		for _, inst := range instances {
// 			fmt.Printf("  - Node: %s, Address: %v, Version: %s\n", inst.Id, inst.Endpoints, inst.Version)
// 		}
// 	}
// }

// // simpleLogger implements discovery.Logger interface for demonstration purposes.
// type simpleLogger struct{}

// func (l *simpleLogger) Info(msg string, args ...any)  { log.Printf("[INFO] "+msg, args...) }
// func (l *simpleLogger) Warn(msg string, args ...any)  { log.Printf("[WARN] "+msg, args...) }
// func (l *simpleLogger) Error(msg string, err error, args ...any) { log.Printf("[ERROR] %v: "+msg, err) }
