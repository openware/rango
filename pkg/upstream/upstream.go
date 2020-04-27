package upstream

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

type Msg struct {
	Channel string
	Message map[string][]interface{}
}

type tradeJSON struct {
	TID       uint64 `json:"id"`
	Price     string `json:"price"`
	Amount    string `json:"amount"`
	Date      uint64 `json:"date"`
	TakerType string `json:"taker_type"`
}

// {"ethusd.trades":{"trades":[{"tid":438195050,"amount":"0.023","price":193.76,"date":1587890939,"taker_type":"buy"}]}}
func AMQPUpstream(output chan Msg) {
	url := "amqp://127.0.0.1:5672"
	connection, err := amqp.Dial(url)
	if err != nil {
		panic("could not establish connection with RabbitMQ:" + err.Error())
	}
	channel, err := connection.Channel()

	if err != nil {
		panic("could not open RabbitMQ channel:" + err.Error())
	}

	err = channel.ExchangeDeclare("peatio.events.ranger", "topic", false, false, false, false, nil)

	if err != nil {
		panic(err)
	}

	msgs, err := channel.Consume("test", "", false, false, false, false, nil)

	if err != nil {
		panic("error consuming the queue: " + err.Error())
	}

	for msg := range msgs {
		fmt.Println(msg.RoutingKey)
		fmt.Println("message received: " + string(msg.Body))
		msg.Ack(false)
	}
	msg := Msg{"ethusd.trades", nil}

	// msg.Message["trades"] = {"trades":[{"tid":438195050,"amount":"0.023","price":193.76,"date":1587890939,"taker_type":"buy"}]}
	trade := tradeJSON{438195050, "0.023", "193.76", 1587890939, "buy"}
	msg.Message = make(map[string][]interface{})
	msg.Message["trades"] = append(msg.Message["trades"], trade)
	for {
		time.Sleep(2 * time.Second)
		output <- msg
	}
}
