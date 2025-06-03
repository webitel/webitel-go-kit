package internal

import (
	"os"
	"strconv"
	"strings"
)

type EnvReader struct {
	GetEnv    func(string) string
	Namespace string
}

func keyWithNamespace(space, name string) string {
	if space == "" {
		return name
	}
	return (space + "_" + name)
}

// GetEnvValue gets an OTLP environment variable value of the specified key
// using the GetEnv function.
// This function prepends the OTLP specified namespace to all key lookups.
func (e *EnvReader) GetEnvValue(key string) (string, bool) {
	v := strings.TrimSpace(e.GetEnv(
		keyWithNamespace(e.Namespace, key),
	))
	return v, v != ""
}

type EnvOption func(*EnvReader)

func (c *EnvReader) Apply(opts ...EnvOption) {
	for _, opt := range opts {
		opt(c)
	}
}

func EnvString(key string, eval func(string)) EnvOption {
	return func(e *EnvReader) {
		if s, ok := e.GetEnvValue(key); ok {
			// log.Printf("[%s] = [%s]", keyWithNamespace(e.Namespace, key), s)
			if u, err := strconv.Unquote(s); err == nil {
				s = u // unquoted(!)
			}
			// // unquote
			// for n := len(s); n > 1; n = len(s) {
			// 	switch quote := s[0]; quote {
			// 	case '"', '\'', '`':
			// 		if s[n-1] == quote {
			// 			s = s[1 : n-1]
			// 			continue
			// 		}
			// 	}
			// 	break
			// }
			// log.Printf("[%s] = [%s]", keyWithNamespace(e.Namespace, key), s)
			eval(s)
		}
	}
}

var (
	Environment = EnvReader{
		GetEnv:    os.Getenv,
		Namespace: "OTEL",
	}
)
