# pgw

PostgreSQL connection manager for primary/standby topologies, built on top of [pgx](https://github.com/jackc/pgx).

Handles pool lifecycle, health checks, automatic reconnects, replica routing, migration verification, constraint-error mapping, and query tracing behind a single `PoolManager`.

## Installation

```
go get github.com/webitel/webitel-go-kit/infra/pgw
```

## Quick start

```go
manager, err := pgw.NewConnectionManager(ctx,
    pgw.WithMasterPoolConfig(pgw.PrimaryConfig{
        DSN:      "postgres://user:pass@localhost:5432/mydb",
        MaxConns: 20,
        MinConns: 2,
        pgw.DefaultMasterPoolConfig.HealthCheckInterval,  // embed defaults
    }),
)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

primary, err := manager.Primary()
if err != nil {
    return err // pgw.ErrUnreachable if the primary is down
}

rows, err := primary.Query(ctx, "SELECT id, name FROM users WHERE active = $1", true)
```

## Connection routing

| Method | Behaviour |
|---|---|
| `Primary()` | Returns the read-write primary. Errors with `ErrUnreachable` if disconnected. |
| `Standby()` | Returns a replica via the configured pick strategy. Errors with `ErrUnreachable` if none are healthy. |
| `StandbyPreferred()` | Returns a replica when available, falls back to primary. |

## Configuration

```go
manager, err := pgw.NewConnectionManager(ctx,
    pgw.WithApplicationName("my-service"),
    pgw.WithMasterPoolConfig(pgw.PrimaryConfig{
        DSN:                    "postgres://...",
        MaxConns:               20,
        MinConns:               2,
        HealthCheckInterval:    5 * time.Second,
        HealthCheckTimeout:     3 * time.Second,
        RetryAttempts:          5,
        RetryInterval:          5 * time.Second,
        RetryStrategy:          pgw.RetryStrategyLinear,
        RetryStrategyBaseValue: 2,
    }),
    pgw.WithReplicaPoolConfig(pgw.StandbyConfig{
        DSN:                           []string{"postgres://replica1/...", "postgres://replica2/..."},
        MaxConns:                      10,
        HealthCheckInterval:           5 * time.Second,
        HealthCheckTimeout:            3 * time.Second,
        PickStrategy:                  pgw.RandomPickStrategy,
        UnhealthyReplicaRetryInterval: 30 * time.Minute,
        RetriesBeforeUnhealthy:        5,
        RetryStrategy:                 pgw.RetryStrategyLinear,
        RetryStrategyBaseValue:        2,
    }),
)
```

`DefaultMasterPoolConfig` and `DefaultReplicaPoolConfig` are applied automatically — only override the fields you need.

## Retry strategies

| Strategy | Formula | Use case |
|---|---|---|
| `RetryStrategyLinear` | `a × x` seconds | Predictable, bounded waits |
| `RetryStrategyExponential` | `a ^ x` seconds | Back off aggressively under sustained failure |

`RetryStrategyBaseValue` is `a`; the attempt number is `x`.

## Pick strategies

| Strategy | Description |
|---|---|
| `RandomPickStrategy` | Picks a replica at random (default) |
| `LeastConnectionsPickStrategy` | Picks the replica with the fewest active connections |

Provide your own by implementing `func(*safemap.SafeMap[string, *Pool]) *Pool` and passing it as `StandbyConfig.PickStrategy`.

## Migration verification

`WithMigrationVerifier` registers a callback that runs once when the primary pool connects (and on each reconnect). If it returns an error, the pool is not marked as healthy and the connection attempt is retried.

```go
pgw.WithMigrationVerifier(func(ctx context.Context, conn *pgxpool.Conn) error {
    var version int
    err := conn.QueryRow(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&version)
    if err != nil {
        return err
    }
    if version < RequiredSchemaVersion {
        return fmt.Errorf("schema at version %d, need %d", version, RequiredSchemaVersion)
    }
    return nil
}),
```

Migration failures are wrapped with `ErrMigrationVerify`:

```go
if errors.Is(err, pgw.ErrMigrationVerify) {
    // schema is not ready
}
```

## Constraint error mapping

Register processors for specific PostgreSQL constraint violations so that raw `pgconn.PgError` values are translated to your own error types before being returned from any pool method.

```go
manager.RegisterUniqueViolation("users_email_key", func(e *pgconn.PgError) error {
    return ErrEmailAlreadyTaken
})

manager.RegisterForeignKeyViolation("orders_user_id_fkey", func(e *pgconn.PgError) error {
    return ErrUserNotFound
})

manager.RegisterCheckViolation("orders_amount_check", func(e *pgconn.PgError) error {
    return ErrInvalidAmount
})

manager.RegisterNotNullViolation("users", "email", func(e *pgconn.PgError) error {
    return ErrEmailRequired
})
```

Unregistered violations and connection exceptions are wrapped in `DataExceptionError` and `ConnectionExceptionError` respectively, both implementing the `pgw.Error` interface.

## Tracing

Implement `pgw.Tracer` and pass it via `WithTracer`. The adapter covers queries, batch operations, `COPY FROM`, and prepared statements.

```go
type myTracer struct{}

func (t *myTracer) ShouldTrace(ctx context.Context) bool {
    return trace.SpanFromContext(ctx).IsRecording()
}

func (t *myTracer) StartTrace(ctx context.Context, method, sql string, args []any) (context.Context, func(error)) {
    ctx, span := otel.Tracer("pgw").Start(ctx, method, trace.WithAttributes(
        attribute.String("db.statement", sql),
    ))
    return ctx, func(err error) {
        if err != nil {
            span.RecordError(err)
        }
        span.End()
    }
}
```

```go
pgw.WithTracer(&myTracer{})
```

## Pool states

Each pool transitions through these states, visible via `Pool.GetState()` and subscribable via `Pool.SubscribeStateChange(ctx)`:

| State | Meaning |
|---|---|
| `connecting` | Initial state before the first health check |
| `connected` | Reachable and passed health check |
| `error` | Last health check or migration verification failed |
| `closed` | Pool was shut down |

## Pool operations

`*Pool` exposes the standard pgx surface:

- `Exec`, `Query`, `QueryRow`
- `Begin`, `BeginTx`
- `Acquire`, `AcquireAllIdle`, `AcquireFunc`
- `CopyFrom`, `SendBatch`

All methods pass errors through the registered constraint processors before returning.
