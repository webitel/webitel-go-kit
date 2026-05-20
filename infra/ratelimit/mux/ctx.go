package limitmux

import (
	"context"
	"slices"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

type ctxRouteKey struct{}

// returns COPY of the [req] with given [route] context binding ..
func requestWithRoute(req *ratelimit.Request, route *Route) *ratelimit.Request {
	vs := CurrentRoute(req) // parent
	if c := len(vs); c > 0 && vs[c-1] == route {
		return req // given [route] is the same as [current]
	}

	vs = slices.Clone(vs)
	vs = append(slices.Clone(vs), route)

	r2 := req.Clone(req.Context)
	r2.Http = r2.Http.WithContext(context.WithValue(
		r2.Http.Context(), ctxRouteKey{}, vs,
	))

	return r2

	// rl2 := (*req) // shallowcopy
	// ctx := rl2.Http.Context()
	// ctx = context.WithValue(ctx, ctxRouteKey{}, route)
	// rl2.Http = rl2.Http.WithContext(ctx)
	// return &rl2
}

// CurrentRoute returns the matched route for the current request, if any.
// This only works when called inside the handler of the matched route
// because the matched route is stored in the request context which is cleared
// after the handler returns.
func CurrentRoute(req *ratelimit.Request) []*Route {
	if vs := req.Http.Context().Value(ctxRouteKey{}); vs != nil {
		return vs.([]*Route)
	}
	return nil
}
