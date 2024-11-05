package internal

import (
	"errors"
	"mime"
	"net/url"
	"strings"
)

// Maybe rawDSN is of the form scheme:opts.
// (Scheme must be [a-zA-Z][a-zA-Z0-9+.-]*)
// If so, return scheme, opts; else return rawDSN, "".
func GetScheme(rawDSN string) (scheme, path string, err error) {
	for i := 0; i < len(rawDSN); i++ {
		c := rawDSN[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		// do nothing
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				// invalid scheme spec !
				return "", rawDSN, nil
			}
		case c == ':':
			if i == 0 {
				return "", "", errors.New("missing scheme")
			}
			return rawDSN[:i], rawDSN[i+1:], nil
		default:
			// we have encountered an invalid character,
			// so there is no valid scheme
			return "", rawDSN, nil
		}
	}
	// return "", rawDSN, nil
	return rawDSN, "", nil
}

// PraseDSN format(s):
// - scheme:path[;param=[;paramN=]]
// - scheme://path[?param=[&paramN=]]
func ParseDSN(rawDSN string) (path string, params map[string]string, err error) {
	_, path, err = GetScheme(rawDSN)
	if err != nil {
		return "", nil, err
	}
	var (
		opts  byte = ';' // ;param(s)=
		isURL bool       // false
	)
	path, isURL = strings.CutPrefix(path, "//")
	if isURL {
		opts = '?'                          // ?query=
		path, _, _ = strings.Cut(path, "#") // #fragment
		path, err = url.PathUnescape(path)
		if err != nil {
			return // "", nil, err
		}
	}
	var rest string
	path, rest, _ = strings.Cut(path, string(opts))
	rest = strings.TrimSpace(rest)
	if len(rest) == 0 {
		// no ";param=" spec
		return // path, nil, nil
	}
	if isURL {
		// scan: ?param1=[&paramN=]...
		query, qpe := url.ParseQuery(rest)
		if err = qpe; err != nil {
			return // path, nil, err
		}
		n := len(query)
		if n == 0 {
			return // path, nil, nil
		}
		params = make(map[string]string, n)
		for h, vs := range query {
			if n = len(vs); n > 0 {
				params[h] = vs[n-1]
			}
		}
		return // path, params, nil
	}
	// scan: ;param1=[;paramN=]
	const faketype = "text/plain;"
	_, params, err = mime.ParseMediaType(faketype + rest)
	if err != nil {
		return // path, nil, err
	}
	return // path, params, nil
}
