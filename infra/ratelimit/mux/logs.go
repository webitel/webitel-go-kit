package limitmux

import (
	"context"
	"log/slog"
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

type routeLog struct {
	route *Route
	slog.Handler
	prefix string
	groups []string
	attrs  []slog.Attr
	tree   treeDir
	exit   bool
}

var _ slog.Handler = (*routeLog)(nil)

func (c *routeLog) Handle(ctx context.Context, rec slog.Record) error {
	var cd string
	if !c.exit {
		cd = c.tree.path(false)
	} else {
		cd = c.tree.exit()
		c.exit = false // processed
	}
	if !strings.HasPrefix(rec.Message, cd) {
		rec.Message = cd + rec.Message
	}
	rec.AddAttrs(c.attrs...)
	return c.Handler.Handle(ctx, rec)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// The Handler owns the slice: it may retain, modify or discard it.
func (c *routeLog) WithAttrs(attrs []slog.Attr) slog.Handler {

	if len(attrs) == 0 {
		return c
	}

	c2 := new(routeLog)
	*c2 = *c // shallowcopy

	c2.attrs = slices.Clone(c2.attrs)
	c2.attrs = slices.Grow(c2.attrs, len(c2.attrs)+len(attrs))

	for _, add := range attrs {
		e := slices.IndexFunc(c2.attrs, func(set slog.Attr) bool {
			return set.Key == add.Key
		})
		if e < 0 {
			// not found
			c2.attrs = append(c2.attrs, add)
			continue
		}
		// rewrite
		c2.attrs[e] = add
		// continue
	}
	return c2
}

const (
	treeRoot = ""
	treePath = "│  " // "│   "
	treeFile = "├─ " // "├── "
	treeExit = "└─ " // "└── "
)

type treeDir struct {
	depth uint8
}

func (cd *treeDir) path(exit bool) (dir string) {
	if cd.depth == 0 {
		return ""
	}
	if cd.depth > 1 {
		dir = strings.Repeat(
			treePath, int(cd.depth)-1,
		)
	}
	if exit {
		dir += treeExit
	} else {
		dir += treeFile
	}
	return dir
}

func (cd *treeDir) open() (pwd string) {
	pwd = cd.path(false)
	cd.depth++
	return pwd
}

func (cd *treeDir) exit() (dir string) {
	dir = cd.path(true)
	if cd.depth > 0 {
		cd.depth--
	}
	return dir
}

func (c *routeLog) openDir() (pwd string) {
	return c.tree.open()
}

// first call  - begin exit
// second call - complete exit
func (c *routeLog) exitDir() {
	c.exit = true
}
