package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrPublishFailed  = errors.New("message publish failed")
	ErrPublishTimeout = errors.New("publish confirmation timeout")
	ErrMessageNacked  = errors.New("message nacked by broker")
)

type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte, headers amqp.Table) error
	Close() error
}

var _ Publisher = (*MessagePublisher)(nil)

type MessagePublisher struct {
	broker    *ConnectionBroker
	cfg       *RabbitConfig
	ch        *amqp091.Channel
	confirmCh chan amqp091.Confirmation
	mu        sync.Mutex
}

func NewPublisher(broker *ConnectionBroker, cfg *RabbitConfig) (*MessagePublisher, error) {
	ch, err := broker.GetChannel(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher channel: %w", err)
	}

	if err := ch.Confirm(false); err != nil {
		return nil, fmt.Errorf("failed to enable confirm mode: %w", err)
	}

	return &MessagePublisher{
		broker:    broker,
		cfg:       cfg,
		ch:        ch,
		confirmCh: ch.NotifyPublish(make(chan amqp091.Confirmation, 1)),
	}, nil
}

func (p *MessagePublisher) Publish(
	ctx context.Context,
	routingKey string,
	body []byte,
	headers amqp091.Table,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for attempt := 0; attempt < p.cfg.Publisher.MaxRetries; attempt++ {
		err := p.publishWithConfirmation(ctx, routingKey, body, headers)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}

	return fmt.Errorf("%w: after %d attempts", ErrPublishFailed, p.cfg.Publisher.MaxRetries)
}

func (p *MessagePublisher) publishWithConfirmation(
	ctx context.Context,
	routingKey string,
	body []byte,
	headers amqp091.Table,
) error {
	err := p.ch.Publish(
		p.cfg.Exchange.Name,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
			Headers:     headers,
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	select {
	case confirm := <-p.confirmCh:
		if !confirm.Ack {
			return ErrMessageNacked
		}
		return nil
	case <-time.After(p.cfg.Publisher.ConfirmationTimeout):
		return ErrPublishTimeout
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *MessagePublisher) Close() error {
	return p.ch.Close()
}
