# [O]pen[Tel]emetry Configuration

Per-package detail for metrics: [`sdk/metric/README.md`](metric/README.md).

## [Environment Variable Specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#general-sdk-configuration)

| Environment | Description | Default | Notes |
|-------------|-------------|---------|-------|
|||||
|[General SDK Configuration](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#general-sdk-configuration)||||
|`OTEL_SDK_DISABLED`|Disable the SDK for all signals|`false`|Boolean value. If `true`, a no-op SDK implementation will be used for all telemetry signals. Any other value or absence of the variable will have no effect and the SDK will remain enabled. This setting has no effect on propagators configured through the `OTEL_PROPAGATORS` variable.|
|`OTEL_RESOURCE_ATTRIBUTES`|Key-value pairs to be used as resource attributes</br>See Resource semantic conventions for details.||See [Resource SDK](https://opentelemetry.io/docs/specs/otel/resource/sdk/#specifying-resource-information-via-an-environment-variable) for more details.|
|`OTEL_SERVICE_NAME`|Sets the value of the `service.name` resource attribute||If `service.name` is also provided in `OTEL_RESOURCE_ATTRIBUTES`, then `OTEL_SERVICE_NAME` takes precedence.|
|`OTEL_LOG_LEVEL`|Log level used by the SDK logger|`error`|`debug`, `info`, `warn`, `error`|
|`OTEL_PROPAGATORS`|Propagators to be used as a comma-separated list|`tracecontext,baggage`|Values MUST be deduplicated in order to register a Propagator only once.|
|`OTEL_TRACES_SAMPLER`|Sampler to be used for traces|`parentbased_always_on`|See [Sampling](https://opentelemetry.io/docs/specs/otel/trace/sdk/#sampling).|
|`OTEL_TRACES_SAMPLER_ARG`|String value to be used as the sampler argument||The specified value will only be used if `OTEL_TRACES_SAMPLER` is set. Each Sampler type defines its own expected input, if any. Invalid or unrecognized input MUST be logged and MUST be otherwise ignored, i.e. the implementation MUST behave as if `OTEL_TRACES_SAMPLER_ARG` is not set.|
|||||
|[Exporter Selection](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#exporter-selection)|||`file:` exporter(s) use rotation mechanism.</br>See undelying [lumberjack.v2.Logger](https://pkg.go.dev/gopkg.in/natefinch/lumberjack.v2#Logger) for details.</br>Defaults: `;max-size=100` `;max-age=30` `;backups=3` `;localtime=true` `;compress=false`|
|`OTEL_LOGS_EXPORTER`|Logs exporter(s) to be used as a comma-separated list</br>`otlp`, `console`, `none`|`stdout`|`otlpgrpc`, `otlphttp`,</br>`stdout`, `stderr`, `file:/path/logs.otel`</br>e.g. `stdout:,otlpgrpc://collector:4317`</br>`none` disables logs and must be the only value.|
|`OTEL_TRACES_EXPORTER`|Trace exporter to be used</br>`otlp`, `zipkin`, `console`, `none`||`otlpgrpc`, `otlphttp`,</br>`stdout`, `stderr`, `file:/path/traces.otel`|
|`OTEL_METRICS_EXPORTER`|Metrics exporter to be used||`otlpgrpc`, `otlphttp`, `prometheus`,</br>`stdout`, `stderr`, `file:/path/metrics.otel`</br>The spec values `otlp`, `console`, `none` are not accepted; unset means no metrics.|
|||||
|[Attribute Limits](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#attribute-limits)||||
|`OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT`|Maximum allowed attribute value size||no limit|
|`OTEL_ATTRIBUTE_COUNT_LIMIT`|Maximum allowed attribute count|`128`||
|||||
|[LogRecord Limits](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#logrecord-limits)||||
|`OTEL_LOGRECORD_ATTRIBUTE_VALUE_LENGTH_LIMIT`|Maximum allowed attribute value size||no limit|
|`OTEL_LOGRECORD_ATTRIBUTE_COUNT_LIMIT`|Maximum allowed log record attribute count|`128`||
|||||
| Standard LogRecord Encoding. ||||
|`OTEL_LOGRECORD_CODEC`|Codec to be used for standard output. Can be applied while `OTEL_LOGS_EXPORTER`=[`stdout`\|`stderr`\|`file:`].</br>Accept: `text`, `json`, `otel`</br>The DSN parameter `codec` overrides it for one exporter, e.g. `stdout:;codec=text,file:/path/logs.json;codec=json` or `stderr://?codec=json`.|`otel`||
|`OTEL_LOGRECORD_COLOR`|Colorize, depending on level, `OTEL_LOGRECORD_CODEC`=`text` records for console output.|`false`||
|`OTEL_LOGRECORD_INDENT`|Use indentation, pretty print. Can be applied while `OTEL_LOGRECORD_CODEC`=[`json`\|`otel`].</br>Accept: boolean or whitespace(s) characters.</br>`true`,`'\t'`, `"  "`|`false`||
|`OTEL_LOGRECORD_TIMESTAMP`|Timestamps layout. Can be applied for any standard codec.|`"2006-01-02T15:04:05.999Z07:00"`|See [Time.Format](https://pkg.go.dev/time#pkg-constants).|
|||||
| [Log Redaction](#log-redaction)||||
|`OTEL_LOGS_REDACTION`|Redact sensitive log record attributes by name|`true`|Webitel-local. Boolean value; `false`/`FALSE`/`f`/`0` disable, anything else that is not a boolean logs a warning and leaves redaction enabled. Has no effect while the sensitive attributes map is empty. Either this variable or `WithSensitiveAttributes(nil)` in code disables redaction. See [Log redaction](#log-redaction).|
|||||
|[Batch LogRecord Processor](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#batch-logrecord-processor)||||
|`OTEL_BLRP_SCHEDULE_DELAY`|Delay interval (in milliseconds) between two consecutive exports|`1000`||
|`OTEL_BLRP_EXPORT_TIMEOUT`|Maximum allowed time (in milliseconds) to export data|`30000`||
|`OTEL_BLRP_MAX_QUEUE_SIZE`|Maximum queue size|`2048`||
|`OTEL_BLRP_MAX_EXPORT_BATCH_SIZE`|Maximum batch size|`512`||
|||||
|[Batch Span Processor](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#batch-span-processor)||||
|`OTEL_BSP_SCHEDULE_DELAY`|Delay interval (in milliseconds) between two consecutive exports|`5000`||
|`OTEL_BSP_EXPORT_TIMEOUT`|Maximum allowed time (in milliseconds) to export data|`30000`||
|`OTEL_BSP_MAX_QUEUE_SIZE`|Maximum queue size|`2048`||
|`OTEL_BSP_MAX_EXPORT_BATCH_SIZE`|Maximum batch size|`512`|Must be less than or equal to `OTEL_BSP_MAX_QUEUE_SIZE`|
|||||
|[Span Limits](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#span-limits)||||
|`OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT`|Maximum allowed attribute value size||no limit|
|`OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT`|Maximum allowed span attribute count|`128`||
|`OTEL_SPAN_EVENT_COUNT_LIMIT`|Maximum allowed span event count|`128`||
|`OTEL_SPAN_LINK_COUNT_LIMIT`|Maximum allowed span link count|`128`||
|`OTEL_EVENT_ATTRIBUTE_COUNT_LIMIT`|Maximum allowed attribute per span event count|`128`||
|`OTEL_LINK_ATTRIBUTE_COUNT_LIMIT`|Maximum allowed attribute per span link count|`128`||
|||||
| <s>[Zipkin Exporter](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#zipkin-exporter)</s>|||||
|`OTEL_EXPORTER_ZIPKIN_ENDPOINT`|Endpoint for Zipkin traces|`http://localhost:9411/api/v2/spans`||
|`OTEL_EXPORTER_ZIPKIN_TIMEOUT`|Maximum time (in milliseconds) the Zipkin exporter will wait for each batch export|`10000`||
|||||
| [Prometheus Exporter](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#prometheus-exporter)||||Runs its own listener on these two variables and serves `/metrics`; nothing to mount. Loopback by default and unauthenticated. Names use `UnderscoreEscapingWithSuffixes` (dots become underscores, unit and `_total` suffixes appended); every series carries `otel_scope_name`, `otel_scope_version` and `otel_scope_schema_url` labels. An invalid port is an error; a blank one is unset. `localhost` binds one address family — set the host explicitly if the scraper uses the other.|
|`OTEL_EXPORTER_PROMETHEUS_HOST`|Host used by the Prometheus exporter|`localhost`||
|`OTEL_EXPORTER_PROMETHEUS_PORT`|Port used by the Prometheus exporter|`9464`||
|||||
| [Metrics SDK Configuration](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#metrics-sdk-configuration)||||
|`OTEL_METRICS_EXEMPLAR_FILTER`|Filter for which measurements can become Exemplars|`trace_based`||
|`OTEL_METRICS_RUNTIME`|Enable the Go runtime metrics (`go.memory.*`, `go.goroutine.count`, `go.processor.limit`, `go.config.gogc`, `go.schedule.duration`)|`false`|Webitel-local. Boolean as `strconv.ParseBool` reads it; an unparsable value logs a warning and leaves it off. Needs a metrics exporter. `WithRuntimeMetrics(false)` in code overrides the variable.|
|||||
|[Periodic exporting MetricReader](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#periodic-exporting-metricreader)||||
|`OTEL_METRIC_EXPORT_INTERVAL`|The time interval (in milliseconds) between the start of two export attempts.|`60000`||
|`OTEL_METRIC_EXPORT_TIMEOUT`|Maximum allowed time (in milliseconds) to export data.|`30000`||
|||||
| [OTLP Exporter Configuration](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/)||||
|`OTEL_EXPORTER_OTLP_ENDPOINT`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_ENDPOINT`|Target to which the exporter sends spans, metrics, or logs. A URL with scheme `https` is a **secure** connection, `http` an **insecure** one; either takes precedence over `_INSECURE`.|OTLP/**gRPC**:</br>`localhost:4317` over TLS</br>OTLP/**HTTP**:</br>`https://localhost:4318`</br>`_LOGS_ENDPOINT`=`…/v1/logs`</br>`_TRACES_ENDPOINT`=`…/v1/traces`</br>`_METRICS_ENDPOINT`=`…/v1/metrics`|The Go exporters default to TLS. For a plain collector set `http://…` or `_INSECURE=true`.|
|`OTEL_EXPORTER_OTLP_PROTOCOL`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_PROTOCOL`|The transport protocol.||Not read by the Go exporters: the protocol is the DSN scheme, `otlpgrpc` or `otlphttp`.|
|`OTEL_EXPORTER_OTLP_INSECURE`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_INSECURE`|Whether to enable client transport security for the exporter’s gRPC connection. This option only applies to OTLP/gRPC when an `endpoint` is provided without the `http` or `https` scheme - OTLP/HTTP always uses the scheme provided for the `endpoint`.|`false`||
|`OTEL_EXPORTER_OTLP_TIMEOUT`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_TIMEOUT`|Maximum time the OTLP exporter will wait for each batch export.|`10000`||
|`OTEL_EXPORTER_OTLP_HEADERS`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_HEADERS`|Key-value pairs to be used as headers associated with gRPC or HTTP requests. See [Specifying headers](https://opentelemetry.io/docs/specs/otel/protocol/exporter/#specifying-headers-via-environment-variables) for more details.||Example: `api-key=key,other-config-value=value`|
|`OTEL_EXPORTER_OTLP_CERTIFICATE`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_CERTIFICATE`|The trusted certificate to use when verifying a server’s TLS credentials. Should only be used for a secure connection.|||
|`OTEL_EXPORTER_OTLP_CLIENT_KEY`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_CLIENT_KEY`|Clients private key to use in mTLS communication in PEM format.|||
|`OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_CLIENT_CERTIFICATE`|Client certificate/chain trust for clients private key to use in mTLS communication in `PEM` format.|||
|`OTEL_EXPORTER_OTLP_COMPRESSION`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_COMPRESSION`|Compression key for supported compression types. Supported compression: `gzip`.|||

## Log redaction

`Configure` registers a log record processor that renders the values of
sensitive attributes before any exporter sees the record. Attributes are
matched by name, also inside map values (slog groups) and slices of maps:

- `pii` values become a keyed pseudonym, `pii:` and 32 hex characters, or are
  dropped while no HMAC salt is set with `redact.SetPolicy`;
- `secret` values become `set` or `empty`, or are dropped with
  `SecretMode: redact.Drop`;
- values of any other class are dropped.

These are the rules of `.PII()`/`.Secret()` fields in `pkg/logging`. A value
that already looks rendered (`redact.Rendered`) is kept as is, so a field
marked in code is not rendered twice. The exception is `SecretMode:
redact.Drop`: a rendered `secret` value is dropped too.

The attributes come from `WithSensitiveAttributes`, which maps names to
`"pii"` or `"secret"`, usually the `SensitiveAttributes` map of a generated
`semconv` version:

```go
shutdown, err := otelsdk.Configure(ctx,
	otelsdk.WithSensitiveAttributes(semconv.SensitiveAttributes),
)
```

Until a `semconv` version with `SensitiveAttributes` is released, the default
map is empty and nothing is redacted unless a map is passed. A nil or empty
map, or `OTEL_LOGS_REDACTION=false`, disables redaction, and the processor is
not registered. Without redaction, only fields marked in code are protected.

Try an [example](https://github.com/webitel/webitel-go-kit/blob/features/otel/otel/example)

```sh
LOG_LEVEL=info \
OTEL_LOG_LEVEL= \
\
OTEL_LOGS_EXPORTER=stdout \
OTEL_LOGRECORD_CODEC=text \
\
OTEL_TRACES_EXPORTER=otlpgrpc \
\
OTEL_EXPORTER_OTLP_ENDPOINT=https://remote.collector.otlp:4317 \
\
go run otel/example/*.go
```
