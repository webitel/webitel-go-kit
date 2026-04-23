package limitzone

import (
	"fmt"
	"strings"
	"sync"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
	"github.com/webitel/webitel-go-kit/infra/ratelimit/zone/local"
)

// DriverFunc represents [Zone] factory method constructor
type DriverFunc func(dataSource string, options ratelimit.Options) (ratelimit.Zone, error)

// global drivers registry
var drivers = struct {
	mx     sync.Mutex
	scheme map[string]DriverFunc
}{
	scheme: make(map[string]DriverFunc),
}

// Register NEW driver for DSN (URL) scheme(s)..
func Register(driver DriverFunc, scheme ...string) {

	if driver == nil {
		panic(fmt.Errorf("ratelimit: register no driver"))
	}

	if len(scheme) == 0 {
		panic(fmt.Errorf("ratelimit: register driver no scheme"))
	}

	for _, in := range scheme {
		cn, _, err := GetScheme(in)
		if err != nil {
			panic(fmt.Errorf("ratelimit: register scheme error ; %v", err))
		}
		if cn != in {
			panic(fmt.Errorf("ratelimit: register scheme %q invalid", in))
		}
	}

	drivers.mx.Lock()
	defer drivers.mx.Unlock()

	for _, scheme := range scheme {
		scheme = strings.ToLower(scheme)
		drivers.scheme[scheme] = driver
	}
}

// GetDriver for given [scheme] name
func GetDriver(scheme string) DriverFunc {

	scheme = strings.ToLower(scheme)

	drivers.mx.Lock()
	defer drivers.mx.Unlock()

	ctor, _ := drivers.scheme[scheme]

	return ctor // nil?
}

// NewZone builds NEW [ratelimit.Zone] using the registered dataSource [scheme:] driver factory.
func NewZone(dataSource string, options ratelimit.Options) (ratelimit.Zone, error) {

	// split [scheme:][connString]
	scheme, _, err := GetScheme(dataSource)
	if err != nil {
		return nil, err
	}

	newZone := GetDriver(scheme)
	if newZone == nil {
		return nil, fmt.Errorf("ratelimit: scheme(%q) driver not supported", scheme)
	}

	if options.Key == nil {
		// FIXME: require ?
		options.Key = ratelimit.KeyValue("!nokey!", ratelimit.Undefined)
	}

	if !options.Rate.IsValid() {
		// helper zone ; MOVE to []
		return (*Forbidden)(&options), nil
	}

	zone, err := newZone(dataSource, options)
	if err != nil {
		// driver specific error
		return nil, err
	}

	return zone, nil
}

// Maybe [dataSource] is of the form [scheme[:]][path].
// (Scheme must be [a-zA-Z][a-zA-Z0-9+.-]*)
// If so, return scheme, opaque ; else return "", dataSource.
//
// https://cs.opensource.google/go/go/+/refs/tags/go1.26.1:src/net/url/url.go;l=369
func GetScheme(dataSource string) (scheme, opaque string, err error) {
	for i := 0; i < len(dataSource); i++ {
		c := dataSource[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		// do nothing
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				return "", dataSource, nil
			}
		case c == ':':
			if i == 0 {
				return "", "", fmt.Errorf("missing protocol scheme")
			}
			return dataSource[:i], dataSource[i+1:], nil
		default:
			// we have encountered an invalid character,
			// so there is no valid scheme
			return "", dataSource, nil
		}
	}
	// return "", connString, nil
	return dataSource, "", nil
}

func init() {
	// builtin & DEFAULT
	Register(local.New, "local", "memory", "")
}
