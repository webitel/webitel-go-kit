# Webitel Discovery Service SDK

[![Go Version](https://img.shields.io/badge/go-1.25%2B-blue.svg)](https://go.dev/)
[![Coverage](https://img.shields.io/codecov/c/github/webitel/webitel-go-kit)](https://codecov.io/gh/webitel/webitel-go-kit)

A lightweight, extensible **Go SDK** for **Service Discovery** and **Distributed Configuration (KV)**.

The SDK provides a unified abstraction over multiple service discovery backends, allowing you to switch providers **without changing business logic**.
Initial implementation supports **HashiCorp Consul**, with the architecture ready for **etcd** and **Kubernetes**.

---

## ✨ Features

* **Provider-agnostic API**
  Swap Consul / etcd / Kubernetes without touching application code.

* **Reactive Watchers**
  Real-time service topology and KV updates using efficient long polling or streaming.

* **Factory-based Initialization**
  Providers are registered automatically via anonymous imports.

* **Integrated Distributed KV**
  Built-in key-value store support for dynamic configuration.

* **Type-safe Configuration**
  Functional options with generics for extensibility and safety.

* **Test-friendly Design**
  Supports unit tests and E2E tests via Testcontainers.

---

## 🧠 Architecture

The SDK follows the **Dependency Inversion Principle**.

* `infra/discovery`
  Defines core interfaces and domain models
* `infra/discovery/{provider}`
  Concrete implementations (e.g. `consul`)
* Providers self-register via `init()` and are resolved through a factory

```
Application
    ↓
Discovery Interfaces (infra/discovery)
    ↓
Provider Factory
    ↓
Consul / etcd / Kubernetes
```

---

## 📦 Installation

```bash
go get github.com/webitel/webitel-go-kit/infra/discovery
```

---

## 🚀 Quick Start

### 1. Register a Service

```go
import (
    "context"
    "time"

    "github.com/webitel/webitel-go-kit/infra/discovery"
    _ "github.com/webitel/webitel-go-kit/infra/discovery/consul"
)

func main() {
    ctx := context.Background()

    provider, _ := discovery.DefaultFactory.CreateProvider(
        discovery.ProviderConsul,
        logger,
        "127.0.0.1:8500",
        discovery.WithTimeout[discovery.DiscoveryProvider](5*time.Second),
    )

    svc := &discovery.ServiceInstance{
        Id:   "order-api-1",
        Name: "order-service",
        Endpoints: []string{
            "http://10.0.0.5:8080",
        },
    }

    provider.Register(ctx, svc)
    defer provider.Deregister(ctx, svc)
}
```

---

### 2. Watch Service Topology Changes

```go
watcher, _ := provider.GetWatcher(ctx, "payment-service")
defer watcher.Stop()

for {
    instances, err := watcher.Next()
    if err != nil {
        break
    }

    fmt.Printf("Active instances: %d\n", len(instances))
}
```

---

### 3. Distributed Configuration (KV)

```go
if kv, ok := provider.(discovery.KVProvider); ok {
    kv.PutToKV(ctx, "config/max_retries", []byte("10"))

    kvWatcher := kv.GetKVWatcher(ctx, "config/max_retries")

    go func() {
        for {
            val, _ := kvWatcher.Next()
            fmt.Printf("Config updated: %s\n", val)
        }
    }()
}
```

---

## 🧪 Testing

The SDK is designed with testability as a first-class concern.

### Unit Tests

* Fake HTTP servers
* No external dependencies

### End-to-End Tests

* Real Consul instance via **Testcontainers**

Run E2E tests (Docker required):

```bash
go test -v ./infra/discovery/e2e/...
```
