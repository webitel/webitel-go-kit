# semconv

Go binding for [Webitel semantic conventions](https://github.com/webitel/opentelemetry-semantic-conventions).

Conventions OpenTelemetry already defines are not repeated here:
use `go.opentelemetry.io/otel/semconv` for those.

## Usage

Each registry release is a separate package. Attributes are in the version
package, metrics and events in one `<namespace>conv` package per namespace:

```go
import (
	semconv "github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.2.0"
	"github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.2.0/healthconv"
)

semconv.WebitelHealthCheckNameKey            // webitel.health.check.name
healthconv.NewCheckDurationObservable(meter) // webitel.health.check.duration
```

An event is built by its constructor, with required attributes as parameters
and optional ones as options, and logged with `pkg/logging`:

```go
log.WarnContext(ctx, "relay attempt failed",
	logging.Ev(outboxconv.NewRelayFailedEvent("timeout",
		outboxconv.RelayFailedWithMessagingDestinationName("orders"))))
```

The generated packages do not import `pkg/logging`: an event type satisfies
`logging.Event` through its `EventName` and `Attributes` methods.

`SensitiveAttributes` in the version package maps every attribute the registry
marks sensitive to its class, `pii` or `secret`, for log redaction. Deprecated
attributes stay in it. Versions generated before the marking existed have no
such map.

## Generating

```sh
make generate TAG=v0.2.0
```

### Unreleased conventions

To try a convention change before it is released, generate into `dev/` from a
local checkout or a pushed branch:

```sh
make generate TAG=dev REGISTRY=../../../../opentelemetry-semantic-conventions/model
make generate TAG=dev REGISTRY='https://github.com/webitel/opentelemetry-semantic-conventions.git@my-branch[model]'
```

`templates/registry/go` is vendored from
[opentelemetry-go v1.46.0](https://github.com/open-telemetry/opentelemetry-go/tree/58db4c898f5b5594f8ba78f156475bf48486e2f2/semconv/templates/registry/go).
