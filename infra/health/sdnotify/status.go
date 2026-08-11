package sdnotify

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// statusStarting is the WithStartTimeout fallback line, in the design's own
// wording.
const statusStarting = "starting degraded"

// maxStatusBytes caps the STATUS= line; see truncate.
const maxStatusBytes = 1024

// statusText renders the STATUS= line for one snapshot. It is the only place
// that decides what STATUS= looks like.
//
// CheckResult.Err is never read here: check.go says it is for logs only.
func statusText(s health.Snapshot) string {
	if s.State == health.StateStopping {
		return health.NameStopping // a draining registry's check list is noise
	}
	if len(s.Checks) == 0 {
		return health.NameUnknown + ": no checks registered"
	}

	// pending rather than unknown, so the line does not read "unknown: unknown".
	var failing, pending, informational []string
	for _, c := range s.Checks {
		if c.Group == health.GroupInformational {
			// Split the same way the counting groups are: an informational
			// check that has never run is pending, not "informational [s3]",
			// which an operator reads as "s3 is down".
			if c.Status == health.StatusFail {
				informational = append(informational, sanitize(c.Name))
			}
			if c.Status == health.StatusUnknown {
				pending = append(pending, sanitize(c.Name))
			}

			continue
		}
		if c.Status == health.StatusFail {
			failing = append(failing, sanitize(c.Name))
		}
		if c.Status == health.StatusUnknown {
			pending = append(pending, sanitize(c.Name))
		}
	}

	var parts []string
	if len(failing) > 0 {
		parts = append(parts, fmt.Sprintf("failing [%s]", strings.Join(failing, " ")))
	}
	if len(pending) > 0 {
		parts = append(parts, fmt.Sprintf("pending [%s]", strings.Join(pending, " ")))
	}
	if len(informational) > 0 {
		parts = append(parts, fmt.Sprintf("informational [%s]", strings.Join(informational, " ")))
	}
	if len(parts) == 0 {
		return truncate(s.State.String())
	}

	return truncate(s.State.String() + ": " + strings.Join(parts, "; "))
}

// truncate caps the line on a rune boundary. systemd's PID 1 drops an
// oversized notify datagram whole, taking the READY=1 riding with it, so a
// consumer registering enough checks would silently lose readiness rather than
// merely lose the status text.
func truncate(s string) string {
	if len(s) <= maxStatusBytes {
		return s
	}

	const ell = "..."

	cut := maxStatusBytes - len(ell)
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}

	return s[:cut] + ell
}

// sanitize maps control characters to '_' so STATUS= is always one line: a
// newline in a check name would otherwise be parsed as a second assignment.
func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return '_'
		}

		return r
	}, name)
}
