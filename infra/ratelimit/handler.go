package ratelimit

// Limit Handler
type Handler interface {
	// LimitRequest consumes a token(-s) to limit resource overuse.
	// Zone(-s) use a [Key] option that classifies the origin of the Request.
	LimitRequest(Request) (Status, error)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as Rate-Limit handlers.
// If [fn] is a function with the appropriate signature, HandlerFunc(fn) is a Handler that calls fn.
type HandlerFunc func(Request) (Status, error)

// HnadlerFunc implements Handler
var _ Handler = HandlerFunc(nil)

// LimitRequest implements a ratelimit.Handler interface
func (fn HandlerFunc) LimitRequest(req Request) (Status, error) {
	return fn(req)
}
