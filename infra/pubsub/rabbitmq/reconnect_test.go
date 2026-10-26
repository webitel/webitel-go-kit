package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestRabbitMQReconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("needs RabbitMQ on localhost:5672")
	}

	defer goleak.VerifyNone(t)

	cfg := &Config{
		URL:            "amqp://guest:guest@localhost:5672/",
		ConnectTimeout: time.Second,
	}

	conn, err := NewConnection(cfg, nil)
	require.NoError(t, err)

	for range 20 {
		conn.mu.RLock()
		amqpConn := conn.conn
		conn.mu.RUnlock()

		require.NotNil(t, amqpConn)
		_ = amqpConn.Close()

		require.Eventually(t, func() bool {
			ch, err := conn.Channel(context.Background())
			if err != nil {
				return false
			}
			return ch != nil
		}, 5*time.Second, 50*time.Millisecond)
	}

	_ = conn.Close()
}
