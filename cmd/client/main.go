package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	// Creating the connection
	connectionString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connectionString)
	if err != nil {
		fmt.Println("Something happened creating the connection.")
		return
	}

	defer conn.Close()

	fmt.Println("Connection successful!")

	// Declare and Bind
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println("Something happened welcoming the client.")
		return
	}

	queueName := routing.PauseKey + "." + username
	channel, queue, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.QueueTypeTransient)
	if err != nil {
		fmt.Println("Something happened declaring and binding.")
		return
	}

	// Create the game state
	gamestate := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queue.Name, routing.PauseKey, pubsub.QueueTypeTransient, handlerPause(gamestate))
	if err != nil {
		fmt.Println(err.Error())
	}

	// CH4 L4
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, routing.ArmyMovesPrefix+".*", pubsub.QueueTypeTransient, handlerMove(gamestate))
	if err != nil {
		fmt.Println(err.Error())
	}

	quitGame := false
	for !quitGame {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}

		switch input[0] {
		case "spawn":
			gamestate.CommandSpawn(input)
		case "move":
			move, err := gamestate.CommandMove(input)
			if err != nil {
				fmt.Println(err)
			} else {
				routingKey := routing.ArmyMovesPrefix + "." + username
				err = pubsub.PublishJSON(channel, routing.ExchangePerilTopic, routingKey, move)
				// Log success message
				if err != nil {
					fmt.Println("Failed to publish move:", err)
				} else {
					fmt.Println("Move published successfully")
				}
			}

		case "status":
			gamestate.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			quitGame = true
		default:
			fmt.Println("Unknown command.")
		}
	}

	// fmt.Println("Press Enter to exit...")
	// fmt.Scanln()
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(move)
	}
}
