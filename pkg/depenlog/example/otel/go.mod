module github.com/webitel/webitel-go-kit/pkg/depenlog/example/otel

go 1.25.4

require (
	github.com/webitel/webitel-go-kit/infra/otel v0.0.0
	github.com/webitel/webitel-go-kit/pkg/depenlog v0.0.0
	github.com/webitel/webitel-go-kit/pkg/semconv v0.0.0
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/sdk v1.43.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/webitel/webitel-go-kit/pkg/logger v0.0.0 // indirect
	github.com/webitel/wlog v0.0.0-20250325101442-de4f125c1ec7 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/bridges/otelslog v0.17.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.0 // indirect
	go.opentelemetry.io/otel/log v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.19.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/fx v1.24.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

replace github.com/webitel/webitel-go-kit/pkg/depenlog => ../../

replace github.com/webitel/webitel-go-kit/pkg/logger => ../../../logger

replace github.com/webitel/webitel-go-kit/pkg/semconv => ../../../semconv

replace github.com/webitel/webitel-go-kit/infra/otel => ../../../../infra/otel
