package fts_client

import "testing"

type pubsub struct {
}

func (p *pubsub) Send(exchange string, rk string, body []byte) error {
	return nil
}

func TestClient(t *testing.T) {
	c := New(&pubsub{})
	c.Create(1, "aaa", 1, map[string]any{
		"a": "a",
	})
}
