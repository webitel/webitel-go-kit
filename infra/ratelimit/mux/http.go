package limitmux

import (
	"cmp"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

// routeHandler implements basic [Route] Handler interfaces
type routeHandler Route

var _ http.Handler = (*routeHandler)(nil)

func (*routeHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
	// TODO nothing here ; just implement basic http.Handler interface
}

var _ ratelimit.Handler = (*routeHandler)(nil)

func (h *routeHandler) LimitRequest(req *ratelimit.Request) (res *ratelimit.Status, err error) {

	// ------------------------------------------------------------

	route := (*Route)(h)
	level := slog.LevelDebug

	// ------------------------------------------------------------ //

	req = requestWithRoute(req, route)

	inputLog := req.Logger
	debugLog, _ := inputLog.Handler().(*routeLog)
	if debugLog == nil {
		debugLog = &routeLog{
			route:   route,
			Handler: inputLog.Handler(),
			// prefix:  treeFile, // openDir
			// groups:  nil, // []string{},
			// attrs:   nil, // []slog.Attr{},
			// exit:    false,
		}
		req.Logger = slog.New(debugLog)
		defer func() {
			req.Logger = inputLog
		}()
	}

	begin := time.Now()

	defer func() {
		// track: timing
		spent := time.Since(begin).Round(time.Microsecond)

		status := "🗷 FORBIDDEN" // FOREVER ; NOT temporary
		if res == nil {
			status = "🗹 BYPASS"
		} else if res.Allowed > 0 {
			if res.Limit > 0 {
				status = "🗹 ALLOW" // SUCCESS ; applied & pass
			} else {
				status = "🗹 PASS" // PASSTHROUGH ; not_applied & pass
			}
		} else if res.Limit > 0 { // && res.Allowed == 0
			// Has [Limit] exhausted (temporary)
			status = "🗷 DENY"
		}
		params := []any{
			"status", res,
			// "time", spent,
		}
		if err != nil {
			level = slog.LevelError
			params = append(params,
				// "route.err"
				"err", err.Error(),
			)
		} else if re := res.Err(); re != nil {
			level = slog.LevelError
			params = append(params,
				// "limit.err"
				"err", re.Error(),
			)
		}
		params = append(params, "time", spent)

		debugLog.exitDir()     // begin
		req.Log(level, status, // fmt.Sprintf("LIMIT / %s", status), // "ROUTE / STATUS",
			params...,
		)
	}()

	warn := route.GetError()
	if warn != nil {
		// build route error
		level = slog.LevelWarn
		// params = append(params,
		// 	slog.String("route.err", warn.Error()),
		// )
	}

	req.Log(level, fmt.Sprintf("• ROUTE / %s %s", req.Http.Method, cmp.Or(route.opts.path, "/*")),
		// http.method=http.uri
		slog.String(
			// "http."+req.Http.Method, req.Http.URL.RequestURI(),
			cmp.Or(req.Http.URL.Scheme, "http")+"."+req.Http.Method, req.Http.URL.RequestURI(),
		),
		// route.* (deferred)
		"route", ratelimit.LogValue(func() slog.Value {
			args := make([]slog.Attr, 0, 2)
			if name := route.opts.name; name != "" {
				args = append(args, slog.String("name", name))
			}
			if tmpl := route.opts.path; tmpl != "" && tmpl != req.Http.URL.Path {
				args = append(args, slog.String("path", tmpl))
			}
			if warn != nil {
				args = append(args, slog.String("err", warn.Error()))
			}
			return slog.GroupValue(args...)
		}),
	)
	// for the NEXT log entries ..
	debugLog.openDir()

	// ------------------------------------------------------------ //

	if h.handler == nil {
		// Dummy Route ; No constraints ! ALLOW !
		return ratelimit.Allow(req), nil
	}

	// params := []any{
	// 	// http.method=url.path
	// 	// slog.String(req.Http.Method, req.Http.URL.Path),
	// 	slog.String("route.name", h.opts.name),
	// }

	// if route.opts.name == "" {
	// 	params = params[:len(params)-1] // trim, no value ..
	// }

	// stdlog := req.Logger
	// req.Logger = req.Logger.With(params...)
	// defer func() {
	// 	req.Logger = stdlog
	// }()

	// Invoke undelying ratelimit.Handler
	res, err = h.handler.LimitRequest(req)
	return // res, err
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
			// ratelimit.HttpWriteError(w, err)
			// return // terminate
		}

		if !ratelimit.HttpWriteStatus(w, status) {
			return // terminate
		}

		// passthrough ..
		next.ServeHTTP(w, r)

	})
}
