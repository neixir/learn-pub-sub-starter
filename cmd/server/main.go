package main

import (
	"fmt"
	"os"
	"os/signal"

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

	// Publishing something
	data := routing.PlayingState{
		IsPaused: true,
	}

	pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, data)

	// Wait for a signal (e.g. Ctrl+C) to exit the program.
	// If a signal is received, print a message to the console that the program is shutting down and close the connection.

	// The Go standard library has a package called os/signal that you can use to listen for signals.
	// Here's an example of how you can listen for a signal:
	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("Shutting down...")

}
