package ratelimit

import (
	"net/http"

	"github.com/webitel/webitel-go-kit/infra/errors"
	"google.golang.org/grpc/status"
)

var (
	// HTTP: (#403) Forbidden
	// GRPC: (#7) PERMISSION_DENIED
	ErrForbidden = errors.Forbidden(
		errors.Status("FORBIDDEN"),
		errors.Message("service: forbidden request"),
	)
	// HTTP: (#429) Too Many Requests
	// GRPC: (#8) RESOURCE_EXHAUSTED
	ErrFloodWait = errors.New(
		errors.Code(http.StatusTooManyRequests),
		errors.Status("FLOOD_WAIT"), // RATE_LIMIT // TOO_MANY_REQUESTS
		errors.Message("service: too many requests"),
		// 429, // http.StatusTooManyRequests    // ~ grpc.codes.ResourceExhausted ?
		// 503, // http.StatusServiceUnavailable // ~ grpc.codes.Unavailable !
	)
)

// Error of Rate-Limit status
type Error struct {
	res Status
}

// Rate-Limit Status associated
func (e *Error) Status() Status {
	return e.res
}

// Error message details
func (e *Error) Error() string {
	return ErrFloodWait.Message
}

// GRPCStatus implements grpc.FromError(err).(*Status)
func (e *Error) GRPCStatus() *status.Status {
	return ErrFloodWait.GRPCStatus()
}
