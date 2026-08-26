package consul

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hashicorp/consul/api"

	"github.com/webitel/webitel-go-kit/infra/discovery"
)

// LeadershipElector runs leader-only work on exactly one instance of a service.
// onStart must return once its context is canceled: the term is released only
// after it does, and onStop then reports the release.
type LeadershipElector interface {
	Run(ctx context.Context, onStart func(ctx context.Context) error, onStop func())
}

var _ LeadershipElector = (*LeaderElector)(nil)

// Defaults of the leader session. Consul invalidates a TTL session at up to
// twice its TTL and then holds the key for the lock delay, so those two bound
// how long a dead instance keeps leadership.
const (
	DefaultSessionTTL      = 15 * time.Second
	DefaultRetryInterval   = 10 * time.Second
	DefaultErrorCooldown   = 5 * time.Second
	DefaultMonitorInterval = 5 * time.Second
)

// LeaderElector elects one instance of a service through Consul.
type LeaderElector struct {
	client *api.Client
	log    *slog.Logger
	key    string
	nodeID string

	sessionName     string
	sessionTTL      time.Duration
	lockDelay       time.Duration
	retryInterval   time.Duration
	errCooldown     time.Duration
	monitorInterval time.Duration
}

// NewLeaderElector builds an elector that holds nodeID in the leader key of
// serviceName; discovery.GenerateInstanceID gives a suitable node id.
func NewLeaderElector(
	consulAddr, serviceName, nodeID string,
	log *slog.Logger,
	opts ...discovery.Option[*LeaderElector],
) (*LeaderElector, error) {
	cfg := api.DefaultConfig()
	cfg.Address = consulAddr

	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("consul client init failed: %w", err)
	}

	if log == nil {
		log = slog.Default()
	}

	le := &LeaderElector{
		client:          client,
		key:             LeaderKey(serviceName),
		nodeID:          nodeID,
		sessionName:     serviceName + "-leader-lock",
		sessionTTL:      DefaultSessionTTL,
		retryInterval:   DefaultRetryInterval,
		errCooldown:     DefaultErrorCooldown,
		monitorInterval: DefaultMonitorInterval,
	}

	for _, opt := range opts {
		opt(le)
	}

	le.log = log.With("component", "leader-elector", "key", le.key)

	return le, nil
}

// LeaderKey returns the KV key a service elects on.
func LeaderKey(serviceName string) string {
	return fmt.Sprintf("service/%s/leader", serviceName)
}

// Run blocks and keeps trying to acquire leadership until ctx is done.
func (le *LeaderElector) Run(ctx context.Context, onStart func(ctx context.Context) error, onStop func()) {
	for {
		select {
		case <-ctx.Done():
			le.log.Info("stopping leader election: context canceled")

			return
		default:
			le.attemptLeadership(ctx, onStart, onStop)
		}
	}
}

func (le *LeaderElector) attemptLeadership(
	ctx context.Context,
	onStart func(ctx context.Context) error,
	onStop func(),
) {
	sessionID, err := le.createSession()
	if err != nil {
		le.log.Error("failed to create session", "err", err)
		le.wait(ctx, le.errCooldown)

		return
	}

	defer le.destroySession(sessionID)

	acquired, err := le.acquireLock(sessionID)
	if err != nil {
		le.log.Error("error during lock acquisition", "err", err)
		le.wait(ctx, le.errCooldown)

		return
	}

	if !acquired {
		le.log.Debug("leader lock held by another instance")
		le.wait(ctx, le.retryInterval)

		return
	}

	le.log.Info("node promoted to leader", "node_id", le.nodeID, "session", sessionID)

	// leaderCtx bounds the leader-only work to this term.
	leaderCtx, cancelLeader := context.WithCancel(ctx)
	defer cancelLeader()

	renewDone, renewResult := le.renewSession(sessionID, cancelLeader)

	workResult := make(chan error, 1)

	go func() {
		err := onStart(leaderCtx)
		if err != nil {
			le.log.Error("leader task execution failed", "err", err)
			cancelLeader()
		}

		workResult <- err
	}()

	le.monitorLeadership(leaderCtx, sessionID)

	// Stop the work before the session goes away, otherwise a demoted instance
	// keeps working while the next one already leads.
	cancelLeader()
	workErr := <-workResult

	close(renewDone)
	renewErr := <-renewResult

	le.log.Warn("node demoted: releasing leadership")
	onStop()

	// Without this a failed term is retried as fast as Consul answers.
	if workErr != nil || renewErr != nil {
		le.wait(ctx, le.errCooldown)
	}
}

// renewSession keeps the session alive until done is closed; result then carries
// the outcome.
func (le *LeaderElector) renewSession(
	sessionID string,
	cancelLeader context.CancelFunc,
) (chan struct{}, <-chan error) {
	done := make(chan struct{})
	result := make(chan error, 1)

	go func() {
		err := le.client.Session().RenewPeriodic(le.sessionTTL.String(), sessionID, nil, done)
		if err != nil {
			le.log.Error("consul session renewal failed, stepping down", "err", err)
			cancelLeader()
		}

		result <- err
	}()

	return done, result
}

// sessionEntry describes the leader session; the lock delay is sent explicitly
// because Consul reads a missing one as its own default.
func (le *LeaderElector) sessionEntry() *api.SessionEntry {
	return &api.SessionEntry{
		Name:      le.sessionName,
		TTL:       le.sessionTTL.String(),
		LockDelay: le.lockDelay,
		Behavior:  api.SessionBehaviorRelease,
	}
}

func (le *LeaderElector) createSession() (string, error) {
	sessionID, _, err := le.client.Session().Create(le.sessionEntry(), nil)

	return sessionID, err
}

func (le *LeaderElector) acquireLock(sessionID string) (bool, error) {
	kv := &api.KVPair{
		Key:     le.key,
		Value:   []byte(le.nodeID),
		Session: sessionID,
	}

	acquired, _, err := le.client.KV().Acquire(kv, nil)

	return acquired, err
}

// monitorLeadership returns as soon as the key stops naming our session.
func (le *LeaderElector) monitorLeadership(ctx context.Context, sessionID string) {
	ticker := time.NewTicker(le.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pair, _, err := le.client.KV().Get(le.key, nil)
			if err != nil || pair == nil || pair.Session != sessionID {
				le.log.Debug("leadership check failed or session changed")

				return
			}
		}
	}
}

func (le *LeaderElector) destroySession(sessionID string) {
	if sessionID != "" {
		_, _ = le.client.Session().Destroy(sessionID, nil)
	}
}

func (le *LeaderElector) wait(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}
