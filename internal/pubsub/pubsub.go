package pubsub

import (
	"encoding/json"
	"context"
	"errors"
	"log"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)


func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	ctx := context.Background()
	jdata, err := json.Marshal(val)
	if err != nil {
		return err
	}

	pub := amqp.Publishing{
		ContentType: "application/json", 
		Body: jdata,
	}

	ch.PublishWithContext(ctx, exchange, key, false, false, pub)
	return nil
}

type AckType string

const (
	Ack AckType = "ACK"
	NackRequeue = "NACK_REQUEUE"
	NackDiscard = "NACK_DISCARD"
)

type SubscribeParams struct {
	Connection *amqp.Connection
	Exchange   string
	QueueName  string
	Key        string
	QueueType SimpleQueueType
}

func SubscribeJSON[T any](sub *SubscribeParams, handler func(T) AckType) error {
	bindParams := DeclareAndBindParams{
		Connection: sub.Connection,
		Exchange: sub.Exchange,
		QueueName: sub.QueueName,
		Key: sub.Key,
		QueueType: sub.QueueType,
	}
	ch, _, err := DeclareAndBind(bindParams)
	if err != nil {
		return err
	}

	delivery, err := ch.Consume("", "", false, false, false, false, nil)  
	if err != nil {
		return err
	}
	
	go func() {
		for d := range delivery {
			var value T
			err := json.Unmarshal(d.Body, &value)
			if err != nil {
				log.Fatal(err)
			}
			ack := handler(value)
			switch ack {
			case Ack:
				fmt.Println(string(ack))
				d.Ack(false)
			case NackRequeue:
				fmt.Println(string(ack))
				d.Nack(false, true)
			case NackDiscard:
				fmt.Println(string(ack))
				d.Nack(false, false)
			}
		}
	}()
	
	return nil
}

type SimpleQueueType string
const (
	Durable   SimpleQueueType = "durable"
	Transient                 = "transient"
)

type DeclareAndBindParams struct {
	Connection *amqp.Connection
	Exchange   string
	QueueName  string
	Key        string
	QueueType  SimpleQueueType
}

func DeclareAndBind(params DeclareAndBindParams) (*amqp.Channel, amqp.Queue, error) {
	conn := params.Connection
	exchange := params.Exchange
	queueType := params.QueueType
	queueName := params.QueueName
	key := params.Key

	if conn == nil {
		return nil, amqp.Queue{}, errors.New("pubsub: connection is nil")
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	var (
		isDirable    bool
		isAutoDelete bool
		isExclusive  bool
		isNoWait     bool = false
	)

	switch queueType { 
	case Durable:
		isDirable = true
	case Transient:
		isAutoDelete = true
		isExclusive = true
	}

	queue, err := ch.QueueDeclare(queueName, isDirable, isAutoDelete, isExclusive, isNoWait, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	err = ch.QueueBind(queueName, key, exchange, isNoWait, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return ch, queue, nil
}
