package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestPingNotConnected(t *testing.T) {
	conn := &Connection{done: make(chan struct{})}

	require.ErrorIs(t, conn.Ping(context.Background()), ErrConnectionNotAvailable)
}

func TestPingCanceledContext(t *testing.T) {
	conn := &Connection{done: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.ErrorIs(t, conn.Ping(ctx), context.Canceled)
}

func TestPingFollowsConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("needs RabbitMQ on localhost:5672")
	}

	defer goleak.VerifyNone(t)

	cfg := &Config{
		URL:            "amqp://guest:guest@localhost:5672/",
		ConnectTimeout: time.Second,
	}

	conn, err := NewConnection(cfg, &NoopLogger{})
	require.NoError(t, err)

	require.NoError(t, conn.Ping(context.Background()))

	conn.mu.RLock()
	amqpConn := conn.conn
	conn.mu.RUnlock()
	_ = amqpConn.Close()

	require.Eventually(t, func() bool {
		return conn.Ping(context.Background()) == nil
	}, 5*time.Second, 50*time.Millisecond)

	require.NoError(t, conn.Close())
	require.ErrorIs(t, conn.Ping(context.Background()), ErrConnectionNotAvailable)
}
