# semconv

Webitel's own semantic conventions, and the Go package generated from them.

Only conventions with no upstream equivalent live here. Anything OpenTelemetry
already defines is imported from `go.opentelemetry.io/otel/semconv` directly.

## Regenerating

```sh
./generate.sh            # regenerate attribute_group.go and webitelconv/
./generate.sh --check    # fail if the committed output is stale (this is what CI runs)
```

The generated Go is committed, so nothing downstream needs Weaver.

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

Add a `type: metric` group to `registry/registry.yaml`. Its attributes are
`ref`s into a `registry.webitel.*` group, and this is where their requirement
level is set. Run `./generate.sh` and commit the result.

```yaml
- id: metric.webitel.health.check.duration
  type: metric
  metric_name: webitel.health.check.duration
  stability: development
  brief: Elapsed time of a check's last completed run.
  instrument: gauge
  unit: s
  attributes:
    - ref: webitel.health.check.name
      requirement_level: required
    - ref: webitel.health.check.group
      requirement_level: required
```

Metrics land in `webitelconv/`, one constructor per metric with a typed method
per attribute:

```go
duration, err := webitelconv.NewHealthCheckDuration(meter)
// ...
o.ObserveFloat64(duration.Inst(), secs, metric.WithAttributes(
	duration.AttrHealthCheckName(name),
	duration.AttrHealthCheckGroup(webitelconv.HealthCheckGroupCritical),
))
```
