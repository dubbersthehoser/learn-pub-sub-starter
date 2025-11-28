package main

import (
	"fmt"
	"log"
	//"os"
	//"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
)

func main() {
	ConnString := "amqp://guest:guest@localhost:5672"
	conn, err := amqp.Dial(ConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting Peril server...")

	bindParams := pubsub.DeclareAndBindParams{
		Connection: conn,
		Exchange: routing.ExchangePerilTopic,
		QueueName: routing.GameLogSlug,
		Key: routing.GameLogSlug + ".*",
		QueueType: pubsub.Durable,
	}
	_, _, err = pubsub.DeclareAndBind(bindParams)
	if err != nil {
		log.Fatal(err)
	}


	gamelogic.PrintServerHelp()
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		exe := input[0]
		switch exe {
		case "pause":
			log.Println("sending a pause")
			msg := routing.PlayingState{
				IsPaused: true,
			}
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, msg)
			if err != nil {
				log.Println("error:", err)
			}

		case "resume":
			log.Println("sending a resume")
			msg := routing.PlayingState{
				IsPaused: false,
			}
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, msg)
			if err != nil {
				log.Println("error:", err)
			}
		
		case "quit":
			log.Println("got a 'quit', exiting")
			goto end
		default:
			log.Printf("unkown command '%s'", exe)
		}
	}
	end:
	fmt.Println("Closing Peril server.")
	//signals := make(chan os.Signal, 1)
	//signal.Notify(signals, os.Interrupt)
	//_ = <-signals
	conn.Close()

}
