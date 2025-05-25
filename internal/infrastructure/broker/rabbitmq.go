package broker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageBroker interface {
	Publish(ctx context.Context, queue string, body []byte) error
	Close() error
}

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	log     *slog.Logger
}

func NewPublisher(url string, log *slog.Logger) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	if err := ch.Confirm(false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("rabbitmq confirm: %w", err)
	}

	return &RabbitMQPublisher{conn: conn, channel: ch, log: log}, nil
}

func (r *RabbitMQPublisher) Publish(ctx context.Context, queue string, body []byte) error {
	if _, err := r.channel.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("queue declare: %w", err)
	}

	pub := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
	}

	if err := r.channel.PublishWithContext(
		ctx,
		"",
		queue,
		false,
		false,
		pub,
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	select {
	case confirm := <-r.channel.NotifyPublish(make(chan amqp.Confirmation, 1)):
		if !confirm.Ack {
			return fmt.Errorf("rabbitmq nack")
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (r *RabbitMQPublisher) Close() error {
	_ = r.channel.Close()
	return r.conn.Close()
}
