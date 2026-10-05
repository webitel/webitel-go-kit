module github.com/webitel/webitel-go-kit/pkg/depenlog

go 1.25.4

require (
	github.com/webitel/webitel-go-kit/pkg/logger v0.0.0
	github.com/webitel/webitel-go-kit/pkg/semconv v0.0.0
	go.opentelemetry.io/otel/trace v1.39.0
	go.uber.org/fx v1.24.0
	google.golang.org/grpc v1.80.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/webitel/wlog v0.0.0-20250325101442-de4f125c1ec7 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.0.0-20240812153829-bb9ac54eca05 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/log v0.4.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
)

// Local sibling modules; replace with version tags on release.
replace github.com/webitel/webitel-go-kit/pkg/logger => ../logger

replace github.com/webitel/webitel-go-kit/pkg/semconv => ../semconv
