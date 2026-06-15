package depenlog

import (
	stdlog "log"
	"net/http"
	"strings"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/logger"
)

// ErrorLog returns a *log.Logger suitable for http.Server.ErrorLog, forwarding
// each line the server emits to l at error level (component=http). This pulls
// net/http's internal errors into the unified schema.
func ErrorLog(l logger.Logger) *stdlog.Logger {
	return stdlog.New(errorLogWriter{log: WithComponent(l, "http")}, "", 0)
}

type errorLogWriter struct {
	log logger.Logger
}

func (w errorLogWriter) Write(p []byte) (int, error) {
	w.log.Error(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// Middleware logs each HTTP request (method, path, status, duration) through l,
// using the request context so trace_id/span_id are attached (component=http).
func Middleware(l logger.Logger) func(http.Handler) http.Handler {
	l = WithComponent(l, "http")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r)
			l.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

// statusRecorder captures the response status code for logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
