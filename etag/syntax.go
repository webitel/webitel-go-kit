package etag

import (
	"strings"
)

// Name canonize s to alphanumeric lower code name
func Name(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	// for _, r := range s {
	// 	switch {
	// 	case 'a' <= r && r <= 'z':
	// 	case '0' <= r && r <= '9':
	// 	case '_' == r:
	// 	default:
	// 	}
	// }
	return s
}

// AddScope appends UNIQUE(+lower) names to scope
// and returns, optionaly new, scope slice
func AddScope(scope []string, names ...string) []string {
	if cap(scope) < len(scope)+len(names) {
		grow := make([]string, len(scope), len(scope)+len(names))
		copy(grow, scope)
		scope = grow
	}
	var name string
	for _, class := range names {
		name = Name(class) // CaseIgnoreMatch(!)
		if len(name) == 0 {
			continue
		}
		if !HasScope(scope, name) {
			scope = append(scope, name)
		}
	}
	return scope
}

func HasScope(scope []string, name string) bool {
	if len(scope) == 0 {
		return false // nothing(!)
	}
	name = Name(name) // CaseIgnoreMatch(!)
	if len(name) == 0 {
		return true // len(scope) != 0
	}
	e, n := 0, len(scope)
	for ; e < n && scope[e] != name; e++ {
		// break; match found !
	}
	return e < n
}
