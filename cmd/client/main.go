package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

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
	channel, _, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.SimpleQueueTypeTransient)
	if err != nil {
		fmt.Println("Something happened declaring and binding.")
		return
	}

	/*
		queueName := routing.GameLogSlug + "." + username
		channel, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.GameLogSlug, pubsub.SimpleQueueTypeTransient)
		if err != nil {
			fmt.Println("Something happened declaring and binding.")
			return
		}
	*/

	// Create the game state
	gamestate := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, routing.PauseKey+"."+gamestate.GetUsername(), routing.PauseKey, pubsub.SimpleQueueTypeTransient, handlerPause(gamestate))
	if err != nil {
		//fmt.Println(err.Error())
		log.Fatal(err.Error())
	}

	// CH4 L4
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, routing.ArmyMovesPrefix+".*", pubsub.SimpleQueueTypeTransient, handlerMove(gamestate, channel))
	if err != nil {
		//fmt.Println(err.Error())
		log.Fatal(err.Error())
	}

	// CH5 L6
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "war", routing.WarRecognitionsPrefix+".*", pubsub.SimpleQueueTypeDurable, handlerWar(gamestate, channel))
	if err != nil {
		//fmt.Println(err.Error())
		log.Fatal(err.Error())
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
			//fmt.Println("Spamming not allowed yet!")
			// 1. Ensure that a second "word" was provided in the command. E.g. spam 10 or spam 1000.
			// Convert that word into an integer.
			if len(input) > 1 {
				n, err := strconv.Atoi(input[1])
				if err != nil {
					fmt.Println("Not a number")
					break
				}

				// 2. Do the following n times, where n is the integer from the command:

				for i := 0; i < n; i++ { //i := range n {

					// 2.1 Use gamelogic.GetMaliciousLog to get a malicious log message.
					msg := gamelogic.GetMaliciousLog()

					// 2.2 Publish the log message (a struct) to Rabbit. Use the following parameters:
					// Exchange: peril_topic
					// Key: game_logs.username, where username is the username of the player
					gl := routing.GameLog{
						CurrentTime: time.Now(),
						Message:     msg,
						Username:    username,
					}
					PublishGamelogic(channel, routing.ExchangePerilTopic, routing.GameLogSlug+"."+username, gl)
				}

			}

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
