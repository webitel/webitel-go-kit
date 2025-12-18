package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestRabbitMQReconnect(t *testing.T) {
	defer goleak.VerifyNone(t)

	cfg := &Config{
		URL:            "amqp://guest:guest@localhost:5672/",
		ConnectTimeout: time.Second,
	}

	logger := &NoopLogger{}

	conn, err := NewConnection(cfg, logger)
	require.NoError(t, err)

	for i := 0; i < 20; i++ {
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
