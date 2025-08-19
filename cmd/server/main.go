package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	// Creating the connection
	connectionString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connectionString)

	if err != nil {
		fmt.Println("Something happened creating the connection.")
		return
	}

	defer conn.Close()

	fmt.Println("Connection successful!")

	// Creating the channel
	channel, err := conn.Channel()
	if err != nil {
		fmt.Println("Something happened creating the channel.")
		return
	}

	queueName := routing.GameLogSlug
	_, err = channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		fmt.Println("Something happened declaring the queue.")
		return
	}

	// func (ch *Channel) QueueBind(name, key, exchange string, noWait bool, args Table) error
	err = channel.QueueBind(queueName, routing.GameLogSlug+".*", routing.ExchangePerilTopic, false, amqp.Table{})
	if err != nil {
		fmt.Println("Something happened binding the queue.")
		return
	}

	data := routing.PlayingState{}

	gamelogic.PrintServerHelp()
	quitGame := false
	for !quitGame {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}

		switch input[0] {
		case "pause":
			// log to the console that you're sending a pause message, and publish the pause message as you were doing before.
			fmt.Println("Pausing game.")
			data.IsPaused = true
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, data)
		case "resume":
			// log to the console that you're sending a resume message, and publish the resume message as you were doing before.
			fmt.Println("Resuming game.")
			data.IsPaused = false
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, data)
		case "quit":
			// log to the console that you're exiting, and break out of the loop.
			fmt.Println("Quitting game.")
			quitGame = true
		default:
			// If it's anything else, log to the console that you don't understand the command.
			fmt.Println("Unknown command.")

		}
	}

	// Wait for a signal (e.g. Ctrl+C) to exit the program.
	// If a signal is received, print a message to the console that the program is shutting down and close the connection.

	// The Go standard library has a package called os/signal that you can use to listen for signals.
	// Here's an example of how you can listen for a signal:
	// wait for ctrl+c
	/*
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		<-signalChan

		fmt.Println("Shutting down...")
	*/

}
