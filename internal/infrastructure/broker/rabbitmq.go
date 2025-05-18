package broker

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQPublisher реализует MessageBroker
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewPublisher соединяется и открывает канал
func NewPublisher(url string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &RabbitMQPublisher{conn: conn, channel: ch}, nil
}

// Publish кладёт body в очередь queue (JSON)
func (r *RabbitMQPublisher) Publish(queue string, body []byte) error {
	_, err := r.channel.QueueDeclare(
		queue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}
	return r.channel.Publish(
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// Close закрывает канал и соединение
func (r *RabbitMQPublisher) Close() error {
	r.channel.Close()
	return r.conn.Close()
}
