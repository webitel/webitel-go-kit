package depenlog

import (
	"github.com/webitel/webitel-go-kit/pkg/logger"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
)

// WithComponent returns a sub-logger tagged with the originating component
// (e.g. WithComponent(l, "grpc")). It standardizes the per-component log split
// that was previously bespoke to im-account-service, using the shared
// semconv.ComponentKey so the tag is queryable the same way everywhere.
func WithComponent(l logger.Logger, name string) logger.Logger {
	return l.With(semconv.ComponentKey, name)
}
