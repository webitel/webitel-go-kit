package depenlog

import (
	"bufio"
	"errors"
	stdlog "log"
	"net"
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
			l.InfoContext(r.Context(), "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

// statusRecorder captures the response status code for logging. It forwards the
// optional ResponseWriter interfaces (Flusher/Hijacker/Pusher) and exposes the
// wrapped writer via Unwrap so middleware does not break streaming, WebSocket
// upgrades, or HTTP/2 push.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap exposes the wrapped writer to http.ResponseController and any other
// middleware that unwraps the chain.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// Flush forwards to the wrapped writer when it supports streaming; otherwise it
// is a no-op, matching net/http's behavior for non-flushable writers.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack forwards to the wrapped writer so connection-upgrade handlers
// (e.g. WebSocket) keep working; it errors if the writer is not a Hijacker.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("depenlog: ResponseWriter does not support Hijack")
}

// Push forwards HTTP/2 server push to the wrapped writer, or reports that it is
// unsupported.
func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := r.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}
