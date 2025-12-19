package consul

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func TestConsulKVWatcherNextFakeServer(t *testing.T) {
	t.Run("Successfully returns new value when index increases", func(t *testing.T) {
		expectedValue := []byte("updated-config")

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Equal(t, "10", query.Get("index"))

			w.Header().Set("X-Consul-Index", "11")
			w.WriteHeader(http.StatusOK)

			resp := []*api.KVPair{
				{
					Key:   "test/key",
					Value: expectedValue,
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer ts.Close()

		client, _ := api.NewClient(&api.Config{Address: ts.URL})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		w := &consulKVWatcher{
			ctx:       ctx,
			cancel:    cancel,
			key:       "test/key",
			client:    client,
			lastIndex: 10,
		}

		val, err := w.Next()

		assert.NoError(t, err)
		assert.Equal(t, expectedValue, val)
		assert.Equal(t, uint64(11), w.lastIndex)
	})

	t.Run("Returns nil when key is deleted", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Consul-Index", "20")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[]"))
		}))
		defer ts.Close()

		client, _ := api.NewClient(&api.Config{Address: ts.URL})
		w := &consulKVWatcher{
			ctx:       context.Background(),
			key:       "test/key",
			client:    client,
			lastIndex: 15,
		}

		val, err := w.Next()

		assert.NoError(t, err)
		assert.Nil(t, val)
		assert.Equal(t, uint64(20), w.lastIndex)
	})

	t.Run("Handles context cancellation", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(2 * time.Second):
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer ts.Close()

		client, _ := api.NewClient(&api.Config{Address: ts.URL})
		ctx, cancel := context.WithCancel(context.Background())

		w := &consulKVWatcher{
			ctx:       ctx,
			cancel:    cancel,
			key:       "test/key",
			client:    client,
			lastIndex: 100,
		}

		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		val, err := w.Next()

		assert.Error(t, err)
		assert.Nil(t, val)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
