package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

type simpleQueueType string

const (
	QueueTypeDurable   simpleQueueType = "durable"
	QueueTypeTransient simpleQueueType = "transient"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}

	pub := amqp.Publishing{
		ContentType: "application/json",
		Body:        b,
	}

	ch.PublishWithContext(context.Background(), exchange, key, false, false, pub)

	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType simpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	// The durable parameter should only be true if queueType is durable.
	// The autoDelete parameter should be true if queueType is transient.
	// The exclusive parameter should be true if queueType is transient.
	queue, err := channel.QueueDeclare(queueName, queueType == QueueTypeDurable, queueType == QueueTypeTransient, queueType == QueueTypeTransient, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	// func (ch *Channel) QueueBind(name, key, exchange string, noWait bool, args Table) error
	err = channel.QueueBind(queueName, key, exchange, false, amqp.Table{})

	return channel, queue, err

}
