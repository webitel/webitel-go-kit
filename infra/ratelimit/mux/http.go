package limitmux

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

// routeHandler implements basic [Route] interfaces
type routeHandler Route

var _ http.Handler = (*routeHandler)(nil)

func (*routeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO nothing here ; just implement basic http.Handler interface
}

var _ ratelimit.Handler = (*routeHandler)(nil)

func (h *routeHandler) LimitRequest(req *ratelimit.Request) (ratelimit.Status, error) {
	if h.handler != nil {
		// log: route hit ..
		h.debugRequest(req)
		// invoke undelying ratelimit.Handler
		return h.handler.LimitRequest(req)
	}
	// Dummy Route ; No constraints ! ALLOW !
	return ratelimit.Allow(req), nil
}

func (h *routeHandler) debugRequest(req *ratelimit.Request) {

	level := slog.LevelDebug
	route := (*Route)(h)
	warn := route.GetError()
	// ctx := req.Context

	if warn != nil {
		// build route error
		level = slog.LevelWarn
	}

	attrs := []any{
		// method=path
		slog.String(req.Http.Method, req.Http.URL.Path),
		slog.String("route.name", h.opts.name),
	}
	if route.opts.name == "" {
		attrs = attrs[:len(attrs)-1] // trim, no value ..
	}

	// populate route.attrs...
	req.Logger = req.Logger.With(attrs...)

	req.Log(
		// Debug: route hit ..
		level, "| • (route)",
		// route.* (deferred)
		"route", ratelimit.LogValue(func() slog.Value {
			args := make([]slog.Attr, 0, 2)
			// if route.opts.name != "" {
			// 	args = append(args, slog.String("name", h.opts.name))
			// }
			if tmpl := route.opts.path; tmpl != "" && tmpl != req.Http.URL.Path {
				args = append(args, slog.String("path", tmpl))
			}
			if warn != nil {
				args = append(args, slog.String("err", warn.Error()))
			}
			return slog.GroupValue(args...)
		}),
	)

}

// HttpMiddleware intercepts the HTTP request
// and checks the appropriate Rate-Limit constraints
func (c *Router) HttpMiddleware(next http.Handler) http.Handler {

	if next == nil {
		next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK) // (200) OK
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var match mux.RouteMatch
		var handler ratelimit.Handler
		if c.http.Match(r, &match) {
			handler, _ = match.Handler.(ratelimit.Handler)
		}

		if handler == nil {
			// No route == no constraints !
			next.ServeHTTP(w, r)
			return // complete
		}

		ctx := r.Context()
		req := ratelimit.NewRequest(
			ctx, func(req *ratelimit.Request) {
				req.Http = r
			},
		)

		// PERFORM
		status, err := handler.LimitRequest(&req)

		if err != nil {
			ratelimit.HttpWriteError(w, err)
			return // terminate
		}

		if !ratelimit.HttpWriteStatus(w, status) {
			return // terminate
		}

		// passthrough ..
		next.ServeHTTP(w, r)

	})
}
