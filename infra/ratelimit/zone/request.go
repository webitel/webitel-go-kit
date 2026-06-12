package limitzone

import (
	"context"
	"slices"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

// LimitOptions defines additional options for requesting zone limits
// ratelimit.Handler.(Zone).LimitRequest(req, opts...)
type LimitOptions struct {

	// Salt to mix the primary zone.key with an optional secondary keys
	// for more specific route/zone relationship that needs to be limited
	//
	// Please, specify keys from most to least unique.
	Salt []ratelimit.Key

	// Rel as a prefix for a zone.Key.(Value) to be associated with ..
	// Is used to separate limit zone.Key.Value(s) into [sub]section(s)
	// depending on the route its defined within
	//
	// An empty string determine req.http.url.path to be associated with
	// Rel  *string

	// // Burst is the maximum number of tokens a bucket (-like algorithms) can hold,
	// // allowing a temporary, rapid spike in traffic to exceed the average rate limit instantly
	// Burst *uint32
	// // The Delay parameter specifies a limit at which excessive requests become delayed.
	// // Nil value stands for NoDelay option, i.e.
	// // Zero value i.e. all excessive requests are delayed.
	// // Otherwise all excessive (after N) requests are delayed.
	// Delay *uint32
}

// RequestPath is a ratelimit.ValueFunc that returns HTTP requested path.
// MAY be used with ratelimit.KeyFunc for ratelimit.Key registration.
func RequestPath(req *ratelimit.Request) ratelimit.Value {
	if req.Http != nil && req.Http.URL != nil {
		return req.Http.URL.Path
	}
	return ratelimit.Undefined
}

// RequestURI is a ratelimit.ValueFunc that returns HTTP requested URI.
// MAY be used with ratelimit.KeyFunc for ratelimit.Key registration.
func RequestURI(req *ratelimit.Request) ratelimit.Value {
	if req.Http != nil && req.Http.URL != nil {
		return req.Http.URL.RequestURI()
	}
	return ratelimit.Undefined
}

type LimitOption func(*LimitOptions)

// Adiitional [salt] key(s) for more unified of your limit.(zone).Key
func WithLimitKeys(salt ...ratelimit.Key) LimitOption {
	return func(opts *LimitOptions) {
		opts.Salt = slices.Grow(
			opts.Salt, len(salt),
		)
		for _, key := range salt {
			if key == nil {
				return
			}
			if slices.ContainsFunc(
				opts.Salt, func(has ratelimit.Key) bool {
					return ratelimit.EqualKeys(has, key)
				},
			) {
				continue // omit duplicate(s) ..
			}
			opts.Salt = append(opts.Salt, key)
		}
	}
}

func NewLimitOptions(opts ...LimitOption) LimitOptions {
	e := LimitOptions{}
	e.Setup(opts...)
	return e
}

func (e *LimitOptions) Setup(opts ...LimitOption) {
	for _, option := range opts {
		option(e)
	}
}

type ctxLimitOptions struct{}

func GetLimitOptions(ctx context.Context) LimitOptions {
	rv, _ := ctx.Value(ctxLimitOptions{}).(*LimitOptions)
	if rv != nil {
		return (*rv)
	}
	return LimitOptions{}
}

func WithLimitOptions(ctx context.Context, opts LimitOptions) context.Context {
	return context.WithValue(ctx, ctxLimitOptions{}, &opts)
}

// RequestKey returns key.Value and applies optionaly defined LimitOptions
func RequestKey(req *ratelimit.Request, key ratelimit.Key) ratelimit.Value {
	opts := GetLimitOptions(req.Context)
	key = ratelimit.MultiKey("+", key, opts.Salt...)
	return req.Get(key)
}
