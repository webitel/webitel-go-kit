// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package stdout // import "github.com/webitel/webitel-go-kit/otel/log/stdout"

import (
	"io"
	"os"
)

var (
	defaultWriter      io.Writer = os.Stdout
	defaultPrettyPrint           = false
	defaultTimestamps            = true
)

// config contains options for the STDOUT exporter.
type config struct {
	// Output is the destination.  If not set, os.Stdout is used.
	Output io.Writer

	// Formatter
	Codec Encoder

	// PrettyPrint will encode the output into readable JSON. Default is
	// false.
	PrettyPrint bool

	// Timestamps specifies if timestamps should be printed. Default is
	// true.
	Timestamps bool
}

// newConfig creates a validated Config configured with options.
func newConfig(options []Option) config {
	cfg := config{
		Output:      defaultWriter,
		PrettyPrint: defaultPrettyPrint,
		Timestamps:  defaultTimestamps,
	}
	for _, opt := range options {
		cfg = opt.apply(cfg)
	}
	return cfg
}

// Option sets the configuration value for an Exporter.
type Option interface {
	apply(config) config
}

// WithWriter sets the export stream destination.
func WithWriter(w io.Writer) Option {
	return writerOption{w}
}

type writerOption struct {
	W io.Writer
}

func (o writerOption) apply(cfg config) config {
	cfg.Output = o.W
	return cfg
}

// WithPrettyPrint prettifies the emitted output.
func WithPrettyPrint() Option {
	return prettyPrintOption(true)
}

type prettyPrintOption bool

func (o prettyPrintOption) apply(cfg config) config {
	cfg.PrettyPrint = bool(o)
	return cfg
}

// WithoutTimestamps sets the export stream to not include timestamps.
func WithoutTimestamps() Option {
	return timestampsOption(false)
}

type timestampsOption bool

func (o timestampsOption) apply(cfg config) config {
	cfg.Timestamps = bool(o)
	return cfg
}

// WithCodec sets the record formatter.
func WithCodec(codec Encoder) Option {
	return codecOption{codec}
}

type codecOption struct {
	Codec Encoder
}

func (o codecOption) apply(cfg config) config {
	cfg.Codec = o.Codec
	return cfg
}
