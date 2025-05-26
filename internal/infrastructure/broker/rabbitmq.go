package broker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageBroker interface {
	PublishToQueue(ctx context.Context, queue string, body []byte) error
	PublishToTopic(ctx context.Context, topic string, body []byte) error
	Close() error
}

type RabbitMQPublisher struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
	log          *slog.Logger
}

func NewPublisher(url string, log *slog.Logger, exchangeName, queueName string) (*RabbitMQPublisher, error) {
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
	if err := ch.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	if _, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("queue declare: %w", err)
	}
	if err := ch.QueueBind(
		queueName,
		"email-sending",
		exchangeName,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return &RabbitMQPublisher{conn: conn, channel: ch, log: log}, nil
}

func (r *RabbitMQPublisher) PublishToQueue(ctx context.Context, queue string, body []byte) error {
	pub := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
	}

	if err := r.channel.PublishWithContext(
		ctx,
		r.exchangeName,
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

func (r *RabbitMQPublisher) PublishToTopic(ctx context.Context, topic string, body []byte) error {
	pub := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
	}

	if err := r.channel.PublishWithContext(
		ctx,
		r.exchangeName,
		topic,
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
