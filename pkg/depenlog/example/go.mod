module github.com/webitel/webitel-go-kit/pkg/depenlog/example

go 1.25.4

require (
	github.com/webitel/webitel-go-kit/pkg/depenlog v0.0.0
	github.com/webitel/webitel-go-kit/pkg/logger v0.1.1
	github.com/webitel/webitel-go-kit/pkg/semconv v0.0.0
	go.opentelemetry.io/otel/trace v1.44.0
	go.uber.org/fx v1.24.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/webitel/wlog v0.0.0-20250325101442-de4f125c1ec7 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.19.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/log v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

replace github.com/webitel/webitel-go-kit/pkg/depenlog => ../

replace github.com/webitel/webitel-go-kit/pkg/logger => ../../logger

replace github.com/webitel/webitel-go-kit/pkg/semconv => ../../semconv
