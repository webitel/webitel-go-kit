//
// Package ratelimit/mux implements a request router and dispatcher
// for matching incoming requests to their respective middleware handler.
//
// The name mux stands for "HTTP request multiplexer".
// Like the standard http.ServeMux, mux.Router matches incoming requests against a list of
// registered routes and calls a handler for the route that matches the URL or other conditions.
//
// This package is implemented as an HTTP middleware that you can embed in your frontline http.Handler
// for early interception and Rate-Limit constraint(s) checking before invoke http.Handler of your main route.

package limitmux
