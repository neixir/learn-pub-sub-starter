package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType string

const (
	SimpleQueueTypeDurable   SimpleQueueType = "durable"
	SimpleQueueTypeTransient SimpleQueueType = "transient"
)

type Acktype int

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
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

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, pub)

	return err
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
	// It should pass in an amqp.Table to the QueueDeclare function that includes a x-dead-letter-exchange key.
	// The value should be the name of your dead letter exchange ("peril_idx")
	table := amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	}

	queue, err := channel.QueueDeclare(queueName, queueType == SimpleQueueTypeDurable, queueType == SimpleQueueTypeTransient, queueType == SimpleQueueTypeTransient, false, table)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = channel.QueueBind(queueName, key, exchange, false, table)

	return channel, queue, err

}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	// func (ch *Channel) Qos(prefetchCount, prefetchSize int, global bool) error
	// CH7 L5 https://www.boot.dev/lessons/e1e10f9d-beda-4d0e-b948-a7ab800fb936
	// Update your consumption code. It should call channel.Qos before calling channel.Consume. Limit the prefetch count to 10.
	err = channel.Qos(10, 1, true)

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
		for msg := range deliveries {
			// 3.1 Unmarshal the body (raw bytes) of each message delivery into the (generic) T type.
			var t T
			err := json.Unmarshal(msg.Body, &t)
			if err != nil {
				fmt.Println(err.Error())
			}

			ack := handler(t)

			// Depending on the returned "acktype", the goroutine that calls the handler should either call...
			switch ack {
			case Ack:
				msg.Ack(false)
			case NackRequeue:
				msg.Nack(false, true)
			case NackDiscard:
				msg.Nack(false, false)
			}

		}
	}()

	return nil
}

// Adaptat de PublishJSON
func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(val)

	if err != nil {
		return err
	}

	pub := amqp.Publishing{
		ContentType: "application/gob",
		Body:        buf.Bytes(),
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, pub)

	return err
}

// Adaptat de SubscribeJSON
func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	deliveries, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range deliveries {
			var t T
			buf := bytes.NewBuffer(msg.Body)
			dec := gob.NewDecoder(buf)
			err := dec.Decode(&t)
			if err != nil {
				// TODO?
			}
			ack := handler(t)

			// Depending on the returned "acktype", the goroutine that calls the handler should either call...
			switch ack {
			case Ack:
				msg.Ack(false)
			case NackRequeue:
				msg.Nack(false, true)
			case NackDiscard:
				msg.Nack(false, false)
			}

		}
	}()

	return nil
}

/*
func encode(gl GameLog) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(gl)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decode(data []byte) (GameLog, error) {
	var gamelog GameLog
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&gamelog)

	return gamelog, err
}
*/
