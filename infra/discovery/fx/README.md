# discoveryfx

Підключення service discovery до fx-застосунку. Модуль створює Consul-провайдер,
реєструє інстанс сервісу на старті й знімає реєстрацію на зупинці. Надає
`discovery.DiscoveryProvider` — його можна брати з графа для `GetService`, `KV`
тощо.

У графі має бути `*slog.Logger`.

## Використання

```go
import (
    discoveryfx "github.com/webitel/webitel-go-kit/infra/discovery/fx"
    healthfx "github.com/webitel/webitel-go-kit/infra/health/fx"
)

fx.New(
    healthfx.Module(healthfx.Config{HTTPAddr: cfg.Health.Addr}),
    discoveryfx.Module(discoveryfx.Config{
        ConsulAddr: cfg.Consul.Addr,
        Name:       model.ServiceName,
        Version:    model.Version,
        Endpoints:  []string{"grpc://" + cfg.Service.Addr},
        Metadata: map[string]string{
            "commit":         model.Commit,
            "commitDate":     model.CommitDate,
            "branch":         model.Branch,
            "buildTimestamp": model.BuildTimestamp,
        },
    }),
    // ... сервери й залежності ...
    healthfx.Shutdown(),
)
```

`Name` обов'язковий: з нього будується ID інстансу (`discovery.GenerateInstanceID`).
Перший ендпоінт — адреса, яку перевіряє Consul. `Timeout` для викликів Consul
за замовчуванням `DefaultTimeout` (30 с). Окремий `fx.Invoke` для побудови
провайдера не потрібен — модуль робить це сам.

## Readiness

Якщо в графі є `*health.Registry` (наприклад, з `healthfx.Module`), TTL-heartbeat
передає в Consul його вердикт через `consul.WithReadiness(h.ReadyFunc())`: поки
вузол не готовий, check у стані `critical`. Без реєстру heartbeat завжди
передає `passing`.

## Порядок зупинки

`healthfx.Shutdown()` має стояти на кореневому рівні `fx.New`, останнім. Тоді на
зупинці спершу відбувається drain (`DrainHold`, Consul бачить `critical`), а вже
потім `Deregister`. Позиція `discoveryfx.Module` відносно `Shutdown()` на це не
впливає. Не загортати `healthfx.Shutdown()` у власний `fx.Module` — тоді
`Deregister` виконається до drain.

## Перевірка

```bash
cd infra/discovery/fx
go test -race ./...
```
