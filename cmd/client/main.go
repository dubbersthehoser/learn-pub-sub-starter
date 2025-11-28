package main

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
)

func handlerPause(state *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		state.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(state *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(gm gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		mov := state.HandleMove(gm)
		fmt.Println(mov)
		switch mov {
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			return pubsub.Ack
		default:
			return pubsub.NackDiscard
		}
	}
}

func main() {
	fmt.Println("Starting Peril client...")
	ConnString := "amqp://guest:guest@localhost:5672"
	conn, err := amqp.Dial(ConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
	}

	// bind to pauses
	queueNamePause := routing.PauseKey + "." + userName
	keyPause := routing.PauseKey
	bindParams := pubsub.DeclareAndBindParams{
		Connection: conn,
		Exchange: routing.ExchangePerilDirect,
		QueueName: queueNamePause,
		Key: keyPause,
		QueueType: pubsub.Transient,
	}
	_, _, err = pubsub.DeclareAndBind(bindParams)
	if err != nil {
		log.Fatal(err)
	}

	queueNameMove := routing.ArmyMovesPrefix + "." + userName
	keyMoves := routing.ArmyMovesPrefix + ".*"
	bindParams = pubsub.DeclareAndBindParams{
		Connection: conn,
		Exchange: routing.ExchangePerilTopic,
		QueueName: queueNameMove,
		Key: keyMoves,
		QueueType: pubsub.Transient,
	}
	chMove, _, err := pubsub.DeclareAndBind(bindParams)
	if err != nil {
		log.Fatal(err)
	}

	state := gamelogic.NewGameState(userName)

	PauseSub := pubsub.SubscribeParams{
		Connection: conn, 
		Exchange: routing.ExchangePerilDirect,
		QueueName: queueNamePause,
		Key: routing.PauseKey,
		QueueType: pubsub.Transient,
	}

	err = pubsub.SubscribeJSON(
		&PauseSub,
		handlerPause(state),
	)
	if err != nil {
		log.Fatal(err)
	}

	MoveSub := pubsub.SubscribeParams{
		Connection: conn, 
		Exchange: routing.ExchangePerilTopic,
		QueueName: routing.ArmyMovesPrefix + "." + userName,
		Key: routing.ArmyMovesPrefix + ".*",
		QueueType: pubsub.Transient,
	}

	err = pubsub.SubscribeJSON(
		&MoveSub,
		handlerMove(state),
	)
	if err != nil {
		log.Fatal(err)
	}

	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		exe := input[0]

		switch exe {
		case "spawn":
			err = state.CommandSpawn(input)
			if err != nil {
				fmt.Println(err)
			}
		case "move":
			army, err := state.CommandMove(input)
			if err != nil {
				fmt.Println(err)
			} else {
				number := input[2]
				msg := routing.PlayingState{}
				key := routing.ArmyMovesPrefix + "." + userName
				err = pubsub.PublishJSON(chMove, routing.ExchangePerilTopic, key, msg)
				if err != nil {
					log.Println("error:", err)
				} else {
					fmt.Printf("moved %s to %s\n", number, army.ToLocation)
				}
			}

		case "status":
			state.CommandStatus()
			
		case "help":
			gamelogic.PrintClientHelp()

		case "spam":
			fmt.Println("spam is forbidden")

		case "quit":
			gamelogic.PrintQuit()
			goto endREPL
		default:
			fmt.Println("error: unknown command")
		}
	}
	endREPL:
}

