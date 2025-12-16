package consul

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func TestGetFromKV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/kv/test-key" {
			pair := api.KVPair{
				Key:   "test-key",
				Value: []byte("hello-world"),
			}
			data, _ := json.Marshal(api.KVPairs{&pair})
			w.Write(data)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.Listener.Addr().String()
	client, _ := api.NewClient(config)

	kv := &kVClient{client: client}

	t.Run("Success", func(t *testing.T) {
		val, err := kv.GetFromKV(context.Background(), "test-key")
		assert.NoError(t, err)
		assert.Equal(t, []byte("hello-world"), val)
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		_, err := kv.GetFromKV(context.Background(), "unknown-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestPutToKV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		w.Write([]byte("true"))
	}))
	defer server.Close()

	config := api.DefaultConfig()
	config.Address = server.Listener.Addr().String()
	client, _ := api.NewClient(config)

	kv := &kVClient{client: client}

	err := kv.PutToKV(context.Background(), "new-key", []byte("data"))
	assert.NoError(t, err)
}

func TestDeleteFromKV(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("true"))
		}))
		defer server.Close()

		config := api.DefaultConfig()
		config.Address = server.Listener.Addr().String()
		client, _ := api.NewClient(config)

		kv := &kVClient{client: client}

		err := kv.DeleteFromKV(context.Background(), "delete-me")
		assert.NoError(t, err)
	})

	t.Run("ServerError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := api.DefaultConfig()
		config.Address = server.Listener.Addr().String()
		client, _ := api.NewClient(config)

		kv := &kVClient{client: client}

		err := kv.DeleteFromKV(context.Background(), "error-key")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "consul kv delete error (key: error-key)")
	})
}
