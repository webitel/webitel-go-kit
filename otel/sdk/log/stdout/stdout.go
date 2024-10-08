package stdout

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	// "github.com/pkg/errors"

	// "go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel"
	sdk "go.opentelemetry.io/otel/sdk/log"

	"github.com/webitel/webitel-go-kit/otel/internal"
	"github.com/webitel/webitel-go-kit/otel/sdk/log"
	"github.com/webitel/webitel-go-kit/otel/sdk/log/stdout/codec"
)

func withOptions(ctx context.Context, rawDSN string) ([]log.Option, error) {

	var scheme string
	colon := strings.IndexByte(rawDSN, ':')
	if colon < 0 {
		scheme, rawDSN = rawDSN, ""
	} else {
		scheme, rawDSN = rawDSN[0:colon], rawDSN[colon+1:]
	}
	scheme = strings.ToLower(scheme)

	var (
		err error
		out io.WriteCloser
	)
	switch scheme {
	case "stdout":
		out = os.Stdout
	case "stderr":
		out = os.Stderr
	case "file":
		{
			out, err = internal.FileDSN(rawDSN)
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("scheme %s: unknown", scheme)
	}

	var (
		noColor   = true
		encoder   Encoder // formatter
		codecOpts []codec.Option
		newCodec  = func(name string) {
			if encoder != nil {
				return // already
			}
			// Apply: COLORIZE options
			if noColor {
				codecOpts = append(codecOpts,
					codec.WithNoColor(),
				)
			}
			encoder = codec.NewCodec(name, out, codecOpts...)
			if encoder == nil {
				re := fmt.Errorf("invalid OTEL_LOGRECORD_CODEC value %s: codec %[1]q not found", name)
				err = errors.Join(err, re)
				otel.Handle(re)
			}
		}
	)
	internal.Environment.Apply(
		internal.EnvString("LOGRECORD_TIMESTAMP", func(s string) {
			// fmt.Printf("LOGS_FORMAT_TIMESTAMP: [%s]\n", s)
			codecOpts = append(codecOpts, codec.WithTimestamps(s))
		}),
		internal.EnvString("LOGRECORD_INDENT", func(s string) {
			indent := s
			yes, re := strconv.ParseBool(s)
			if re == nil {
				if indent = ""; yes {
					indent = "\t"
				}
			}
			for _, c := range indent {
				if !unicode.IsSpace(c) {
					re = fmt.Errorf("invalid OTEL_LOGS_INDENT value %q: expect boolean or whitespace(s)", indent)
					err = errors.Join(err, re)
					otel.Handle(re)
					return
				}
			}

			codecOpts = append(codecOpts,
				codec.WithPrittyPrint(indent),
			)
		}),
		internal.EnvString("LOGRECORD_COLOR", func(s string) {
			// Accept: [ false | true | "auto" ]
			enable, _ := strconv.ParseBool(s)
			if strings.EqualFold(s, "auto") {
				enable = true
			}
			noColor = !enable
		}),
		internal.EnvString("LOGRECORD_CODEC", func(s string) {
			newCodec(s)
		}),
	)

	// OTEL_LOGRECORD_CODEC = "text"; default
	newCodec("text")

	if err != nil {
		return nil, err
	}

	exporter, err := New(
		// options ...
		WithWriter(out),
		WithCodec(encoder),
	)
	if err != nil {
		return nil, err
	}

	var processor sdk.Processor
	switch scheme {
	case "stdout", "stderr":
		processor = sdk.NewSimpleProcessor(
			exporter, // opts...
		)
	default:
		// case "file":
		processor = sdk.NewBatchProcessor(
			exporter,
			// // options ...
			// sdk.WithMaxQueueSize(2048),
			// sdk.WithExportMaxBatchSize(512),
			// sdk.WithExportTimeout(time.Second*30),
			// sdk.WithExportInterval(time.Second),
		)
	}

	return []sdk.LoggerProviderOption{
		sdk.WithProcessor(processor),
	}, nil
}

func init() {

	log.Register("stdout", withOptions)
	log.Register("stderr", withOptions)
	log.Register("file", withOptions)
}
