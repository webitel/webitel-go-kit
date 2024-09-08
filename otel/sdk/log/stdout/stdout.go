package stdout

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
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
		err    error
		file   string // [/][path/to/]filename
		output io.WriteCloser
	)
	switch scheme {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "file":
		{
			rawDSN, _ = strings.CutPrefix(rawDSN, "//")
			file, err = url.PathUnescape(rawDSN)
			// if err != nil || !filepath.IsAbs(filename) {
			// 	return nil, fmt.Errorf("absolute filepath required")
			// }
			if err == nil {
				switch filepath.Base(file) {
				case ".", string(filepath.Separator):
					err = fmt.Errorf("file:name expected")
				}
			}
			if err == nil {
				file, err = filepath.Abs(file)
			}
			if err != nil {
				return nil, err
			}
			output = &internal.FileWriter{
				Filename:   file,
				MaxSize:    100,   // Mb.
				MaxAge:     30,    // days
				MaxBackups: 3,     // log files
				LocalTime:  false, // UTC !
				Compress:   true,
			}
		}
	default:
		return nil, fmt.Errorf("scheme %s: unknown", scheme)
	}

	var (
		encoder   Encoder // formatter
		codecOpts []codec.Option
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
		internal.EnvString("LOGRECORD_CODEC", func(s string) {
			encoder = codec.NewCodec(s, output, codecOpts...)
			if encoder == nil {
				re := fmt.Errorf("invalid OTEL_LOGS_FORMAT value %s: codec %[1]q not found", s)
				err = errors.Join(err, re)
				otel.Handle(re)
				return
			}
		}),
	)

	// exporter, err := stdoutlog.New(
	// 	// options ...
	// 	stdoutlog.WithWriter(output),
	// 	// stdoutlog.WithPrettyPrint(),
	// 	// stdoutlog.WithoutTimestamps(),
	// )
	exporter, err := New(
		// options ...
		WithWriter(output),
		WithCodec(encoder),
		// stdoutlog.WithPrettyPrint(),
		// stdoutlog.WithoutTimestamps(),
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
