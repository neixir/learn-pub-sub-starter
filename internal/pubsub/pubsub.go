package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType string

const (
	QueueTypeDurable   SimpleQueueType = "durable"
	QueueTypeTransient SimpleQueueType = "transient"
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
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
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

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T),
) error {
	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	// Call DeclareAndBind to make sure that the given queue exists and is bound to the exchange
	queue, err := channel.QueueDeclare(queueName, queueType == QueueTypeDurable, queueType == QueueTypeTransient, queueType == QueueTypeTransient, false, nil)
	if err != nil {
		return err
	}

	err = channel.QueueBind(queueName, key, exchange, false, amqp.Table{})
	if err != nil {
		return err
	}

	// 2. Get a new chan of amqp.Delivery structs by using the channel.Consume method.
	// 2.1. Use an empty string for the consumer name so that it will be auto-generated
	// 2.2 Set all other parameters to false/nil
	// func (ch *Channel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args Table) (<-chan Delivery, error)
	deliveries, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	// 3. Start a goroutine that ranges over the channel of deliveries, and for each message:
	go func() {
		for del := range deliveries {
			// 3.1 Unmarshal the body (raw bytes) of each message delivery into the (generic) T type.
			var t T
			err := json.Unmarshal(del.Body, &t)
			if err != nil {
				fmt.Println(err.Error())
			}

			// 3.2 Call the given handler function with the unmarshaled message
			handler(t)

			// 3.3 Acknowledge the message with delivery.Ack(false) to remove it from the queue
			del.Ack(false)
		}
	}()

	return nil
}
