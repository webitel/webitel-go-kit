package limitmux

import (
	"encoding/json"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

func Test_statusTopWorst_Less(t *testing.T) {

	var (
		statusForbidden    = ratelimit.Status{}
		statusDenied       = ratelimit.Status{Limit: 3, Allowed: 0}
		statusBypass       = ratelimit.Status{Limit: 0, Allowed: 1}
		statusLimit3Allow  = ratelimit.Status{Limit: 3, Allowed: 1, Remaining: 0}
		statusLimit7Remain = ratelimit.Status{Limit: 7, Allowed: 1, Remaining: 5}
		statusWaitLess     = ratelimit.Status{Limit: 7, RetryAfter: (40 * time.Millisecond)}
		statusWaitMore     = ratelimit.Status{Limit: 3, RetryAfter: (60 * time.Millisecond)}
	)

	tests := []struct {
		name  string
		group []*ratelimit.Status
		want  []*ratelimit.Status
	}{
		// TODO: Add test cases.
		{
			group: []*ratelimit.Status{
				&statusLimit3Allow,
				&statusBypass,
				&statusForbidden,
				&statusWaitLess,
				&statusDenied,
				&statusLimit7Remain,
				&statusWaitMore,
			},
			want: []*ratelimit.Status{
				&statusWaitMore,
				&statusWaitLess,
				&statusDenied,
				&statusForbidden,
				&statusLimit3Allow,
				&statusLimit7Remain,
				&statusBypass,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			sort.Sort(statusTopWorst(tt.group))
			// TODO: update the condition below to compare got with tt.want.
			if !slices.Equal(tt.group, tt.want) {
				got, _ := json.MarshalIndent(tt.group, "", "  ")
				want, _ := json.MarshalIndent(tt.want, "", "  ")
				t.Errorf("Less() = %s, want %s", got, want)
			}
		})
	}
}
