package rabbitmq

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
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

	conn, err := NewConnection(cfg, nil)
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

func newTestConnection(t *testing.T) *Connection {
	t.Helper()

	if testing.Short() {
		t.Skip("needs RabbitMQ on localhost:5672")
	}

	conn, err := NewConnection(&Config{
		URL:            "amqp://guest:guest@localhost:5672/",
		ConnectTimeout: time.Second,
	}, nil)
	require.NoError(t, err)

	return conn
}

func TestSharedChannelReopens(t *testing.T) {
	tests := []struct {
		name  string
		close func(ch *amqp.Channel)
	}{
		{
			name:  "closed by caller",
			close: func(ch *amqp.Channel) { _ = ch.Close() },
		},
		{
			name: "closed by channel error",
			close: func(ch *amqp.Channel) {
				_, _ = ch.QueueDeclarePassive("rabbitmq-test-missing-queue", false, false, false, false, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			conn := newTestConnection(t)

			old, err := conn.Channel(context.Background())
			require.NoError(t, err)

			tt.close(old)

			require.Eventually(t, func() bool {
				ch, err := conn.Channel(context.Background())

				return err == nil && ch != old
			}, 5*time.Second, 50*time.Millisecond)

			require.NoError(t, conn.Close())
		})
	}
}
