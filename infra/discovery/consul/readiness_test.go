package consul

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientTTLStatus(t *testing.T) {
	tests := []struct {
		name       string
		readiness  func() (bool, error)
		wantStatus string
		wantOutput string
	}{
		{
			name:       "no readiness func",
			wantStatus: api.HealthPassing,
			wantOutput: "pass",
		},
		{
			name:       "ready",
			readiness:  func() (bool, error) { return true, nil },
			wantStatus: api.HealthPassing,
			wantOutput: "pass",
		},
		{
			name:       "ready wins over an error",
			readiness:  func() (bool, error) { return true, errors.New("ignored") },
			wantStatus: api.HealthPassing,
			wantOutput: "pass",
		},
		{
			name:       "not ready with reason",
			readiness:  func() (bool, error) { return false, errors.New("health: not_ready: failing grpc") },
			wantStatus: api.HealthCritical,
			wantOutput: "health: not_ready: failing grpc",
		},
		{
			name:       "not ready without error",
			readiness:  func() (bool, error) { return false, nil },
			wantStatus: api.HealthCritical,
			wantOutput: "not ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient()
			c.readiness = tt.readiness

			status, output := c.ttlStatus()
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantOutput, output)
		})
	}
}

func TestWithReadiness(t *testing.T) {
	r := &Registry{client: newTestClient()}

	WithReadiness(func() (bool, error) { return false, errors.New("down") })(r)

	require.NotNil(t, r.client.readiness)
	ok, err := r.client.readiness()
	assert.False(t, ok)
	assert.EqualError(t, err, "down")
}

type ttlUpdate struct {
	path   string
	Status string
	Output string
}

func newTTLAgent(t *testing.T) (*api.Client, func() []ttlUpdate) {
	t.Helper()

	var (
		mu      sync.Mutex
		updates []ttlUpdate
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := ttlUpdate{path: r.URL.Path}
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			t.Errorf("decode ttl update: %v", err)
		}

		mu.Lock()
		updates = append(updates, u)
		mu.Unlock()
	}))
	t.Cleanup(srv.Close)

	client, err := api.NewClient(&api.Config{Address: srv.URL})
	require.NoError(t, err)

	return client, func() []ttlUpdate {
		mu.Lock()
		defer mu.Unlock()

		return append([]ttlUpdate(nil), updates...)
	}
}

func TestClientSendTTLReportsReadiness(t *testing.T) {
	tests := []struct {
		name       string
		readiness  func() (bool, error)
		wantStatus string
		wantOutput string
	}{
		{
			name:       "ready",
			readiness:  func() (bool, error) { return true, nil },
			wantStatus: api.HealthPassing,
			wantOutput: "pass",
		},
		{
			name:       "not ready",
			readiness:  func() (bool, error) { return false, errors.New("health: stopping: shutting down") },
			wantStatus: api.HealthCritical,
			wantOutput: "health: stopping: shutting down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, updates := newTTLAgent(t)

			c := newTestClient()
			c.client = agent
			c.readiness = tt.readiness

			require.NoError(t, c.sendTTL(context.Background(), "svc-1"))

			got := updates()
			require.Len(t, got, 1)
			assert.Equal(t, "/v1/agent/check/update/"+ServiceStr+"svc-1:ttl:1", got[0].path)
			assert.Equal(t, tt.wantStatus, got[0].Status)
			assert.Equal(t, tt.wantOutput, got[0].Output)
		})
	}
}
