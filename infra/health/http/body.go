package healthhttp

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// The /livez one-word bodies. Liveness has its own vocabulary because the state
// token contradicts the status line: a draining process is stopping, and alive.
const (
	// NameAlive is the /livez body on 200.
	NameAlive = "alive"

	// NameWedged is the /livez body on 503.
	NameWedged = "wedged"
)

// checkJSON is one check on the wire; declaration order is wire order.
type checkJSON struct {
	Name   string `json:"name"`
	Group  string `json:"group"`
	Status string `json:"status"`
	Since  string `json:"since,omitempty"`
}

type bodyJSON struct {
	Status string      `json:"status"`
	Checks []checkJSON `json:"checks"`
}

// detail is the only place that decides what a health response looks like. The
// status it reports is always the readiness state, never the /livez verdict.
//
// CheckResult.Err is never read here: check.go says it is for logs only.
func detail(s health.Snapshot) bodyJSON {
	out := bodyJSON{
		Status: s.State.String(),
		Checks: make([]checkJSON, 0, len(s.Checks)), // never nil: encodes as [], not null
	}

	for _, c := range s.Checks {
		cj := checkJSON{Name: c.Name, Group: c.Group.String(), Status: c.Status.String()}

		// since tracks the hysteresis status, not the rendered one: a stale
		// check renders unknown while carrying the time it went ok. It is zero
		// for the whole cold start and below FailThreshold, and omitted then.
		if !c.Since.IsZero() {
			cj.Since = c.Since.UTC().Format(time.RFC3339)
		}

		out.Checks = append(out.Checks, cj)
	}

	return out
}

func writeText(w http.ResponseWriter, code int, token string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	_, _ = io.WriteString(w, token+"\n") // net/http drops this for HEAD
}

// writeJSON marshals before writing: an encoder failing mid-stream would leave
// a truncated body behind an already-sent status line.
func writeJSON(w http.ResponseWriter, code int, s health.Snapshot, log *slog.Logger) {
	buf, err := json.Marshal(detail(s))
	if err != nil {
		log.Error("health/http: encoding the body failed", "err", err)
		http.Error(w, "encoding failed", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write(buf)
}
