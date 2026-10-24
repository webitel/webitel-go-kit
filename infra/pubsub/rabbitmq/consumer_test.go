package rabbitmq

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type countingLogger struct {
	NoopLogger
	errors atomic.Int64
}

func (l *countingLogger) Error(string, error, ...any) { l.errors.Add(1) }

func TestConsumerFailureKeepsSharedChannel(t *testing.T) {
	defer goleak.VerifyNone(t)

	logger := &countingLogger{}
	conn := newTestConnection(t, logger)

	shared, err := conn.Channel(context.Background())
	require.NoError(t, err)

	queueCfg, err := NewQueueConfig("rabbitmq-test-missing-queue")
	require.NoError(t, err)

	consumerCfg, err := NewConsumerConfig("rabbitmq-test", WithConsumerReconnectDelay(100*time.Millisecond))
	require.NoError(t, err)

	consumer := NewConsumer(conn, queueCfg, consumerCfg, func(context.Context, amqp.Delivery) error { return nil }, logger)
	require.NoError(t, consumer.Start(context.Background()))

	require.Eventually(t, func() bool {
		return logger.errors.Load() >= 2
	}, 5*time.Second, 50*time.Millisecond)

	ch, err := conn.Channel(context.Background())
	require.NoError(t, err)
	require.Same(t, shared, ch)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, consumer.Shutdown(ctx))
	require.NoError(t, conn.Close())
}
