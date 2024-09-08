package stdout

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdk "go.opentelemetry.io/otel/sdk/trace"

	"github.com/webitel/webitel-go-kit/otel/internal"
	"github.com/webitel/webitel-go-kit/otel/sdk/trace"
)

func Options(ctx context.Context, rawDSN string) ([]sdk.TracerProviderOption, error) {

	var scheme string
	colon := strings.IndexByte(rawDSN, ':')
	if colon < 0 {
		scheme, rawDSN = rawDSN, ""
	} else {
		scheme, rawDSN = rawDSN[0:colon], rawDSN[colon+1:]
	}
	scheme = strings.ToLower(scheme)

	var (
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
			file, err := url.PathUnescape(rawDSN)
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
				LocalTime:  false, // UTC
				Compress:   true,
			}
		}
	default:
		return nil, fmt.Errorf("unknown %s: scheme", scheme)
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
