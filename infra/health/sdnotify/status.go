package sdnotify

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// statusStarting is the WithStartTimeout fallback line.
const statusStarting = "starting degraded"

// maxStatusBytes caps the STATUS= line; see truncate.
const maxStatusBytes = 1024

// statusText renders the STATUS= line. CheckResult.Err is never read here: it
// is for logs only.
func statusText(s health.Snapshot) string {
	if s.State == health.StateStopping {
		return health.NameStopping // a draining registry's check list is noise
	}
	if len(s.Checks) == 0 {
		return health.NameUnknown + ": no checks registered"
	}

	// pending, so the line does not read "unknown: unknown".
	var failing, pending, informational []string
	for _, c := range s.Checks {
		if c.Group == health.GroupInformational {
			// Never-run reads as pending, not as down.
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

// truncate caps the line on a rune boundary: systemd drops an oversized
// datagram whole, taking any READY=1 riding with it.
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

// sanitize maps control characters to '_' so STATUS= is always one line.
func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return '_'
		}

		return r
	}, name)
}
