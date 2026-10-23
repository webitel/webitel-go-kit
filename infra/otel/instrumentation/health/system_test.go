package health

import (
	"testing"

	"github.com/webitel/webitel-go-kit/infra/health"
	"github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.4.0/healthconv"
)

// health cannot import semconv, so it keeps its own copy of the enum. This
// catches a value drifting apart, not a member missing from health: generated
// enums cannot be listed.
func TestSystemValuesMatchSemconv(t *testing.T) {
	for _, tc := range []struct {
		health  health.System
		semconv healthconv.CheckSystemAttr
	}{
		{health.SystemPostgreSQL, healthconv.CheckSystemPostgreSQL},
		{health.SystemRedis, healthconv.CheckSystemRedis},
		{health.SystemRabbitMQ, healthconv.CheckSystemRabbitMQ},
		{health.SystemGRPC, healthconv.CheckSystemGRPC},
		{health.SystemHTTP, healthconv.CheckSystemHTTP},
	} {
		if string(tc.health) != string(tc.semconv) {
			t.Errorf("health %q != semconv %q", tc.health, tc.semconv)
		}
	}
}
