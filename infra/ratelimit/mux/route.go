package limitmux

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/gorilla/mux"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

// Route stores information to match a HTTP request
type Route struct {
	http      *mux.Route           // HTTP matching rules
	opts      routeOptions         // debug options
	handler   ratelimit.Handler    // Rate-Limit request handler for the route.
	namedZone ratelimit.NamedZones // registry of known zone(s)
	// Error resulted from building a route.
	err error
}

// ----------------------------------------------------------------------------
// Route attributes
// ----------------------------------------------------------------------------

// GetError returns an error resulted from building the route, if any.
func (c *Route) GetError() error {
	if c.err != nil {
		return c.err
	}
	if c.http != nil {
		return c.http.GetError()
	}
	return nil
}

// Handler --------------------------------------------------------------------

// Handler sets a custom [ratelimit.Handler] for the route.
func (c *Route) Handler(hook ratelimit.Handler) *Route {
	if c.GetError() == nil {
		c.handler = hook
	}
	return c
}

// Name -----------------------------------------------------------------------

// Name sets the name for the route, used to build URLs.
// It is an error to call Name more than once on a route.
func (c *Route) Name(name string) *Route {
	if c.GetError() == nil {
		_ = c.http.Name(name)
		c.opts.setName(name)
	}
	return c
}

// GetName returns the name for the route, if any.
func (c *Route) GetName() string {
	return c.http.GetName()
}

// MatcherFunc ----------------------------------------------------------------

// // MatcherFunc is the function signature used by custom matchers.
// type MatcherFunc func(*http.Request, *RouteMatch) bool

// // Match returns the match for a given request.
// func (m MatcherFunc) Match(r *http.Request, match *RouteMatch) bool {
// 	return m(r, match)
// }

// shorthand data types
type (
	RouteMatch  = mux.RouteMatch
	MatcherFunc = mux.MatcherFunc
)

// MatcherFunc adds a custom function to be used as request matcher.
func (c *Route) MatcherFunc(match MatcherFunc) *Route {
	_ = c.http.MatcherFunc(match)
	return c
}

// Methods --------------------------------------------------------------------

// Methods adds a matcher for HTTP methods.
// It accepts a sequence of one or more methods to be matched, e.g.:
// "GET", "POST", "PUT".
func (c *Route) Methods(verb ...string) *Route {
	if len(verb) == 0 {
		// NOTE: zero-lenght / no [verb] arguments cause
		// gorilla/mux.methodMatcher to NEVER match requests !
		return c // not affected ; skip ..
	}
	if c.GetError() == nil {
		_ = c.http.Methods(verb...)
		c.opts.addMethods(verb)
	}
	return c
}

// Path -----------------------------------------------------------------------

// Path adds a matcher for the URL path.
// It accepts a template with zero or more URL variables enclosed by {}. The
// template must start with a "/".
// Variables can define an optional regexp pattern to be matched:
//
// - {name} matches anything until the next slash.
//
// - {name:pattern} matches the given regexp pattern.
//
// For example:
//
//	r := mux.NewRouter().NewRoute()
//	r.Path("/products/").Handler(ProductsHandler)
//	r.Path("/products/{key}").Handler(ProductsHandler)
//	r.Path("/articles/{category}/{id:[0-9]+}").
//	  Handler(ArticleHandler)
//
// Variable names must be unique in a given route. They can be retrieved
// calling mux.Vars(request).
func (c *Route) Path(tmpl string) *Route {
	if tmpl == "" {
		// not affected ; match ANY path !
		return c
	}
	_ = c.http.Path(tmpl)
	c.opts.setPath(tmpl)
	return c
}

// PathPrefix -----------------------------------------------------------------

// PathPrefix adds a matcher for the URL path prefix. This matches if the given
// template is a prefix of the full URL path. See Route.Path() for details on
// the tpl argument.
//
// Note that it does not treat slashes specially ("/foobar/" will be matched by
// the prefix "/foo") so you may want to use a trailing slash here.
//
// Also note that the setting of Router.StrictSlash() has no effect on routes
// with a PathPrefix matcher.
func (c *Route) PathPrefix(tmpl string) *Route {
	if tmpl == "" {
		// not affected ; match ANY path !
		return c
	}
	_ = c.http.PathPrefix(tmpl)
	c.opts.setPathPrefix(tmpl)
	return c
}

// Zone -----------------------------------------------------------------------

// LimitRequest Options
// for future expansion purpose
type LimitOptions struct {
	// // Zone.(Handler) for examination ..
	// Zone Zone
	// // Burst is the maximum number of tokens a bucket (-like algorithms) can hold,
	// // allowing a temporary, rapid spike in traffic to exceed the average rate limit instantly
	// Burst *uint32
	// // The Delay parameter specifies a limit at which excessive requests become delayed.
	// // Nil value stands for NoDelay option, i.e.
	// // Zero value i.e. all excessive requests are delayed.
	// // Otherwise all excessive (after N) requests are delayed.
	// Delay *uint32
}

// Zone adds a [sub]request to the route to check the limit of the given [name] zone with options.
func (c *Route) Zone(name string, opts LimitOptions) *Route {

	if c.err != nil {
		return c
	}

	zone, ok := c.namedZone[name]
	if !ok || zone == nil {
		c.err = fmt.Errorf("route: zone %q not found", name)
		// zone: not found
		return c
	}

	// check Handler implementation ..
	route, _ := c.handler.(*routeLimitGroup)
	if route == nil && c.handler != nil {
		// intercepted ; custom handler
		// zone could not be affected !
		return c
	}

	if route == nil && c.handler == nil {
		route := &routeLimitGroup{
			// route: c,
			// namedZone: c.namedZone,
			limitOpts: map[string]LimitOptions{
				name: opts,
			},
			routeZone: []ratelimit.Zone{zone},
		}
		c.handler = route
		return c
	}
	// override req.(zone).options
	_, exists := route.limitOpts[name]
	if route.limitOpts[name] = opts; !exists {
		route.routeZone = append(route.routeZone, zone)
	}

	return c
}

// Zone Handler --------------------------------------------------------------

// routeLimitGroup Handler
type routeLimitGroup struct {
	// route *Route // top related HTTP route (matcher)
	// namedZone ratelimit.NamedZones // .well-known zones registry
	limitOpts map[string]LimitOptions // named zone(s) request options
	routeZone []ratelimit.Zone        // prepared (forward) requests
}

// // prepare (build) configuration(s) for handler
// func (h *routeLimitGroup) routes() []ratelimit.Zone {
// 	if h.routeZone != nil {
// 		return h.routeZone
// 	}
// 	// build ; once ..
// 	routes := make([]ratelimit.Zone, 0, len(h.limitOpts))
// 	for name, opts := range h.limitOpts {
// 		_ = opts // not affected for now ...
// 		zone, _ := h.namedZone[name]
// 		if zone != nil {
// 			// defined !
// 			routes = append(routes, zone)
// 		}
// 	}
// 	// sort: worst zone.rate on top ..
// 	sort.Sort(limitTopWorst(routes))
// 	h.routeZone = routes
// 	return h.routeZone
// }

var _ ratelimit.Handler = (*routeLimitGroup)(nil)

// LimitRequest implements ratelimit.Handler interface.
func (h *routeLimitGroup) LimitRequest(req ratelimit.Request) (res ratelimit.Status, err error) {

	// res == Forbidden

	routes := h.routeZone // h.routes()
	n := len(routes)      // [BULK] request(s) count

	if n == 0 {
		// NO [limit_req] directives ..
		return ratelimit.Allow(&req), nil // OK, nil
	}

	if n == 1 {
		// simple case ..
		return routes[0].LimitRequest(req)
	}

	defer func() {

		args := []any{
			// worst status as a single result of the group of (sub)requests
			slog.Any("limit", &res),
		}

		level := slog.LevelDebug

		if err != nil {
			level = slog.LevelError
			args = append(args, slog.Any("err", err))
		}

		req.Log(
			// final status result of (sub)requests group
			level, "| = (group)",
			args...,
		)

	}()

	// group of result(s)
	group := make([]*ratelimit.Status, 0, n)

	// TODO: sort top-worst zones
	for _, zone := range routes {
		// // populate LimitOptions
		// req.setup(sub.RequestOption)
		// // Resolve Key.(Value) once for Group of Request(s) ..
		// req.Key = env.Key(sub.Zone.Options().Key).Value(req.Context)
		// // zone.Value = groupEnv(zone.Key, zone.Value)
		status, err := zone.LimitRequest(req)

		if err != nil {
			// Zone.Limiter.(Storage) error
			return res, err
		}

		group = append(group, &status)
	}

	// Has affected limit zone(s) ?
	if len(group) == 0 {
		// NONE zone of group affected
		// -or- No ANY key(s) available !
		// res = Passthrough
		res = ratelimit.Allow(&req)
		return res, nil
	}
	// [Dis]allowed first !
	sort.Sort(statusTopWorst(group))
	res = *(group[0])
	return res, nil
}

// Results of limit_req(s) group
// Sort.Interface by [top]=worst Status
type limitTopWorst []ratelimit.Zone

var _ sort.Interface = limitTopWorst(nil)

// Len is the number of elements in the collection.
func (vs limitTopWorst) Len() int {
	return len(vs)
}

// Less reports whether the element with index i
// must sort before the element with index j.
//
// If both Less(i, j) and Less(j, i) are false,
// then the elements at index i and j are considered equal.
// Sort may place equal elements in any order in the final result,
// while Stable preserves the original input order of equal elements.
//
// Less must describe a transitive ordering:
//   - if both Less(i, j) and Less(j, k) are true, then Less(i, k) must be true as well.
//   - if both Less(i, j) and Less(j, k) are false, then Less(i, k) must be false as well.
//
// Note that floating-point comparison (the < operator on float32 or float64 values)
// is not a transitive ordering when not-a-number (NaN) values are involved.
// See Float64Slice.Less for a correct implementation for floating-point values.
func (vs limitTopWorst) Less(i int, j int) bool {
	a, b := vs[i].Options().Rate, vs[j].Options().Rate
	x, y := a.Every(), b.Every()
	return y == 0 || x <= y
}

// Swap swaps the elements with indexes i and j.
func (vs limitTopWorst) Swap(i int, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

// Results of limit_req(s) group
// Sort.Interface by [top]=worst Status
type statusTopWorst []*ratelimit.Status

var _ sort.Interface = statusTopWorst(nil)

// Len is the number of elements in the collection.
func (x statusTopWorst) Len() int {
	return len(x)
}

// Less reports whether the element with index i
// must sort before the element with index j.
//
// If both Less(i, j) and Less(j, i) are false,
// then the elements at index i and j are considered equal.
// Sort may place equal elements in any order in the final result,
// while Stable preserves the original input order of equal elements.
//
// Less must describe a transitive ordering:
//   - if both Less(i, j) and Less(j, k) are true, then Less(i, k) must be true as well.
//   - if both Less(i, j) and Less(j, k) are false, then Less(i, k) must be false as well.
//
// Note that floating-point comparison (the < operator on float32 or float64 values)
// is not a transitive ordering when not-a-number (NaN) values are involved.
// See Float64Slice.Less for a correct implementation for floating-point values.
func (x statusTopWorst) Less(i int, j int) bool {
	a, b := x[i], x[j]
	return a.RetryAfter > b.RetryAfter || a.Remaining < b.Remaining // || a.Allowed < 1
}

// Swap swaps the elements with indexes i and j.
func (x statusTopWorst) Swap(i int, j int) {
	x[i], x[j] = x[j], x[i]
}
