package correlation

import (
	"context"
	"strings"

	"go.opentelemetry.io/contrib/processors/baggagecopy"
	"go.opentelemetry.io/otel/baggage"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

const keyPrefix = "webitel."

var allowedKeys = map[string]bool{
	string(UserIDKey): true,
}

// LogProcessor copies the baggage members whose key starts with "webitel." or
// is one of the allowed upstream keys, such as [UserIDKey], from the emit
// context into log record attributes.
//
// It changes the record in place, so it must be registered before the
// processors that export records.
type LogProcessor struct {
	*baggagecopy.LogProcessor
}

var _ sdklog.Processor = LogProcessor{}

func NewLogProcessor() LogProcessor {
	return LogProcessor{baggagecopy.NewLogProcessor(filter)}
}

// Enabled returns false, while the upstream processor returns true. The SDK
// Logger.Enabled is true when any processor is enabled, so a true here would
// enable every record and defeat the severity filtering of the export
// processors. This processor only decorates records that are emitted anyway.
func (LogProcessor) Enabled(context.Context, sdklog.EnabledParameters) bool {
	return false
}

func filter(m baggage.Member) bool {
	return strings.HasPrefix(m.Key(), keyPrefix) || allowedKeys[m.Key()]
}
