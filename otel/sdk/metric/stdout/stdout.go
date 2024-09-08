package stdout

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdk "go.opentelemetry.io/otel/sdk/metric"

	"github.com/webitel/webitel-go-kit/otel/internal"
	"github.com/webitel/webitel-go-kit/otel/sdk/metric"
)

// Options
func Options(ctx context.Context, rawDSN string) ([]metric.Option, error) {

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
	scheme = strings.ToLower(scheme)
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
		return nil, fmt.Errorf("unknown %q: scheme", scheme)
	}

	exporter, err := stdoutmetric.New(
		// options ...
		stdoutmetric.WithWriter(output),
		// stdoutmetric.WithPrettyPrint(),
		// stdoutmetric.WithoutTimestamps(),
	)
	if err != nil {
		return nil, err
	}
	return []metric.Option{
		sdk.WithReader(
			sdk.NewPeriodicReader(
				exporter,
				// sdk.WithTimeout(time.Second*30),
				// sdk.WithInterval(time.Second*60),
			),
		),
	}, nil
}

func init() {

	metric.Register("stdout", Options)
	metric.Register("stderr", Options)
	metric.Register("file", Options)
}
