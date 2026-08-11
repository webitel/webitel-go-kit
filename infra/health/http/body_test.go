package healthhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// sentinel is a string that could not occur by accident, standing in for the
// kind of connection error that must never reach a response body.
const sentinel = "dial tcp 10.0.0.5:5432: connect: connection refused / zzz-sentinel-9c1f"

func rfc3339(t *testing.T, s string) time.Time {
	t.Helper()

	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parsing %q: %v", s, err)
	}

	return ts
}

func TestHealthzGolden(t *testing.T) {
	// Snapshots are hand-built so the timestamps are fixed and the bytes are
	// golden. Compare strings, not decoded maps: field order and omitempty are
	// the contract.
	for _, tt := range []struct {
		name string
		snap health.Snapshot
		want string
	}{
		{
			name: "D1 example",
			snap: health.Snapshot{
				State: health.StateDegraded,
				Checks: []health.CheckResult{
					{Name: "grpc", Group: health.GroupCritical, Status: health.StatusOK,
						Since: rfc3339(t, "2026-08-08T07:04:11Z")},
					{Name: "postgres", Group: health.GroupInformational, Status: health.StatusFail,
						Since: rfc3339(t, "2026-08-08T07:07:52Z")},
				},
			},
			want: `{"status":"degraded","checks":[{"name":"grpc","group":"critical","status":"ok","since":"2026-08-08T07:04:11Z"},{"name":"postgres","group":"informational","status":"fail","since":"2026-08-08T07:07:52Z"}]}`,
		},
		{
			name: "cold start, never run",
			snap: health.Snapshot{
				State:  health.StateUnknown,
				Checks: []health.CheckResult{{Name: "grpc", Group: health.GroupCritical, Status: health.StatusUnknown}},
			},
			want: `{"status":"unknown","checks":[{"name":"grpc","group":"critical","status":"unknown"}]}`,
		},
		{
			name: "empty registry",
			snap: health.Snapshot{State: health.StateUnknown},
			want: `{"status":"unknown","checks":[]}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := json.Marshal(detail(tt.snap))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if got := string(buf); got != tt.want {
				t.Fatalf("body = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestHealthzEmptyChecks(t *testing.T) {
	// An empty registry is a real production state: engine's first moments.
	body := get(Handler(newTestRegistry(t, fastConfig())), "/healthz").Body.String()

	if !strings.Contains(body, `"checks":[]`) {
		t.Fatalf("body = %s, want an empty checks array", body)
	}
	if strings.Contains(body, `"checks":null`) {
		t.Fatalf("body = %s, want [] rather than null", body)
	}
}

func TestSinceOmitted(t *testing.T) {
	zero := detail(health.Snapshot{Checks: []health.CheckResult{{Name: "grpc"}}})
	if zero.Checks[0].Since != "" {
		t.Fatalf("since = %q, want it omitted for a zero timestamp", zero.Checks[0].Since)
	}

	buf, err := json.Marshal(zero)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(buf), "since") {
		t.Fatalf("body = %s, want no since key at all", buf)
	}

	// Nanoseconds and a non-UTC zone both have to be normalised away.
	local := time.Date(2026, 8, 8, 10, 4, 11, 123456789, time.FixedZone("EEST", 3*60*60))
	set := detail(health.Snapshot{Checks: []health.CheckResult{{Name: "grpc", Since: local}}})
	if got := set.Checks[0].Since; got != "2026-08-08T07:04:11Z" {
		t.Fatalf("since = %q, want 2026-08-08T07:04:11Z", got)
	}
}

func TestHealthzSorted(t *testing.T) {
	// D1 locks a byte-stable, name-sorted array. detail() preserves input
	// order, so this drives a real registry: it exists so a future core change
	// cannot silently break the contract at the transport boundary.
	reg := newTestRegistry(t, fastConfig())
	for _, name := range []string{"zulu", "alpha", "mike"} {
		reg.Critical(name, func(context.Context) error { return nil })
	}
	waitState(t, reg, health.StateReady)

	var body bodyJSON
	raw := get(Handler(reg), "/healthz").Body.Bytes()
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("body %s does not parse: %v", raw, err)
	}

	for i, want := range []string{"alpha", "mike", "zulu"} {
		if body.Checks[i].Name != want {
			t.Fatalf("checks[%d] = %q, want %q", i, body.Checks[i].Name, want)
		}
	}
}

// failingRegistry has one check per group, all failing with the sentinel.
func failingRegistry(t *testing.T) *health.Registry {
	t.Helper()

	fail := func(context.Context) error { return errors.New(sentinel) }

	reg := newTestRegistry(t, fastConfig())
	reg.Liveness("live", fail)
	reg.Critical("db", fail)
	reg.Informational("s3", fail)
	waitState(t, reg, health.StateNotReady)

	return reg
}

func singleEndpointHandler(t *testing.T, r *health.Registry, ep string) http.Handler {
	t.Helper()

	if ep == epLivez {
		return LivenessHandler(r)
	}
	if ep == epReadyz {
		return ReadinessHandler(r)
	}

	return HealthHandler(r)
}

func assertNoSentinel(t *testing.T, where, body string, header http.Header) {
	t.Helper()

	if strings.Contains(body, sentinel) {
		t.Fatalf("%s: the body leaks a check error: %q", where, body)
	}
	for key, values := range header {
		for _, value := range values {
			if strings.Contains(value, sentinel) {
				t.Fatalf("%s: header %s leaks a check error: %q", where, key, value)
			}
		}
	}
}

func TestNoErrorTextLeaks(t *testing.T) {
	// The property this whole phase exists for, over the cross-product of
	// endpoint x mode x constructor.
	reg := failingRegistry(t)

	// Without this the whole test could pass vacuously on a healthy registry.
	for _, c := range reg.Snapshot().Checks {
		if c.Status != health.StatusFail {
			t.Fatalf("check %q is %s, want %s", c.Name, c.Status, health.NameFail)
		}
	}

	srv := startServer(t, reg)

	for _, ep := range []string{epLivez, epReadyz, epHealthz} {
		for _, query := range []string{"", "?verbose"} {
			target := "/" + ep + query

			rec := get(Handler(reg), target)
			assertNoSentinel(t, "Handler "+target, rec.Body.String(), rec.Header())

			single := get(singleEndpointHandler(t, reg, ep), target)
			assertNoSentinel(t, "single-endpoint "+target, single.Body.String(), single.Header())

			_, body, header := fetch(t, "http://"+srv.Addr()+target)
			assertNoSentinel(t, "Server "+target, body, header)
		}
	}
}
