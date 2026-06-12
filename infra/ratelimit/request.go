package ratelimit

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Request to use Limit token(s)
type Request struct {

	// Environment used to cache resolved keys
	Env Env

	// Date of request / event
	Date time.Time

	// Call Options

	// Cost of this Request ( in token(-s) count ). Default: 1.
	Cost uint32

	// Burst *uint32 // OPTIONAL. Burst for *-bucket like .Zone.Algo strategies. Default: 1.
	// Delay *uint32 // OPTIONAL. Delay request(s) after (stat.Limit - stat.Remaining) < N < stat.Limit

	// HTTP (-like) request source
	Http *http.Request

	// // Sets the status code to return in response to rejected requests.
	// Code int // https://nginx.org/en/docs/http/ngx_http_limit_req_module.html#limit_req_status

	// Logger for debugging
	Logger *slog.Logger

	// Context associated with this Request.
	Context context.Context
}

// RequestOption to configure the Request
type RequestOption func(req *Request)

// NewRequest with options..
func NewRequest(ctx context.Context, opts ...RequestOption) Request {
	req := Request{
		Env:     make(Env),
		Date:    time.Now(),
		Logger:  noLogs,
		Context: ctx,
	}
	req.setup(opts...)
	return req
}

func (req *Request) setup(opts ...RequestOption) {
	for _, option := range opts {
		option(req)
	}
	// normalize ..
	req.Cost = max(1, req.Cost)
}

// Clone returns a deep copy of [req] with its context changed to ctx.
// The provided [ctx] must be non-nil.
func (req *Request) Clone(ctx context.Context) *Request {
	r2 := new(Request)
	*r2 = *req
	r2.Context = ctx
	return r2
}

// Get [env] Key.(Value) for this Request
func (req *Request) Get(env Key) Value {
	// return req.Env.Get(req.Context, key)
	return req.Env.Key(env).Value(req)
}

// Set [env] Key.(Value) for this Request
func (req *Request) Set(env Key) {
	// determine & cache [env] Value
	_ = req.Get(env)
}
