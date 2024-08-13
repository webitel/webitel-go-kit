package stdout

import (
	"context"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	// "go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/webitel/webitel-go-kit/otel/internal"
	"github.com/webitel/webitel-go-kit/otel/sdk/log"
	"github.com/webitel/webitel-go-kit/otel/sdk/log/stdout/codec"
)

// Options
func Options(ctx context.Context, rawDSN string) ([]log.Option, error) {

	// dsn, err := url.ParseRequestURI(rawDSN)
	dsn, err := url.Parse(rawDSN)
	if err != nil {
		return nil, err
	}
	scheme := dsn.Scheme
	if scheme == "" {
		scheme = dsn.Path
	}
	// scheme, rawDSN, err := internal.GetScheme(rawDSN)
	// if err != nil {
	// 	return nil, err
	// }
	var (
		filename string
		output   io.WriteCloser
	)
	scheme = strings.ToLower(scheme)
	switch scheme {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "file":
		{
			// file := rawDSN
			filename, err = url.PathUnescape(dsn.EscapedPath())
			if err != nil || !filepath.IsAbs(filename) {
				return nil, errors.Errorf("absolute filepath required")
			}
			output = &internal.FileWriter{
				Filename:   filename,
				MaxSize:    100,   // Mb.
				MaxAge:     30,    // days
				MaxBackups: 3,     // log files
				LocalTime:  false, // UTC !
				Compress:   true,
			}
		}
	default:
	}

	var (
		encoder   Encoder // style ...
		codecOpts []codec.Option
	)
	internal.Environment.Apply(
		internal.EnvString("LOG_FORMAT_TIMESTAMP", func(s string) {
			// fmt.Printf("LOG_FORMAT_TIMESTAMP: [%s]\n", s)
			codecOpts = append(codecOpts, codec.WithTimestamps(s))
		}),
		internal.EnvString("LOG_FORMAT_INDENT", func(s string) {
			indent := s
			yes, err := strconv.ParseBool(s)
			if err == nil {
				if indent = ""; yes {
					indent = "\t"
				}
			}
			codecOpts = append(codecOpts,
				codec.WithPrittyPrint(indent),
			)
		}),
		internal.EnvString("LOG_FORMAT", func(s string) {
			encoder = codec.NewCodec(s, output, codecOpts...)
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
	return []sdklog.LoggerProviderOption{
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(
				exporter,
				// // options ...
				// sdklog.WithExportInterval(time.Second),
				// sdklog.WithExportMaxBatchSize(512),
				// sdklog.WithExportTimeout(time.Second*30),
				// sdklog.WithMaxQueueSize(2048),
			),
		),
	}, nil
}

func init() {

	log.Register("stdout", Options)
	log.Register("stderr", Options)
	log.Register("file", Options)
}
