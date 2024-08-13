package internal

import (
	"os"
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
		if v, ok := e.GetEnvValue(key); ok {
			eval(v)
		}
	}
}

var (
	Environment = EnvReader{
		GetEnv:    os.Getenv,
		Namespace: "OTEL", // "WEBITEL",
	}
)
