package limitmux

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/gorilla/mux"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
	limitzone "github.com/webitel/webitel-go-kit/infra/ratelimit/zone"
)

// Route stores information to match a HTTP request
type Route struct {
	http      *mux.Route           // HTTP matching rules
	opts      routeOptions         // debug options
	handler   ratelimit.Handler    // Rate-Limit request handler for the route.
	namedZone limitzone.NamedZones // registry of known zone(s)
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

type LimitOptions = limitzone.LimitOptions

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
	route, _ := c.handler.(*routeZones)
	if route == nil && c.handler != nil {
		// intercepted ; custom handler
		// zone could not be affected !
		return c
	}

	if route == nil && c.handler == nil {
		route := &routeZones{
			// route: c,
			// namedZone: c.namedZone,
			opts: map[string]LimitOptions{
				name: opts,
			},
			zone: []limitzone.Zone{zone},
		}
		c.handler = route
		return c
	}
	// override req.(zone).options
	_, exists := route.opts[name]
	if route.opts[name] = opts; !exists {
		route.zone = append(route.zone, zone)
	}

	return c
}

type RouteZone struct {
	Zone limitzone.Zone
	limitzone.LimitOptions
}

func (c *Route) GetZone() []RouteZone {

	group, _ := c.handler.(*routeZones)
	if group == nil {
		return nil
	}

	route := make([]RouteZone, len(group.zone))
	for e, zone := range group.zone {
		route[e] = RouteZone{
			Zone:         zone,
			LimitOptions: group.opts[zone.Options().Name],
		}
	}
	return route
}

// Zone Handler --------------------------------------------------------------

// routeZones Handler
type routeZones struct {
	// route *Route // top related HTTP route (matcher)
	// namedZone ratelimit.NamedZones // .well-known zones registry
	opts map[string]LimitOptions // named zone(s) request options
	zone []limitzone.Zone        // prepared (forward) requests
}

var _ ratelimit.Handler = (*routeZones)(nil)

// LimitRequest implements ratelimit.Handler interface.
func (h *routeZones) LimitRequest(req *ratelimit.Request) (res *ratelimit.Status, err error) {

	// res == Forbidden

	route := h.zone // h.routes()
	n := len(route) // [BULK] request(s) count

	if n == 0 {
		// NO [limit_req] directives ..
		// MAY be used as an EXCEPT of the "default" route !
		// ALLOW such configuration !
		return ratelimit.Allow(req), nil // OK, nil
	}

	if n == 1 {
		// simple case: single zone !
		return route[0].LimitRequest(req)
	}

	// defer func() {

	// 	args := []any{
	// 		// worst status as a single result of the group of (sub)requests
	// 		slog.Any("status", res),
	// 	}

	// 	level := slog.LevelDebug

	// 	if err != nil {
	// 		level = slog.LevelError
	// 		args = append(args, slog.String("err", err.Error()))
	// 	}

	// 	req.Log(
	// 		// final status result of (sub)requests group
	// 		level, "└── (group)", // "| = (group)",
	// 		args...,
	// 	)

	// }()

	// group of result(s)
	group := make([]*ratelimit.Status, 0, n)

	// TODO: sort top-worst zones
	ctx := req.Context
	defer func() {
		// back to original
		req.Context = ctx
	}()
	for _, zone := range route {

		// populate zone.LimitOptions for this request
		opts, _ := h.opts[zone.Options().Name]
		req.Context = limitzone.WithLimitOptions(ctx, opts)

		status, err := zone.LimitRequest(req)

		if err != nil {
			// Just LOG & check returned status !
			req.Log(slog.LevelWarn, "Limit handler error", "err", err.Error())
			continue
			// // Zone.Limiter.(Storage) error
			// return res, err
		}

		// Applied ?
		if status != nil {
			group = append(group, status)
		}
	}

	// Has affected limit zone(s) ?
	if len(group) == 0 {
		// NONE zone of group affected
		// -or- No ANY key(s) available !
		// FIXME: Not Applied ?
		return nil, nil
		// // res = Passthrough
		// res = ratelimit.Allow(req)
		// return res, nil
	}
	// [Dis]allowed first !
	sort.Sort(statusTopWorst(group))
	res = group[0]
	return res, nil
}

// Results of limit_req(s) group
// Sort.Interface by [top]=worst Status
type limitTopWorst []limitzone.Zone

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

// Swap swaps the elements with indexes i and j.
func (x statusTopWorst) Swap(i int, j int) {
	x[i], x[j] = x[j], x[i]
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
	xi, xj := x[i], x[j]
	// 1. More time to wait ..
	if xi.RetryAfter > 0 {
		return (xi.RetryAfter - xj.RetryAfter) >= 0
	}
	if xj.RetryAfter > 0 {
		// xi.RetryAfter <= 0 && xj.RetryAfter > 0
		// SWAP: [j] MUST sort BEFORE [i]
		return false
	}
	// 2. NOT Allowed ?
	if xi.Allowed == 0 {
		// [i] MUST sort BEFORE [j]
		return true
	}
	if xj.Allowed == 0 {
		// SWAP: [j] MUST sort BEFORE [i]
		return false
	}
	// 3. Has LESS tokens left ..
	if xi.Limit > 0 {
		if xj.Limit > 0 {
			// both has Limits, less tokens remaining on TOP ..
			return xi.Remaining <= xj.Remaining
		}
		// [i] has Limit -and- [j] dont !
		// [i] MUST sort BEFORE [j]
		return true
	}
	if xj.Limit > 0 {
		// [j] has Limit -and- [i] dont !
		return false
	}
	// It seems to be equal ; no need to swap records ..
	return true
}

// func (x statusTopWorst) Less(i int, j int) bool {
// 	a, b := x[i], x[j]
// 	return a.RetryAfter > b.RetryAfter || a.Remaining < b.Remaining // || a.Allowed < 1
// }
