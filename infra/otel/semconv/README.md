# semconv

Go binding for [Webitel semantic conventions](https://github.com/webitel/opentelemetry-semantic-conventions).

Conventions OpenTelemetry already defines are not repeated here:
use `go.opentelemetry.io/otel/semconv` for those.

## Usage

Each registry release is a separate package. Attributes are in the version
package, metrics in one `<namespace>conv` package per namespace:

```go
import (
	semconv "github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.2.0"
	"github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.2.0/healthconv"
)

semconv.WebitelHealthCheckNameKey            // webitel.health.check.name
healthconv.NewCheckDurationObservable(meter) // webitel.health.check.duration
```

## Generating

```sh
make generate TAG=v0.2.0
```

To move to a new release, bump `TAG` in the `Makefile` and run `make generate`.
It creates a new version directory next to the existing ones; delete an old
one once nothing imports it. CI fails if the committed code differs from what
`make generate` produces.

### Unreleased conventions

To try a convention change before it is released, generate into `dev/` from a
local checkout or a pushed branch:

```sh
make generate TAG=dev REGISTRY=../../../../opentelemetry-semantic-conventions/model
make generate TAG=dev REGISTRY='https://github.com/webitel/opentelemetry-semantic-conventions.git@my-branch[model]'
```

`templates/registry/go` is vendored from
[opentelemetry-go v1.46.0](https://github.com/open-telemetry/opentelemetry-go/tree/58db4c898f5b5594f8ba78f156475bf48486e2f2/semconv/templates/registry/go).
