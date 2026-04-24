package limitmux

import (
	"slices"
	"strings"
)

// used to remember options applied
type routeOptions struct {
	name string
	path string
	verb []string
}

func (e *routeOptions) setName(name string) {
	if e.name != name {
		e.name = name
	}
}

func (e *routeOptions) addMethods(verb []string) {
	n := len(verb)
	if n == 0 {
		return
	}
	if e.verb == nil {
		for k, v := range verb {
			verb[k] = strings.ToUpper(v)
		}
		e.verb = verb
		return
	}
	// reset := false
	for _, v := range verb {
		v = strings.ToUpper(v)
		if slices.Contains(e.verb, v) {
			continue // already exists ..
		}
		e.verb = append(e.verb, v)
		// reset = true
	}
}

func (e *routeOptions) setPath(tmpl string) {
	if e.path != tmpl {
		e.path = tmpl
	}
}

func (e *routeOptions) setPathPrefix(tmpl string) {
	if !strings.HasSuffix(tmpl, "*") {
		tmpl += "*"
	}
	e.setPath(tmpl)
}
