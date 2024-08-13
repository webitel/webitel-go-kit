// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package stdout // import "github.com/webitel/webitel-go-kit/otel/log/stdout"

import (
	"context"
	"sync/atomic"

	otel "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout/codec/otel"
	sdk "go.opentelemetry.io/otel/sdk/log"
)

var _ sdk.Exporter = &Exporter{}

// Exporter writes JSON-encoded log records to an [io.Writer] ([os.Stdout] by default).
// Exporter must be created with [New].
type Exporter struct {
	// encoder    atomic.Pointer[json.Encoder]
	encoder    atomic.Pointer[encoder]
	timestamps bool
}

// New creates an [Exporter].
func New(options ...Option) (*Exporter, error) {
	cfg := newConfig(options)

	// enc := json.NewEncoder(cfg.Output)
	// if cfg.PrettyPrint {
	// 	enc.SetIndent("", "\t")
	// }

	codec := cfg.Codec
	if codec == nil {
		codec = otel.NewCodec(cfg.Output)
	}

	e := Exporter{
		timestamps: cfg.Timestamps,
	}
	e.encoder.Store(&encoder{codec})
	// e.encoder.Store(&encoder{enc})

	return &e, nil
}

// Export exports log records to writer.
func (e *Exporter) Export(ctx context.Context, records []sdk.Record) error {
	enc := e.encoder.Load()
	if enc == nil {
		return nil
	}

	for _, record := range records {
		// Honor context cancellation.
		if err := ctx.Err(); err != nil {
			return err
		}

		// // Encode record, one by one.
		// recordJSON := e.newRecordJSON(record)
		// if err := enc.Encode(recordJSON); err != nil {
		// 	return err
		// }
		if err := enc.Encode(record); err != nil {
			return err
		}
	}
	return nil
}

// Shutdown shuts down the Exporter.
// Calls to Export will perform no operation after this is called.
func (e *Exporter) Shutdown(context.Context) (err error) {
	e.encoder.Store(nil)
	// var output io.Writer
	// if hook, do := output.(io.Closer); do {
	// 	err = hook.Close()
	// }
	return // err?
}

// ForceFlush performs no action.
func (e *Exporter) ForceFlush(context.Context) error {
	return nil
}
