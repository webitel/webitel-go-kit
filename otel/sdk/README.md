# [O]pen[Tel]emetry Configuration

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
|[Exporter Selection](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#exporter-selection)||||
|`OTEL_LOGS_EXPORTER`|Logs exporter to be used</br>`otlp`, `console`, `none`||`otlpgrpc`, `otlphttp`,</br>`stdout`, `stderr`, `file:/path/logs.otel`|
|`OTEL_TRACES_EXPORTER`|Trace exporter to be used</br>`otlp`, `zipkin`, `console`, `none`||`otlpgrpc`, `otlphttp`,</br>`stdout`, `stderr`, `file:/path/traces.otel`|
|`OTEL_METRICS_EXPORTER`|Metrics exporter to be used</br>`otlp`, `prometheus`, `console`, `none`||`otlpgrpc`, `otlphttp`,</br>`stdout`, `stderr`, `file:/path/metrics.otel`|
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
|`OTEL_LOGRECORD_CODEC`|Codec to be used for standard output. Can be applied while `OTEL_LOGS_EXPORTER`=[`stdout`\|`stderr`\|`file:`].</br>Accept: `text`, `json`, `otel`|`otel`||
|`OTEL_LOGRECORD_INDENT`|Use indentation, pretty print. Can be applied while `OTEL_LOGRECORD_CODEC`=[`json`\|`otel`].</br>Accept: boolean or whitespace(s) characters.</br>`true`,`'\t'`, `"  "`|`false`||
|`OTEL_LOGRECORD_TIMESTAMP`|Timestamps layout. Can be applied for any standard codec.|`"2006-01-02T15:04:05.999Z07:00"`|See [Time.Format](https://pkg.go.dev/time#pkg-constants).|
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
| <s>[Prometheus Exporter](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#prometheus-exporter)</s>||||
|`OTEL_EXPORTER_PROMETHEUS_HOST`|Host used by the Prometheus exporter|`localhost`||
|`OTEL_EXPORTER_PROMETHEUS_PORT`|Port used by the Prometheus exporter|`9464`||
|||||
| [Metrics SDK Configuration](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#metrics-sdk-configuration)||||
|`OTEL_METRICS_EXEMPLAR_FILTER`|Filter for which measurements can become Exemplars|`trace_based`||
|||||
|[Periodic exporting MetricReader](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#periodic-exporting-metricreader)||||
|`OTEL_METRIC_EXPORT_INTERVAL`|The time interval (in milliseconds) between the start of two export attempts.|`60000`||
|`OTEL_METRIC_EXPORT_TIMEOUT`|Maximum allowed time (in milliseconds) to export data.|`30000`||
|||||
| [OTLP Exporter Configuration](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/)||||
|`OTEL_EXPORTER_OTLP_ENDPOINT`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_ENDPOINT`|Target to which the exporter is going to send spans, metrics, or logs. The option SHOULD accept any form allowed by the underlying gRPC client implementation. Additionally, the option MUST accept a URL with a scheme of either `http` or `https`. A scheme of `https` indicates a **secure** connection and takes precedence over the `insecure` configuration setting. A scheme of `http` indicates an **insecure** connection and takes precedence over the `insecure` configuration setting. If the gRPC client implementation does not support an endpoint with a scheme of `http` or `https` then the endpoint SHOULD be transformed to the most sensible format for that implementation.|OTLP/**gRPC**:</br>`_{SIGNAL}_ENDPOINT`=`http://localhost:4317`</br>OTLP/**HTTP**:</br>`_ENDPOINT`=`http://localhost:4318`</br>`_LOGS_ENDPOINT`=`http://localhost:4318/v1/logs`</br>`_TRACES_ENDPOINT`=`http://localhost:4318/v1/traces`</br>`_METRICS_ENDPOINT`=`http://localhost:4318/v1/metrics`||
|`OTEL_EXPORTER_OTLP_PROTOCOL`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_PROTOCOL`|The transport protocol. See [Specify Protocol](https://opentelemetry.io/docs/specs/otel/protocol/exporter/#specify-protocol) for more details.|`grpc`|`grpc` to use OTLP/gRPC</br>`http/protobuf` to use OTLP/HTTP + protobuf</br>`http/json` to use OTLP/HTTP + JSON|
|`OTEL_EXPORTER_OTLP_INSECURE`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_INSECURE`|Whether to enable client transport security for the exporter’s gRPC connection. This option only applies to OTLP/gRPC when an `endpoint` is provided without the `http` or `https` scheme - OTLP/HTTP always uses the scheme provided for the `endpoint`.|`false`||
|`OTEL_EXPORTER_OTLP_TIMEOUT`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_TIMEOUT`|Maximum time the OTLP exporter will wait for each batch export.|`10000`||
|`OTEL_EXPORTER_OTLP_HEADERS`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_HEADERS`|Key-value pairs to be used as headers associated with gRPC or HTTP requests. See [Specifying headers](https://opentelemetry.io/docs/specs/otel/protocol/exporter/#specifying-headers-via-environment-variables) for more details.||Example: `api-key=key,other-config-value=value`|
|`OTEL_EXPORTER_OTLP_CERTIFICATE`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_CERTIFICATE`|The trusted certificate to use when verifying a server’s TLS credentials. Should only be used for a secure connection.|||
|`OTEL_EXPORTER_OTLP_CLIENT_KEY`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_CLIENT_KEY`|Clients private key to use in mTLS communication in PEM format.|||
|`OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_CLIENT_CERTIFICATE`|Client certificate/chain trust for clients private key to use in mTLS communication in `PEM` format.|||
|`OTEL_EXPORTER_OTLP_COMPRESSION`</br>`OTEL_EXPORTER_OTLP_{SIGNAL}_COMPRESSION`|Compression key for supported compression types. Supported compression: `gzip`.|||

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
