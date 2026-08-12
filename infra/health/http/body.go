package healthhttp

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// The /livez bodies; liveness has its own vocabulary.
const (
	NameAlive  = "alive"
	NameWedged = "wedged"
)

// checkJSON is one check on the wire.
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

// detail is the only place that shapes a health response. CheckResult.Err is
// never read here: it is for logs only.
func detail(s health.Snapshot) bodyJSON {
	out := bodyJSON{
		Status: s.State.String(),
		Checks: make([]checkJSON, 0, len(s.Checks)), // never nil: encodes as [], not null
	}

	for _, c := range s.Checks {
		cj := checkJSON{Name: c.Name, Group: c.Group.String(), Status: c.Status.String()}

		// since tracks the hysteresis status, not the rendered one, and is
		// zero until the first transition.
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

// writeJSON marshals before writing, so a failure cannot truncate the body.
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
