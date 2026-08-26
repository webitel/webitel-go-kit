# semconv

Webitel's own semantic conventions, and the Go package generated from them.

Only conventions with no upstream equivalent live here. Anything OpenTelemetry
already defines is imported from `go.opentelemetry.io/otel/semconv` directly and
never redefined, so this package stays small and cannot drift from upstream.

## Regenerating

```sh
./generate.sh            # regenerate attribute_group.go and docs/
./generate.sh --check    # fail if the committed output is stale (this is what CI runs)
```

`docs/` is the generated attribute reference, rendered with OpenTelemetry's own
markdown templates. Both it and the Go package are committed, so nothing
downstream needs Weaver.

## Adding an attribute

Add it to a `registry.webitel.*` group in `registry/registry.yaml`, then run
`./generate.sh` and commit the result.

```yaml
- id: webitel.queue.name
  type: string
  stability: development
  brief: The name of the queue.
  examples: ["support", "sales"]
```

Every name must start with `webitel.`; `generate.sh` fails if one does not.
Requirement levels do not belong here — they belong to the metric or span group
that references the attribute, because the same attribute can be required for
one signal and optional for another. See the
[semconv syntax](https://github.com/open-telemetry/weaver/blob/main/schemas/semconv-syntax.md).

## Adding a metric

Not yet possible. The Go templates are opentelemetry-go's own, and their metric
template imports `go.opentelemetry.io/otel/semconv/internal/metricpool`, which
Go forbids this module from importing. `generate.sh` fails with that diagnosis
rather than emitting a package that cannot build.
