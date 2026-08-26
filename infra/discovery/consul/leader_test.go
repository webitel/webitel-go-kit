package consul

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/webitel/webitel-go-kit/infra/discovery"
)

func testElector(t *testing.T, opts ...discovery.Option[*LeaderElector]) *LeaderElector {
	t.Helper()

	le, err := NewLeaderElector(
		"127.0.0.1:8500",
		"webitel-kb",
		"webitel-kb@host-1",
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		opts...,
	)
	require.NoError(t, err)

	return le
}

func TestLeaderKey(t *testing.T) {
	assert.Equal(t, "service/webitel-kb/leader", LeaderKey("webitel-kb"))
}

func TestNewLeaderElectorDefaults(t *testing.T) {
	le := testElector(t)

	assert.Equal(t, "service/webitel-kb/leader", le.key)
	assert.Equal(t, "webitel-kb@host-1", le.nodeID)
	assert.Equal(t, "webitel-kb-leader-lock", le.sessionName)
	assert.Equal(t, DefaultSessionTTL, le.sessionTTL)
	assert.Equal(t, DefaultRetryInterval, le.retryInterval)
	assert.Equal(t, DefaultErrorCooldown, le.errCooldown)
	assert.Equal(t, DefaultMonitorInterval, le.monitorInterval)
	assert.Zero(t, le.lockDelay, "zero leaves the consul default")
}

func TestLeaderElectorOptions(t *testing.T) {
	le := testElector(t,
		WithSessionTTL(10*time.Second),
		WithLockDelay(time.Millisecond),
		WithRetryInterval(3*time.Second),
		WithErrorCooldown(2*time.Second),
		WithMonitorInterval(time.Second),
	)

	assert.Equal(t, 10*time.Second, le.sessionTTL)
	assert.Equal(t, time.Millisecond, le.lockDelay)
	assert.Equal(t, 3*time.Second, le.retryInterval)
	assert.Equal(t, 2*time.Second, le.errCooldown)
	assert.Equal(t, time.Second, le.monitorInterval)
}

// A zero interval would panic the monitor ticker.
func TestLeaderElectorOptionsRejectNonPositive(t *testing.T) {
	le := testElector(t,
		WithSessionTTL(0),
		WithRetryInterval(-time.Second),
		WithErrorCooldown(0),
		WithMonitorInterval(0),
	)

	assert.Equal(t, "service/webitel-kb/leader", le.key)
	assert.Equal(t, "webitel-kb-leader-lock", le.sessionName)
	assert.Equal(t, DefaultSessionTTL, le.sessionTTL)
	assert.Equal(t, DefaultRetryInterval, le.retryInterval)
	assert.Equal(t, DefaultErrorCooldown, le.errCooldown)
	assert.Equal(t, DefaultMonitorInterval, le.monitorInterval)
}

// The session shape decides how long a dead instance keeps leadership.
func TestSessionEntry(t *testing.T) {
	entry := testElector(t,
		WithSessionTTL(10*time.Second),
		WithLockDelay(time.Millisecond),
	).sessionEntry()

	assert.Equal(t, "webitel-kb-leader-lock", entry.Name)
	assert.Equal(t, "10s", entry.TTL)
	assert.Equal(t, time.Millisecond, entry.LockDelay)
	assert.Equal(t, api.SessionBehaviorRelease, entry.Behavior)
}

func TestSessionEntryDefaultTTL(t *testing.T) {
	entry := testElector(t).sessionEntry()

	assert.Equal(t, "15s", entry.TTL)
	assert.Zero(t, entry.LockDelay)
}

func TestRunReturnsOnCanceledContext(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	started := false
	testElector(t).Run(ctx, func(context.Context) error { started = true; return nil }, func() {})

	assert.False(t, started, "no leader work without leadership")
}
