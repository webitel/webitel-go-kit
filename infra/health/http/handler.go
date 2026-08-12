package healthhttp

import (
	"log/slog"
	"net/http"
	"path"
	"sync/atomic"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// The endpoints, matched against the last path segment.
const (
	epLivez   = "livez"
	epReadyz  = "readyz"
	epHealthz = "healthz"
)

// allowedMethods is the Allow header on a 405; gorilla/mux does not gate.
const allowedMethods = "GET, HEAD"

type handler struct {
	reg   *health.Registry
	log   *slog.Logger
	fixed string // one endpoint only; empty means route on the path

	// Atomic: serve grants it, every request reads it.
	verboseAllowed atomic.Bool
}

func newHandler(reg *health.Registry, o options, verboseAllowed bool, fixed string) *handler {
	h := &handler{reg: reg, log: o.log, fixed: fixed}
	h.verboseAllowed.Store(verboseAllowed)

	return h
}

// Handler serves /livez, /readyz and /healthz, routed on the last path segment.
func Handler(r *health.Registry, opts ...Option) http.Handler {
	return newHandler(r, newOptions(opts), false, "")
}

// LivenessHandler serves /livez at whatever path it is mounted on.
func LivenessHandler(r *health.Registry, opts ...Option) http.Handler {
	return newHandler(r, newOptions(opts), false, epLivez)
}

// ReadinessHandler serves /readyz at whatever path it is mounted on.
func ReadinessHandler(r *health.Registry, opts ...Option) http.Handler {
	return newHandler(r, newOptions(opts), false, epReadyz)
}

// HealthHandler serves /healthz at whatever path it is mounted on.
func HealthHandler(r *health.Registry, opts ...Option) http.Handler {
	return newHandler(r, newOptions(opts), false, epHealthz)
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", allowedMethods)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	ep := h.fixed
	if ep == "" {
		ep = path.Base(r.URL.Path)
	}
	if ep != epLivez && ep != epReadyz && ep != epHealthz {
		http.NotFound(w, r)

		return
	}

	// Exactly one snapshot, or a 200 could carry a not_ready body.
	s := h.reg.Snapshot()

	code, token := readyVerdict(s)
	if ep == epLivez {
		code, token = liveVerdict(s)
	}

	if ep == epHealthz || (h.verboseAllowed.Load() && r.URL.Query().Has("verbose")) {
		writeJSON(w, code, s, h.log)

		return
	}

	writeText(w, code, token)
}

func readyVerdict(s health.Snapshot) (int, string) {
	code := http.StatusServiceUnavailable
	if s.State.Ready() {
		code = http.StatusOK
	}

	return code, s.State.String()
}

// liveVerdict is deliberately not derived from s.State: a draining process is
// stopping but not wedged, and an empty registry is unknown but alive.
func liveVerdict(s health.Snapshot) (int, string) {
	wedged := !s.SchedulerAlive
	for _, c := range s.Checks {
		wedged = wedged || (c.Group == health.GroupLiveness && c.Status == health.StatusFail)
	}

	if wedged {
		return http.StatusServiceUnavailable, NameWedged
	}

	return http.StatusOK, NameAlive
}
