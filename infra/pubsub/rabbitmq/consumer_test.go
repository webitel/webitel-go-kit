package rabbitmq

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestConsumerFailureKeepsSharedChannel(t *testing.T) {
	defer goleak.VerifyNone(t)

	conn := newTestConnection(t)

	shared, err := conn.Channel(context.Background())
	require.NoError(t, err)

	queueCfg, err := NewQueueConfig("rabbitmq-test-missing-queue")
	require.NoError(t, err)

	consumerCfg, err := NewConsumerConfig("rabbitmq-test", WithConsumerReconnectDelay(100*time.Millisecond))
	require.NoError(t, err)

	consumer := NewConsumer(conn, queueCfg, consumerCfg, func(context.Context, amqp.Delivery) error { return nil }, nil)
	require.NoError(t, consumer.Start(context.Background()))

	time.Sleep(1500 * time.Millisecond)

	ch, err := conn.Channel(context.Background())
	require.NoError(t, err)
	require.Same(t, shared, ch)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, consumer.Shutdown(ctx))
	require.NoError(t, conn.Close())
}
