package stdout

import (
	"context"
	"fmt"
	"io"
	"os"
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
		return nil, fmt.Errorf("unknown %s: scheme", scheme)
	}

	exporter, err := stdouttrace.New(
		// options
		// stdouttrace.WithoutTimestamps(),
		// stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(out),
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
