//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/webitel/webitel-go-kit/infra/discovery/consul"
)

func startConsul(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "hashicorp/consul:1.15",
			ExposedPorts: []string{"8500/tcp"},
			WaitingFor:   wait.ForHTTP("/v1/status/leader").WithPort("8500/tcp"),
			Cmd:          []string{"agent", "-dev", "-client", "0.0.0.0"},
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Terminate(ctx) })

	host, err := c.Host(ctx)
	require.NoError(t, err)

	port, err := c.MappedPort(ctx, "8500")
	require.NoError(t, err)

	return host + ":" + port.Port()
}

func TestClusterE2E(t *testing.T) {
	ctx := t.Context()
	addr := startConsul(t)

	// The first address refuses connections, so every call also proves the failover.
	cluster, err := consul.NewCluster([]string{"127.0.0.1:1", addr}, consul.WithRequestTimeout(5*time.Second))
	require.NoError(t, err)

	t.Run("kv cas by ModifyIndex", func(t *testing.T) {
		_, err := cluster.KVGet(ctx, "cas/key")
		require.ErrorIs(t, err, consul.ErrKeyNotFound)

		ok, err := cluster.KVCAS(ctx, "cas/key", []byte("v1"), 0)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = cluster.KVCAS(ctx, "cas/key", []byte("other"), 0)
		require.NoError(t, err)
		assert.False(t, ok, "index 0 must not overwrite an existing key")

		got, err := cluster.KVGet(ctx, "cas/key")
		require.NoError(t, err)
		assert.Equal(t, []byte("v1"), got.Value)

		ok, err = cluster.KVCAS(ctx, "cas/key", []byte("v2"), got.ModifyIndex)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = cluster.KVCAS(ctx, "cas/key", []byte("v3"), got.ModifyIndex)
		require.NoError(t, err)
		assert.False(t, ok, "stale index must lose")

		_, err = cluster.KVCAS(ctx, "cas/other", []byte("x"), 0)
		require.NoError(t, err)

		list, err := cluster.KVList(ctx, "cas/")
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, "cas/key", list[0].Key)

		none, err := cluster.KVList(ctx, "nothing/")
		require.NoError(t, err)
		assert.Empty(t, none)

		ok, err = cluster.KVDeleteCAS(ctx, "cas/key", got.ModifyIndex)
		require.NoError(t, err)
		assert.False(t, ok, "stale index must not delete")

		cur, err := cluster.KVGet(ctx, "cas/key")
		require.NoError(t, err)

		ok, err = cluster.KVDeleteCAS(ctx, "cas/key", cur.ModifyIndex)
		require.NoError(t, err)
		assert.True(t, ok)

		_, err = cluster.KVGet(ctx, "cas/key")
		require.ErrorIs(t, err, consul.ErrKeyNotFound)
	})

	t.Run("session lock", func(t *testing.T) {
		cfg := consul.SessionConfig{Name: "e2e", TTL: 10 * time.Second, LockDelay: time.Second}

		s1, err := cluster.CreateSession(ctx, cfg)
		require.NoError(t, err)

		s2, err := cluster.CreateSession(ctx, cfg)
		require.NoError(t, err)

		require.NoError(t, cluster.RenewSession(ctx, s1))

		ok, err := cluster.Acquire(ctx, "lock/key", []byte("one"), s1)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = cluster.Acquire(ctx, "lock/key", []byte("two"), s2)
		require.NoError(t, err)
		assert.False(t, ok, "a held lock must not be taken")

		held, err := cluster.KVGet(ctx, "lock/key")
		require.NoError(t, err)
		assert.Equal(t, s1, held.Session)

		ok, err = cluster.Release(ctx, "lock/key", []byte("one"), s1)
		require.NoError(t, err)
		assert.True(t, ok)

		require.NoError(t, cluster.DestroySession(ctx, s1))
		require.ErrorIs(t, cluster.RenewSession(ctx, s1), consul.ErrSessionNotFound)

		_, err = cluster.Acquire(ctx, "lock/key", []byte("x"), s1)
		require.ErrorIs(t, err, consul.ErrSessionNotFound, "a gone session on acquire")

		// Release of a key the gone session does not hold is answered false by Consul 1.15.
		ok, err = cluster.Release(ctx, "lock/key", []byte("x"), s1)
		if err != nil {
			require.ErrorIs(t, err, consul.ErrSessionNotFound)
		} else {
			assert.False(t, ok)
		}

		sess, _, err := mustAPI(t, addr).Session().Info(s2, nil)
		require.NoError(t, err)
		require.NotNil(t, sess)
		assert.Empty(t, sess.NodeChecks, "session must not carry serfHealth")
		assert.Equal(t, api.SessionBehaviorRelease, sess.Behavior)
	})

	t.Run("services filtered by tag", func(t *testing.T) {
		_, err := mustAPI(t, addr).Catalog().Register(&api.CatalogRegistration{
			Node:    "node-tags",
			Address: "10.3.0.1",
			Service: &api.AgentService{ID: "svc-http", Service: "tagged", Port: 4646, Tags: []string{"http"}},
		}, nil)
		require.NoError(t, err)

		_, err = mustAPI(t, addr).Catalog().Register(&api.CatalogRegistration{
			Node:    "node-tags",
			Address: "10.3.0.1",
			Service: &api.AgentService{ID: "svc-rpc", Service: "tagged", Port: 4647, Tags: []string{"rpc"}},
		}, nil)
		require.NoError(t, err)

		got, err := cluster.ServiceNodes(ctx, "tagged", consul.ServiceQuery{Tag: "http"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, 4646, got[0].Port)

		all, err := cluster.ServiceNodes(ctx, "tagged", consul.ServiceQuery{})
		require.NoError(t, err)
		assert.Len(t, all, 2)
	})

	t.Run("services filtered by node meta and prepared query", func(t *testing.T) {
		a := mustAPI(t, addr)

		for _, dc := range []string{"dc_a", "dc_b"} {
			_, err := a.Catalog().Register(&api.CatalogRegistration{
				Node:     "node-" + dc,
				Address:  map[string]string{"dc_a": "10.1.0.1", "dc_b": "10.2.0.1"}[dc],
				NodeMeta: map[string]string{"webitel_datacenter": dc},
				Service:  &api.AgentService{ID: "api-" + dc, Service: "api", Port: 8080},
			}, nil)
			require.NoError(t, err)
		}

		got, err := cluster.ServiceNodes(ctx, "api", consul.ServiceQuery{
			PassingOnly: true,
			NodeMeta:    map[string]string{"webitel_datacenter": "dc_b"},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "10.2.0.1:8080", got[0].HostPort())

		all, err := cluster.ServiceNodes(ctx, "api", consul.ServiceQuery{PassingOnly: true})
		require.NoError(t, err)
		assert.Len(t, all, 2)

		_, _, err = a.PreparedQuery().Create(&api.PreparedQueryDefinition{
			Name:    "webitel-api",
			Service: api.ServiceQuery{Service: "api", OnlyPassing: true},
		}, nil)
		require.NoError(t, err)

		nodes, err := cluster.ExecuteQuery(ctx, "webitel-api")
		require.NoError(t, err)
		require.Len(t, nodes, 2)

		dcs := map[string]bool{}
		for _, n := range nodes {
			dcs[n.NodeMeta["webitel_datacenter"]] = true
		}

		assert.Equal(t, map[string]bool{"dc_a": true, "dc_b": true}, dcs)
	})
}

func mustAPI(t *testing.T, addr string) *api.Client {
	t.Helper()

	c, err := api.NewClient(&api.Config{Address: addr, Scheme: "http"})
	require.NoError(t, err)

	return c
}
