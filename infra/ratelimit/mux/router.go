package limitmux

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

// Router registers routes to be matched and dispatches a ratelimit.Handler.
type Router struct {
	// // Logs to debug requests
	// Logs *slog.Logger

	http   *mux.Router
	zones  ratelimit.NamedZones
	routes []*Route
}

func NewRouter() *Router {

	router := &Router{
		http:  mux.NewRouter(),
		zones: make(ratelimit.NamedZones),
	}

	// // Configurable Handler to be used when no route matches.
	// // This can be used to render your own 404 Not Found errors.
	// router.http.NotFoundHandler = (*routeHandler)(noroute) // http.HandlerFunc(nil)

	// // Configurable Handler to be used when the request method does not match the route.
	// // This can be used to render your own 405 Method Not Allowed errors.
	// router.http.MethodNotAllowedHandler = (*routeHandler)(noroute) // http.HandlerFunc(nil)

	return router
}

var _ ratelimit.Handler = (*Router)(nil)

// LimitRequest implements ratelimit.Handler interface
func (c *Router) LimitRequest(req *ratelimit.Request) (ratelimit.Status, error) {

	if req.Http == nil {
		// Not enough data to decide !
		// +Passthrough by default ..
		return ratelimit.Allow(req), nil
	}

	var match mux.RouteMatch
	var handler ratelimit.Handler
	if c.http.Match(req.Http, &match) {
		handler, _ = match.Handler.(ratelimit.Handler)
	}

	if handler == nil {
		// No route == no limit constraints !
		return ratelimit.Allow(req), nil
	}

	// PERFORM
	return handler.LimitRequest(req)
}

// ----------------------------------------------------------------------------
// Zone registry
// ----------------------------------------------------------------------------

// NewZone registers new zone that can be used with Route.Zone rule.
func (c *Router) NewZone(zone ratelimit.Zone) error {

	opts := zone.Options()

	// c.mx.Lock()
	c.zones[opts.Name] = zone
	// c.mx.Unlock()

	return nil
}

// ----------------------------------------------------------------------------
// Route factories
// ----------------------------------------------------------------------------

// NewRoute registers an empty route.
func (c *Router) NewRoute() *Route {
	// initialize a route with a copy of the parent router's configuration
	route := &Route{
		http:      c.http.NewRoute(),
		namedZone: c.zones,
	}
	route.http.Handler((*routeHandler)(route))

	c.routes = append(c.routes, route)
	return route
}

// Name registers a new route with a name.
// See Route.Name().
func (c *Router) Name(name string) *Route {
	return c.NewRoute().Name(name)
}

// Match attempts to match the given request against the router's registered routes.
func (c *Router) Match(req *http.Request) *Route {

	var match mux.RouteMatch
	var handler *routeHandler
	if c.http.Match(req, &match) {
		handler, _ = match.Handler.(*routeHandler)
	}
	if handler != nil {
		return (*Route)(handler)
	}
	// No match found
	return nil
}

// MatcherFunc registers a new route with a custom matcher function.
// See Route.MatcherFunc().
func (c *Router) MatcherFunc(match MatcherFunc) *Route {
	return c.NewRoute().MatcherFunc(match)
}

// Methods registers a new route with a matcher for HTTP methods.
// See Route.Methods().
func (c *Router) Methods(verb ...string) *Route {
	return c.NewRoute().Methods(verb...)
}

// Path registers a new route with a matcher for the URL path.
// See Route.Path().
func (c *Router) Path(tmpl string) *Route {
	return c.NewRoute().Path(tmpl)
}

// PathPrefix registers a new route with a matcher for the URL path prefix.
// See Route.PathPrefix().
func (c *Router) PathPrefix(tmpl string) *Route {
	return c.NewRoute().PathPrefix(tmpl)
}
