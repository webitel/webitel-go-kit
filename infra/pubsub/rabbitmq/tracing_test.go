package rabbitmq

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestPublishProcessSpans(t *testing.T) {
	setW3CPropagator(t)
	tracer, recorder := newTestTracer(t)

	pubCtx, pubSpan, headers := startPublishSpan(testContext(t), tracer, "events", "file.created", "m1", []byte("body"), amqp.Table{"a": "b"})
	require.Equal(t, pubSpan, trace.SpanFromContext(pubCtx))
	pubSpan.End()

	msg := amqp.Delivery{
		Headers:     headers,
		Exchange:    "events",
		RoutingKey:  "file.created",
		MessageId:   "m1",
		DeliveryTag: 5,
		Body:        []byte("body"),
	}
	processCtx, processSpan := startProcessSpan(context.Background(), tracer, "storage.events", msg)
	processSpan.End()

	require.Equal(t, "7", baggage.FromContext(processCtx).Member("webitel.domain.id").Value())

	spans := recorder.Ended()
	require.Len(t, spans, 2)
	pubSpanRec, processSpanRec := spans[0], spans[1]

	require.Equal(t, "publish events", pubSpanRec.Name())
	require.Equal(t, trace.SpanKindProducer, pubSpanRec.SpanKind())
	require.Equal(t, testTraceID, pubSpanRec.SpanContext().TraceID().String())
	require.Equal(t, testSpanID, pubSpanRec.Parent().SpanID().String())
	require.Subset(t, pubSpanRec.Attributes(), []attribute.KeyValue{
		semconv.MessagingSystemRabbitMQ,
		semconv.MessagingOperationTypeSend,
		semconv.MessagingOperationName("publish"),
		semconv.MessagingDestinationName("events"),
		semconv.MessagingRabbitMQDestinationRoutingKey("file.created"),
		semconv.MessagingMessageID("m1"),
		semconv.MessagingMessageBodySize(4),
	})

	require.Equal(t, "process storage.events", processSpanRec.Name())
	require.Equal(t, trace.SpanKindConsumer, processSpanRec.SpanKind())
	require.Equal(t, testTraceID, processSpanRec.SpanContext().TraceID().String())
	require.Equal(t, pubSpanRec.SpanContext().SpanID(), processSpanRec.Parent().SpanID())
	require.True(t, processSpanRec.Parent().IsRemote())
	require.Empty(t, processSpanRec.Links())
	require.Subset(t, processSpanRec.Attributes(), []attribute.KeyValue{
		semconv.MessagingSystemRabbitMQ,
		semconv.MessagingOperationTypeProcess,
		semconv.MessagingOperationName("process"),
		semconv.MessagingDestinationName("events"),
		semconv.MessagingDestinationSubscriptionName("storage.events"),
		semconv.MessagingRabbitMQDestinationRoutingKey("file.created"),
		semconv.MessagingRabbitMQMessageDeliveryTag(5),
		semconv.MessagingMessageID("m1"),
		semconv.MessagingMessageBodySize(4),
	})
}

func TestPublishSpanCallerHeadersParent(t *testing.T) {
	setW3CPropagator(t)
	tracer, recorder := newTestTracer(t)

	const otherTraceID = "11111111111111111111111111111111"
	captured := amqp.Table{"traceparent": "00-" + otherTraceID + "-2222222222222222-01"}

	_, span, headers := startPublishSpan(testContext(t), tracer, "", "jobs", "m2", nil, captured)
	span.End()

	pubSpanRec := recorder.Ended()[0]
	require.Equal(t, "publish amq.default", pubSpanRec.Name())
	require.Equal(t, otherTraceID, pubSpanRec.SpanContext().TraceID().String())
	require.Equal(t, "2222222222222222", pubSpanRec.Parent().SpanID().String())

	require.Equal(t, amqp.Table{"traceparent": "00-" + otherTraceID + "-2222222222222222-01"}, captured)
	require.Equal(t, amqp.Table{
		"traceparent": "00-" + otherTraceID + "-" + pubSpanRec.SpanContext().SpanID().String() + "-01",
		"baggage":     testBaggage,
	}, headers)
}

func TestPublishSpanNoopProvider(t *testing.T) {
	setW3CPropagator(t)

	_, span, headers := startPublishSpan(testContext(t), noop.NewTracerProvider().Tracer(""), "events", "", "m3", nil, nil)
	span.End()

	require.Equal(t, amqp.Table{
		"traceparent": testTraceparent,
		"baggage":     testBaggage,
	}, headers)
}

func TestProcessMessageSpan(t *testing.T) {
	setW3CPropagator(t)
	tracer, recorder := newTestTracer(t)

	ackErr := errors.New("channel closed")

	tests := []struct {
		name              string
		handlerErr        error
		settleErr         error
		wantProcessStatus codes.Code
		wantSettleStatus  codes.Code
		wantSettleName    string
		wantAck           bool
	}{
		{name: "ack", wantSettleName: "ack q", wantAck: true},
		{name: "nack", handlerErr: errors.New("boom"), wantProcessStatus: codes.Error, wantSettleName: "nack q"},
		{name: "ack error", settleErr: ackErr, wantSettleStatus: codes.Error, wantSettleName: "ack q", wantAck: true},
		{name: "nack error", handlerErr: errors.New("boom"), settleErr: ackErr, wantProcessStatus: codes.Error, wantSettleStatus: codes.Error, wantSettleName: "nack q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder.Reset()

			var handlerSpan trace.SpanContext
			c := &MessageConsumer{
				queue:    &QueueConfig{Name: "q"},
				consumer: &ConsumerConfig{ProcessingTimeout: time.Second},
				handler: func(ctx context.Context, _ amqp.Delivery) error {
					handlerSpan = trace.SpanContextFromContext(ctx)
					return tt.handlerErr
				},
				workerSem: make(chan struct{}, 1),
				logger:    componentLogger(nil),
				tracer:    tracer,
			}

			ack := &testAcknowledger{err: tt.settleErr}
			msg := amqp.Delivery{
				Acknowledger: ack,
				Headers:      amqp.Table{"traceparent": testTraceparent},
				MessageId:    "m4",
				DeliveryTag:  5,
			}

			c.workerSem <- struct{}{}
			c.wg.Add(1)
			c.processMessage(context.Background(), msg)

			spans := recorder.Ended()
			require.Len(t, spans, 2)
			settle, process := spans[0], spans[1]

			require.Equal(t, "process q", process.Name())
			require.Equal(t, process.SpanContext(), handlerSpan)
			require.Equal(t, testTraceID, process.SpanContext().TraceID().String())
			require.Equal(t, testSpanID, process.Parent().SpanID().String())
			require.Equal(t, tt.wantProcessStatus, process.Status().Code)

			require.Equal(t, tt.wantSettleName, settle.Name())
			require.Equal(t, trace.SpanKindClient, settle.SpanKind())
			require.Equal(t, process.SpanContext(), settle.Parent())
			require.Equal(t, tt.wantSettleStatus, settle.Status().Code)
			require.Contains(t, settle.Attributes(), semconv.MessagingOperationTypeSettle)
			require.Contains(t, settle.Attributes(), semconv.MessagingMessageID("m4"))
			require.Contains(t, settle.Attributes(), semconv.MessagingRabbitMQMessageDeliveryTag(5))

			require.Equal(t, tt.wantAck, ack.acked)
			require.Equal(t, !tt.wantAck, ack.nacked)

			if tt.handlerErr != nil {
				require.Len(t, process.Events(), 1)
				require.Equal(t, "exception", process.Events()[0].Name)
			} else {
				require.Empty(t, process.Events())
			}
			if tt.settleErr != nil {
				require.Len(t, settle.Events(), 1)
				require.Equal(t, "exception", settle.Events()[0].Name)
			}
		})
	}
}

func newTestTracer(t *testing.T) (trace.Tracer, *tracetest.SpanRecorder) {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	return newTracer(tp), recorder
}

type testAcknowledger struct {
	mu     sync.Mutex
	err    error
	acked  bool
	nacked bool
}

func (a *testAcknowledger) Ack(uint64, bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.acked = true
	return a.err
}

func (a *testAcknowledger) Nack(uint64, bool, bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nacked = true
	return a.err
}

func (a *testAcknowledger) Reject(uint64, bool) error {
	return nil
}
