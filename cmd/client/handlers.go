package main

import (
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		fmt.Printf("Ack, because it's a PAUSE")
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, channel *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(move gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)

		if outcome == gamelogic.MoveOutComeSafe {
			fmt.Printf("Ack, because move outcome is 'safe'")
			return pubsub.Ack
		}

		if outcome == gamelogic.MoveOutcomeMakeWar {
			fmt.Printf("NackRequeue, because move outcome is 'make war'")

			rw := gamelogic.RecognitionOfWar{
				Attacker: move.Player,
				Defender: gs.GetPlayerSnap(),
			}

			err := pubsub.PublishJSON(channel, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix+"."+gs.GetUsername(), rw)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				return pubsub.NackRequeue
			}

			return pubsub.Ack
		}

		if outcome == gamelogic.MoveOutcomeSamePlayer {
			fmt.Printf("NackDiscard, because move outcome is 'same player'")
			return pubsub.NackDiscard
		}

		fmt.Printf("NackDiscard, because move outcome is... something else")
		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState, channel *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(rw gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")

		outcome, winner, loser := gs.HandleWar(rw)

		var msg string
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			msg = fmt.Sprintf("%s won a war against %s", winner, loser)
		case gamelogic.WarOutcomeYouWon:
			msg = fmt.Sprintf("%s won a war against %s", winner, loser)
		case gamelogic.WarOutcomeDraw:
			msg = fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
		default:
			fmt.Println("error, outcome not valid")
			return pubsub.NackDiscard
		}

		if msg != "" {
			gl := routing.GameLog{
				CurrentTime: time.Now(),
				Message:     msg,
				Username:    gs.GetUsername(),
			}

			//err := PublishGamelogic(channel, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gs.GetUsername(), gl)
			pubsub.PublishJSON(channel, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gs.GetUsername(), gl)
			//return err
		}

		// No hauriem d'arribar aqui
		return pubsub.NackDiscard
	}
}

func PublishGamelogic(channel *amqp.Channel, exchange, key string, gl routing.GameLog) pubsub.Acktype {
	err := pubsub.PublishGob(channel, exchange, key, gl)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return pubsub.NackRequeue
	}

	return pubsub.Ack

}
