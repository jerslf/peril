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
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, publishCh *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(mv gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		moveOutcome := gs.HandleMove(mv)
		switch moveOutcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				publishCh,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.GetUsername(),
				gamelogic.RecognitionOfWar{Attacker: mv.Player, Defender: gs.GetPlayerSnap()},
			)
			if err != nil {
				fmt.Printf("could not publish ArmyMove: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		}
		fmt.Println("error: unknown war outcome")
		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState, publishCh *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(rw gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(rw)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			msg := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := pubsub.PublishGameLog(
				publishCh,
				routing.ExchangePerilTopic,
				routing.GameLogSlug+"."+gs.GetUsername(),
				time.Now(),
				msg,
				gs.GetUsername(),
			)
			if err != nil {
				fmt.Printf("could not publish GameLog: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			msg := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := pubsub.PublishGameLog(
				publishCh,
				routing.ExchangePerilTopic,
				routing.GameLogSlug+"."+gs.GetUsername(),
				time.Now(),
				msg,
				gs.GetUsername(),
			)
			if err != nil {
				fmt.Printf("could not publish GameLog: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			msg := fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			err := pubsub.PublishGameLog(
				publishCh,
				routing.ExchangePerilTopic,
				routing.GameLogSlug+"."+gs.GetUsername(),
				time.Now(),
				msg,
				gs.GetUsername(),
			)
			if err != nil {
				fmt.Printf("could not publish GameLog: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}
		fmt.Printf("Outcome error")
		return pubsub.NackDiscard
	}
}
