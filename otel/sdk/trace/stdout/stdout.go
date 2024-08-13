package stdout

import (
	"context"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdk "go.opentelemetry.io/otel/sdk/trace"

	"github.com/webitel/webitel-go-kit/otel/internal"
	"github.com/webitel/webitel-go-kit/otel/sdk/trace"
)

func Options(ctx context.Context, rawDSN string) ([]sdk.TracerProviderOption, error) {
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
		output io.WriteCloser
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
			file, err := url.PathUnescape(dsn.EscapedPath())
			if err != nil || !filepath.IsAbs(file) {
				return nil, errors.Errorf("absolute filepath required")
			}
			output = &internal.FileWriter{
				Filename:   file,
				MaxSize:    100,   // Mb.
				MaxAge:     30,    // days
				MaxBackups: 3,     // log files
				LocalTime:  false, // UTC
				Compress:   true,
			}
		}
	default:
	}

	exporter, err := stdouttrace.New(
		// options
		// stdouttrace.WithoutTimestamps(),
		// stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(output),
	)
	if err != nil {
		return nil, err
	}
	return []sdk.TracerProviderOption{
		sdk.WithBatcher(exporter),
	}, nil
}

func init() {

	trace.Register("stdout", Options)
	trace.Register("stderr", Options)
	trace.Register("file", Options)
}
