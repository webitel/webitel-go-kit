package consul

import (
	"time"

	"github.com/webitel/webitel-go-kit/infra/discovery"
)

// WithSessionTTL sets how long leadership survives a dead instance; Consul
// invalidates the session at up to twice this value and accepts 10s..24h.
func WithSessionTTL(ttl time.Duration) discovery.Option[*LeaderElector] {
	return func(le *LeaderElector) {
		if ttl > 0 {
			le.sessionTTL = ttl
		}
	}
}

// WithLockDelay sets how long the key stays unacquirable after the session dies;
// zero leaves the Consul default of fifteen seconds.
func WithLockDelay(delay time.Duration) discovery.Option[*LeaderElector] {
	return func(le *LeaderElector) {
		if delay >= 0 {
			le.lockDelay = delay
		}
	}
}

// WithRetryInterval sets how long a standby waits before trying the key again.
func WithRetryInterval(interval time.Duration) discovery.Option[*LeaderElector] {
	return func(le *LeaderElector) {
		if interval > 0 {
			le.retryInterval = interval
		}
	}
}

// WithErrorCooldown sets how long the elector waits after a failed call or term.
func WithErrorCooldown(cooldown time.Duration) discovery.Option[*LeaderElector] {
	return func(le *LeaderElector) {
		if cooldown > 0 {
			le.errCooldown = cooldown
		}
	}
}

// WithMonitorInterval sets how often the leader verifies it still holds the key.
func WithMonitorInterval(interval time.Duration) discovery.Option[*LeaderElector] {
	return func(le *LeaderElector) {
		if interval > 0 {
			le.monitorInterval = interval
		}
	}
}
