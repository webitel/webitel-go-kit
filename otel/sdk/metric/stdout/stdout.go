package stdout

import (
	"context"
	"fmt"
	"io"
	"os"
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
		err error
		out io.WriteCloser
	)
	scheme = strings.ToLower(scheme)
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
		return nil, fmt.Errorf("unknown %q: scheme", scheme)
	}

	exporter, err := stdoutmetric.New(
		// options ...
		stdoutmetric.WithWriter(out),
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
