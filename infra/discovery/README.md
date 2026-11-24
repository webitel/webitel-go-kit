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

### 4. Consul cluster: CAS, sessions, node-meta (Consul only)

`consul.Cluster` is a handle to one Consul cluster reachable through several servers.

Failover:
* A call that fails on one server (network error, 5xx) moves on to the next; the server that answered becomes current.
* A 4xx is an answer and is returned as is. So is a lock call with a session that is gone, which Consul answers with 500 `invalid session`: it returns `ErrSessionNotFound`.
* A write that is not safe to repeat (`KVCAS`, `KVDeleteCAS`, `Release`) is not sent to another server once its request has been written: the first server may have applied it and lost the response, and the retry would report a lost race. The error matches `ErrOutcomeUnknown`; re-read and decide. Reads, `Acquire`, and the calls on sessions fail over.
* A call that fails everywhere returns the `*EndpointError` of every attempt, joined, so the caller can classify each.

Like the rest of the package, `NewCluster` uses `api.DefaultConfig`, so `CONSUL_HTTP_TOKEN(_FILE)`, `CONSUL_NAMESPACE` and `CONSUL_PARTITION` apply.

```go
cluster, _ := consul.NewCluster([]string{"10.0.0.1:8500", "10.0.0.2:8500"},
    consul.WithRequestTimeout(3*time.Second))

// KV with ModifyIndex and compare-and-swap; index 0 means "create only".
e, err := cluster.KVGet(ctx, "state")          // consul.ErrKeyNotFound when missing
ok, err := cluster.KVCAS(ctx, "state", next, e.ModifyIndex)
list, err := cluster.KVList(ctx, "journal/")
ok, err = cluster.KVDeleteCAS(ctx, "journal/1", list[0].ModifyIndex)

// Session without health checks (not even serfHealth); TTL is required.
sid, _ := cluster.CreateSession(ctx, consul.SessionConfig{Name: "op", TTL: 15*time.Second, LockDelay: 15*time.Second})
got, _ := cluster.Acquire(ctx, "lock", []byte("owner"), sid)
_ = cluster.RenewSession(ctx, sid)             // consul.ErrSessionNotFound when expired
_, _ = cluster.Release(ctx, "lock", []byte("owner"), sid)

// Passing instances with a tag on nodes with given node_meta, and a prepared query.
nodes, _ := cluster.ServiceNodes(ctx, "api", consul.ServiceQuery{PassingOnly: true, Tag: "http", NodeMeta: map[string]string{"dc": "b"}})
nodes, _ = cluster.ExecuteQuery(ctx, "webitel-api")
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
