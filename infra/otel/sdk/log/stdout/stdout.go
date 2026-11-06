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

	"github.com/webitel/webitel-go-kit/infra/otel/internal"
	"github.com/webitel/webitel-go-kit/infra/otel/sdk/log"
	"github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout/codec"
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

	path, params, err := internal.ParseDSN(rawDSN)
	if err != nil {
		return nil, err
	}

	var out io.WriteCloser
	switch scheme {
	case "stdout":
		out = os.Stdout
	case "stderr":
		out = os.Stderr
	case "file":
		{
			out, err = internal.FileDSN(rawDSN, "codec")
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("scheme %s: unknown", scheme)
	}
	if scheme != "file" && path != "" {
		return nil, fmt.Errorf("%s:%s: unexpected path %q, expect %[1]s:;codec= or %[1]s://?codec=", scheme, rawDSN, path)
	}

	var (
		noColor   = true
		codecOpts []codec.Option
		codecName = "text"
		codecFrom = "OTEL_LOGRECORD_CODEC"
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
			codecName = s
		}),
	)
	if s, ok := params["codec"]; ok {
		codecName, codecFrom = s, "DSN codec"
	}

	if noColor {
		codecOpts = append(codecOpts, codec.WithNoColor())
	}
	encoder := codec.NewCodec(codecName, out, codecOpts...)
	if encoder == nil {
		re := fmt.Errorf("invalid %s value %s: codec %[2]q not found", codecFrom, codecName)
		err = errors.Join(err, re)
		otel.Handle(re)
	}

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
