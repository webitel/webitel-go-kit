package stdout

import (
	sdk "go.opentelemetry.io/otel/sdk/log"
	// plugin(s)
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout/codec/json"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout/codec/otel"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout/codec/text"
)

type Encoder interface {
	Encode(sdk.Record) error
}

type encoder struct {
	Encoder
}
