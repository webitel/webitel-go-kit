package interceptor

import (
	"regexp"
	"strings"
)

var reg = regexp.MustCompile(`^(.*\.)`)

func splitFullMethodName(fullMethod string) (string, string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/") // remove leading slash
	if i := strings.Index(fullMethod, "/"); i >= 0 {
		return reg.ReplaceAllString(fullMethod[:i], ""), fullMethod[i+1:]
	}

	return "unknown", "unknown"
}
