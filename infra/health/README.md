# health

Health probes for a service and its dependencies. One registry runs the checks
in the background and caches the last result; transports only read that cache
and never run a check themselves — so the number of readers has no effect on
the load a dependency sees.

Two transports ship with it: HTTP (`/livez`, `/readyz`, `/healthz`) and systemd
`sd_notify` (`READY`, `STATUS`, `STOPPING`, `WATCHDOG`). Service discovery takes
the verdict directly, as a `func() (bool, error)`.

Standard library only. The package reads no environment, files or flags — the
caller fills `Config`, or takes `DefaultConfig()`.

## How it works

A check is an ordinary `func(context.Context) error`, so an internal check and a
database ping look the same to the registry. Each registered check gets its own
goroutine that runs it, records the result, waits one `Interval`, and repeats.

Registration decides what a failure *means*, and it is chosen by which method
you call rather than by a struct field:

| | meaning | when it fails |
|---|---|---|
| `Critical` | a node-local fault — moving traffic to another node genuinely helps | node leaves rotation |
| `Liveness` | "is this process wedged?" | node leaves rotation, and `/livez` returns 503 |
| `Informational` | everything else, typically shared infrastructure | `degraded`: visible, but the node **stays** in rotation |

Making shared infrastructure critical is usually wrong: if a shared database is
critical, losing it takes the whole fleet out of rotation at once, and there is
nowhere left to send traffic.

Three rules do the rest:

- **One run at a time.** The check runs inline in its own loop, so a check that
  ignores its context delays only itself. It cannot pile up goroutines draining
  the very pool it is testing.
- **Hysteresis.** A check goes bad after `FailThreshold` consecutive failures,
  and recovers on the first success but never before `MinUnready` — otherwise a
  flapping dependency keeps the node in rotation for most of its broken life.
- **Staleness.** A result older than `StaleAfter` reads as `unknown`, not
  healthy, so a wedged check cannot serve its last good answer forever.

A registry with no checks — or with only informational ones — is **not ready**:
nothing registered answers "can this node take traffic?", so no verdict has been
earned.

Each `CheckResult` also carries `LastDuration` (elapsed time of the last
completed run; zero until the first) and `Transitions` (how many times the
check changed to `ok` and to `fail` since the process started). Both come from
`Snapshot()`. The first completed run counts as one transition, since a check
starts `unknown`.

Staleness is applied when a result is read, never stored: a stale check reads
`unknown` but `Transitions` does not move. The counter answers "how often did
the verdict flip", not "is this check healthy now" — `Status` answers that.

`infra/otel/instrumentation/health` exports these as OpenTelemetry metrics.

## Quick start

```go
h := health.New(health.DefaultConfig(), log,
    health.WithTransport(healthhttp.NewServer("127.0.0.1:8081")),
    health.WithTransport(sdnotify.New()), // nil when NOTIFY_SOCKET is unset, ignored
)

h.Critical("grpc", health.ListenerCheck(lis))
h.Informational("postgres", store.Ping)

if err := h.Start(ctx); err != nil {
    return err
}
```

The registry owns its transports. `Start` runs the checks, then starts each
transport in order. If one fails, the ones already started are stopped, the
checks are halted, and the returned error joins the failure with any rollback
failure. That registry is finished: exit, or build a new one. Nil and typed-nil
transports are ignored.

A transport implements `health.Transport`: `Start(ctx, *Registry)` and
`Stop(ctx)`. `Stop` on a transport that never started must be safe: it must not
block or fail, though it may still send a shutdown signal, as `sdnotify` does.

`ListenerCheck` dials a listener to prove it still accepts connections. It uses
the listener's own address, not the one advertised to service discovery — a host
often cannot reach itself that way, and the dial would be testing the network
rather than the socket. A wildcard bind is rewritten to loopback in the
listener's own address family — `::1` for `[::]`, `127.0.0.1` for `0.0.0.0` —
since a `tcp6` listener refuses an IPv4 probe.

Hand the verdict to service discovery:

```go
discovery.NewServiceDiscovery(nodeID, url, h.ReadyFunc())
```

`ReadyFunc` reads the cached snapshot and returns instantly. **When the verdict
is `false` the error is never nil**, so a caller may use it without a nil check.

Shut down with `Shutdown`, which runs the sequence in the order the registry
requires: `Drain`, then every transport in reverse start order, then the
DrainHold wait, then the scheduler. `ctx` must carry at least `DrainHold` of
budget: `Drain` returns immediately and the hold is absorbed inside `Stop`, so a
shorter context cuts the hold short and service discovery may never see the node
leave rotation.

```go
ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
defer cancel()

err := h.Shutdown(ctx)
```

Every transport is stopped even if an earlier one fails, and the errors are
joined. The equivalent by hand, if you need to interleave something:

```go
h.Drain()    // not ready from here on, one way
h.Stop(ctx)  // stops the transports, waits out the rest of DrainHold, halts the scheduler
```

## HTTP

Mount on an existing router, past authorization — probes must answer without a
token:

```go
router.Handle("/livez", healthhttp.LivenessHandler(h))
router.Handle("/readyz", healthhttp.ReadinessHandler(h))
router.Handle("/healthz", healthhttp.HealthHandler(h))
```

Or run a listener of your own, for a service with no HTTP server. `NewServer`
is a transport: pass it to `health.WithTransport` and the registry binds it on
`Start` and stops it on `Stop`. It returns nil for an empty address.

```go
h := health.New(cfg, log, health.WithTransport(healthhttp.NewServer("127.0.0.1:8081")))
```

`healthhttp.Handler(h)` serves all three from one mount by routing on the last
path segment, which works at the root, under a prefix, or at one exact path. The
three named handlers ignore the path instead, so a single mount cannot be wrong.

| endpoint | 200 | 503 |
|---|---|---|
| `/livez` | `alive` | `wedged` — the scheduler stopped turning |
| `/readyz` | `ready`, `degraded` | `not_ready`, `unknown`, `stopping` |
| `/healthz` | JSON detail, same codes as `/readyz` | |

Anything other than `GET` or `HEAD` returns 405 with an `Allow` header.

```json
{
  "status": "degraded",
  "checks": [
    { "name": "grpc",     "group": "critical",      "status": "ok",   "since": "2026-08-12T10:04:11Z" },
    { "name": "postgres", "group": "informational", "status": "fail", "since": "2026-08-12T10:07:52Z" }
  ]
}
```

**A check's error text never reaches a response body**, on any endpoint. Names
and states are safe to expose; the text of a database error is not, and it goes
to the logger instead. `?verbose` upgrades `/livez` and `/readyz` to the same
JSON, and only on a `Server` whose listener bound to loopback.

## systemd

```go
h := health.New(cfg, log, health.WithTransport(sdnotify.New())) // New is nil when NOTIFY_SOCKET is unset
```

With no `NOTIFY_SOCKET` the transport does nothing at all, so a dev machine, a
container and `go test` cost nothing and need no flag.

| message | when |
|---|---|
| `READY=1` | once, when every critical check is green |
| `STATUS=` | on a state change — shows in `systemctl status` |
| `STOPPING=1` | at the start of shutdown |
| `WATCHDOG=1` | periodically, at `WATCHDOG_USEC/2` |

`NOTIFY_SOCKET`, `WATCHDOG_USEC` and `WATCHDOG_PID` are systemd's interface, so
this is the one place the package reads the environment. The watchdog ping is
deliberately **not** gated on dependencies — only on whether the registry's own
scheduler is still turning. Gating it on a database would let one outage restart
the entire fleet at once, and a restart does not fix a database.

Under `Type=notify`, pick `WatchdogSec` against `StaleAfter`, not by intuition:
one skipped ping already consumes the whole margin, so use at least
`4 × StaleAfter` (60s at the defaults) and never below `2 × StaleAfter`. Without
`WithStartTimeout` there is no `READY=1` fallback, so a unit whose critical check
never goes green stays in `activating` until `TimeoutStartSec`.

## Configuration

```go
health.Config{
    Interval:      5 * time.Second,  // how often each check runs
    Timeout:       2 * time.Second,  // bounds one run; must be below Interval
    FailThreshold: 3,                // failures in a row before a check is bad
    MinUnready:    15 * time.Second, // shortest time a check stays bad
    StaleAfter:    15 * time.Second, // when a result stops counting
    DrainHold:     10 * time.Second, // how long Stop waits after Drain
}
```

Any field left at zero falls back to its `DefaultConfig` value, so a partial
`Config` is safe. The defaults are a set rather than independent knobs: they are
tuned against how often service discovery asks for the verdict, and together they
put a broken node out of rotation in roughly 25 seconds.
