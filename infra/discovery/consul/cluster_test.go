package consul

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const healthJSON = `[{"Node":{"Node":"n1","Address":"10.0.0.1","Meta":{"dc":"dc_a"}},` +
	`"Service":{"ID":"web-1","Service":"web","Address":"","Port":8080},` +
	`"Checks":[{"Status":"passing"}]}]`

type fakeServer struct {
	*httptest.Server
	hits atomic.Int32
}

func newFakeServer(t *testing.T, h http.HandlerFunc) *fakeServer {
	t.Helper()

	f := &fakeServer{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		h(w, r)
	}))
	t.Cleanup(f.Close)

	return f
}

func (f *fakeServer) addr() string { return f.Listener.Addr().String() }

func status(code int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}
}

func newTestCluster(t *testing.T, addrs ...string) *Cluster {
	t.Helper()

	c, err := NewCluster(addrs, WithRequestTimeout(500*time.Millisecond))
	require.NoError(t, err)

	return c
}

func TestNewClusterNoAddrs(t *testing.T) {
	_, err := NewCluster(nil)
	require.ErrorIs(t, err, ErrNoAddrs)
}

func TestClusterFailover(t *testing.T) {
	ctx := t.Context()

	t.Run("5xx moves on and the answering server stays current", func(t *testing.T) {
		broken := newFakeServer(t, status(http.StatusInternalServerError, "No cluster leader"))
		ok := newFakeServer(t, status(http.StatusOK, healthJSON))
		c := newTestCluster(t, broken.addr(), ok.addr())

		for range 2 {
			_, err := c.ServiceNodes(ctx, "web", ServiceQuery{})
			require.NoError(t, err)
		}

		assert.Equal(t, int32(1), broken.hits.Load())
		assert.Equal(t, int32(2), ok.hits.Load())
	})

	t.Run("4xx is the answer and is not retried", func(t *testing.T) {
		denied := newFakeServer(t, status(http.StatusForbidden, "ACL not found"))
		ok := newFakeServer(t, status(http.StatusOK, healthJSON))
		c := newTestCluster(t, denied.addr(), ok.addr())

		_, err := c.ServiceNodes(ctx, "web", ServiceQuery{})
		require.Error(t, err)

		var ee *EndpointError

		require.ErrorAs(t, err, &ee)
		assert.Equal(t, denied.addr(), ee.Addr)

		var se api.StatusError

		require.ErrorAs(t, err, &se)
		assert.Equal(t, http.StatusForbidden, se.Code)
		assert.Zero(t, ok.hits.Load())
	})

	t.Run("every attempt is reported", func(t *testing.T) {
		a := newFakeServer(t, status(http.StatusBadGateway, ""))
		b := newFakeServer(t, status(http.StatusServiceUnavailable, ""))
		c := newTestCluster(t, a.addr(), b.addr())

		_, err := c.ServiceNodes(ctx, "web", ServiceQuery{})
		require.Error(t, err)

		joined, ok := err.(interface{ Unwrap() []error })
		require.True(t, ok)
		assert.Len(t, joined.Unwrap(), 2)
	})

	t.Run("hung server is cut by the request timeout", func(t *testing.T) {
		hung := newFakeServer(t, func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
		ok := newFakeServer(t, status(http.StatusOK, healthJSON))
		c := newTestCluster(t, hung.addr(), ok.addr())

		_, err := c.ServiceNodes(ctx, "web", ServiceQuery{})
		require.NoError(t, err)
	})

	t.Run("caller context ends the failover", func(t *testing.T) {
		hung := newFakeServer(t, func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
		ok := newFakeServer(t, status(http.StatusOK, healthJSON))
		c := newTestCluster(t, hung.addr(), ok.addr())

		cctx, cancel := context.WithCancel(ctx)
		time.AfterFunc(50*time.Millisecond, cancel)

		_, err := c.ServiceNodes(cctx, "web", ServiceQuery{})
		require.ErrorIs(t, err, context.Canceled)
		assert.Zero(t, ok.hits.Load())
	})
}

func TestClusterKV(t *testing.T) {
	ctx := t.Context()

	t.Run("get returns ModifyIndex", func(t *testing.T) {
		srv := newFakeServer(t, status(http.StatusOK,
			`[{"Key":"k","Value":"dg==","ModifyIndex":42,"Session":"s1"}]`))
		c := newTestCluster(t, srv.addr())

		got, err := c.KVGet(ctx, "k")
		require.NoError(t, err)
		assert.Equal(t, &KVEntry{Key: "k", Value: []byte("v"), ModifyIndex: 42, Session: "s1"}, got)
	})

	t.Run("missing key", func(t *testing.T) {
		srv := newFakeServer(t, status(http.StatusNotFound, ""))
		c := newTestCluster(t, srv.addr())

		_, err := c.KVGet(ctx, "k")
		require.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("cas sends cas index and value", func(t *testing.T) {
		var (
			method, query string
			body          []byte
		)

		srv := newFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			method, query = r.Method, r.URL.Query().Get("cas")
			body, _ = io.ReadAll(r.Body)
			_, _ = w.Write([]byte("false"))
		})
		c := newTestCluster(t, srv.addr())

		ok, err := c.KVCAS(ctx, "k", []byte("v2"), 42)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, http.MethodPut, method)
		assert.Equal(t, "42", query)
		assert.Equal(t, "v2", string(body))
	})
}

func TestClusterSessions(t *testing.T) {
	ctx := t.Context()

	t.Run("create has no checks and explicit behavior", func(t *testing.T) {
		var got map[string]any

		srv := newFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/session/create", r.URL.Path)
			_ = json.NewDecoder(r.Body).Decode(&got)
			_, _ = w.Write([]byte(`{"ID":"sid"}`))
		})
		c := newTestCluster(t, srv.addr())

		id, err := c.CreateSession(ctx, SessionConfig{Name: "n", TTL: 15 * time.Second, LockDelay: 20 * time.Second})
		require.NoError(t, err)
		assert.Equal(t, "sid", id)
		assert.Equal(t, []any{}, got["NodeChecks"])
		assert.Equal(t, "release", got["Behavior"])
		assert.Equal(t, "15s", got["TTL"])
		assert.Equal(t, "20000ms", got["LockDelay"])
	})

	t.Run("renew of a gone session", func(t *testing.T) {
		srv := newFakeServer(t, status(http.StatusNotFound, ""))
		c := newTestCluster(t, srv.addr())

		require.ErrorIs(t, c.RenewSession(ctx, "sid"), ErrSessionNotFound)
	})

	t.Run("acquire and release carry the session", func(t *testing.T) {
		var queries []string

		srv := newFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			queries = append(queries, r.URL.RawQuery)
			_, _ = w.Write([]byte("true"))
		})
		c := newTestCluster(t, srv.addr())

		ok, err := c.Acquire(ctx, "lock", []byte("me"), "sid")
		require.NoError(t, err)
		assert.True(t, ok)

		ok, err = c.Release(ctx, "lock", []byte("me"), "sid")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []string{"acquire=sid", "release=sid"}, queries)
	})
}

func TestClusterServices(t *testing.T) {
	ctx := t.Context()

	t.Run("filters by passing and node-meta", func(t *testing.T) {
		var query map[string][]string

		srv := newFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/health/service/web", r.URL.Path)
			query = r.URL.Query()
			_, _ = w.Write([]byte(healthJSON))
		})
		c := newTestCluster(t, srv.addr())

		got, err := c.ServiceNodes(ctx, "web", ServiceQuery{PassingOnly: true, NodeMeta: map[string]string{"dc": "dc_a"}})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "10.0.0.1:8080", got[0].HostPort(), "empty service address falls back to the node's")
		assert.Equal(t, "passing", got[0].Status)
		assert.Equal(t, map[string]string{"dc": "dc_a"}, got[0].NodeMeta)
		assert.Equal(t, []string{"dc:dc_a"}, query["node-meta"])
		assert.NotEmpty(t, query["passing"])
	})

	t.Run("prepared query returns node meta", func(t *testing.T) {
		srv := newFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/query/webitel-api/execute", r.URL.Path)
			_, _ = w.Write([]byte(`{"Service":"webitel-api","Nodes":` + healthJSON + `}`))
		})
		c := newTestCluster(t, srv.addr())

		got, err := c.ExecuteQuery(ctx, "webitel-api")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "dc_a", got[0].NodeMeta["dc"])
	})
}
