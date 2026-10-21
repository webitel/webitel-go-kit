package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	scopeName = "github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq"

	defaultExchange = "amq.default"

	operationPublish = "publish"
	operationProcess = "process"
	operationAck     = "ack"
	operationNack    = "nack"
)

func newTracer(tp trace.TracerProvider) trace.Tracer {
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	return tp.Tracer(scopeName)
}

func startPublishSpan(
	ctx context.Context,
	tracer trace.Tracer,
	exchange string,
	routingKey string,
	messageID string,
	body []byte,
	headers amqp.Table,
) (context.Context, trace.Span, amqp.Table) {
	attrs := []attribute.KeyValue{
		semconv.MessagingSystemRabbitMQ,
		semconv.MessagingOperationTypeSend,
		semconv.MessagingOperationName(operationPublish),
		semconv.MessagingDestinationName(destinationName(exchange)),
		semconv.MessagingMessageID(messageID),
		semconv.MessagingMessageBodySize(len(body)),
	}
	if routingKey != "" {
		attrs = append(attrs, semconv.MessagingRabbitMQDestinationRoutingKey(routingKey))
	}

	ctx, span := tracer.Start(
		extractHeaders(ctx, headers),
		fmt.Sprintf("%s %s", operationPublish, destinationName(exchange)),
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attrs...),
	)

	return ctx, span, InjectHeaders(ctx, headers)
}

// startProcessSpan starts consumer span as child of publisher span from message headers.
func startProcessSpan(
	ctx context.Context,
	tracer trace.Tracer,
	queue string,
	msg amqp.Delivery,
) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		semconv.MessagingSystemRabbitMQ,
		semconv.MessagingOperationTypeProcess,
		semconv.MessagingOperationName(operationProcess),
		semconv.MessagingDestinationName(destinationName(msg.Exchange)),
		semconv.MessagingDestinationSubscriptionName(queue),
		semconv.MessagingMessageBodySize(len(msg.Body)),
		semconv.MessagingRabbitMQMessageDeliveryTag(int(msg.DeliveryTag)),
	}
	if msg.RoutingKey != "" {
		attrs = append(attrs, semconv.MessagingRabbitMQDestinationRoutingKey(msg.RoutingKey))
	}
	if msg.MessageId != "" {
		attrs = append(attrs, semconv.MessagingMessageID(msg.MessageId))
	}

	return tracer.Start(
		extractHeaders(ctx, msg.Headers),
		fmt.Sprintf("%s %s", operationProcess, queue),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
}

// startSettleSpan starts ack/nack span as child of process span.
func startSettleSpan(
	ctx context.Context,
	tracer trace.Tracer,
	operation string,
	queue string,
	msg amqp.Delivery,
) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		semconv.MessagingSystemRabbitMQ,
		semconv.MessagingOperationTypeSettle,
		semconv.MessagingOperationName(operation),
		semconv.MessagingDestinationName(destinationName(msg.Exchange)),
		semconv.MessagingDestinationSubscriptionName(queue),
		semconv.MessagingRabbitMQMessageDeliveryTag(int(msg.DeliveryTag)),
	}
	if msg.MessageId != "" {
		attrs = append(attrs, semconv.MessagingMessageID(msg.MessageId))
	}

	return tracer.Start(
		ctx,
		fmt.Sprintf("%s %s", operation, queue),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
}

func failSpan(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func destinationName(exchange string) string {
	if exchange == "" {
		return defaultExchange
	}

	return exchange
}
