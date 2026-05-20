package ratelimit

// Limit Handler interface
type Handler interface {
	// LimitRequest consumes token(-s) to limit resource overuse.
	// Returns the current usage Status of the Limit applied.
	//
	// Should return <nil> Status if no Limit was applied.
	// A non-nil error informs about an application problem, but does not restrict the request to perform.
	// To explicitly deny the request, return `&ratelimit.Status{Allowed: 0}` with no permission.
	LimitRequest(*Request) (*Status, error)

	// Zone(-s) use a [Key] option that classifies the origin of the Request.
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as Rate-Limit handlers.
// If [fn] is a function with the appropriate signature, HandlerFunc(fn) is a Handler that calls fn.
type HandlerFunc func(*Request) (*Status, error)

// HnadlerFunc implements Handler
var _ Handler = HandlerFunc(nil)

// LimitRequest implements a ratelimit.Handler interface
func (fn HandlerFunc) LimitRequest(req *Request) (*Status, error) {
	return fn(req)
}
