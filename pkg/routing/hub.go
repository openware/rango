package routing

import (
	"fmt"
	"strings"

	msg "github.com/openware/rango/pkg/message"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

type Request struct {
	client *Client
	msg.Request
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Register Requests from the clients.
	Requests chan Request

	// Unregister requests from clients.
	Unregister chan *Client

	// List of clients registered to Topics
	PublicTopics map[string]*Topic
}

type Message struct {
	Scope  string // global, public, private
	Stream string
	Type   string
	Topic  string
	Body   []byte
}

func NewHub() *Hub {
	return &Hub{
		Requests:     make(chan Request),
		Unregister:   make(chan *Client),
		PublicTopics: make(map[string]*Topic),
	}
}

func getTopic(s, t string) string {
	return s + "." + t
}

func (h *Hub) ListenWebsocketEvents() {
	for {
		select {
		case req := <-h.Requests:
			h.handleRequest(req)

		case client := <-h.Unregister:
			h.unsubscribeAll(client)
			close(client.send)

		}
	}
}

func (h *Hub) ListenAMQP(q <-chan amqp.Delivery) {
	for {
		delivery := <-q
		if log.Logger.GetLevel() <= zerolog.DebugLevel {
			log.Debug().Msgf("AMQP msg received: %s -> %s", delivery.RoutingKey, delivery.Body)
		}
		s := strings.Split(delivery.RoutingKey, ".")
		switch len(s) {
		case 2:
			msg := Message{
				Scope:  s[0],
				Stream: "",
				Type:   s[1],
				Topic:  getTopic(s[0], s[1]),
				Body:   delivery.Body,
			}

			h.routeMessage(&msg)

		case 3:
			msg := Message{
				Scope:  s[0],
				Stream: s[1],
				Type:   s[2],
				Topic:  getTopic(s[1], s[2]),
				Body:   delivery.Body,
			}

			h.routeMessage(&msg)

		default:
			log.Error().Msgf("Bad routing key: %s", delivery.RoutingKey)
		}
		delivery.Ack(true)
	}
}

func (h *Hub) routeMessage(msg *Message) {
	switch msg.Scope {
	case "public", "global":
		topic, ok := h.PublicTopics[msg.Stream]
		if ok {
			topic.broadcast(msg)
		}

	case "private":
		// TODO

	default:
		log.Error().Msgf("Invalid message scope %s", msg.Scope)
	}

}

func (h *Hub) unsubscribeAll(client *Client) {
	for _, topic := range h.PublicTopics {
		topic.unsubscribe(client)
	}
}

func responseMust(e error, r interface{}) []byte {
	res, err := msg.Response(e, r)
	if err != nil {
		log.Panic().Msg("responseMust failed:" + err.Error())
		panic(err.Error())
	}

	return res
}

func (h *Hub) handleRequest(req Request) {
	switch req.Method {
	case "subscribe":
		h.handleSubscribe(req)
	case "unsubscribe":
		h.handleUnsubscribe(req)
	default:
		// req.client.send <- responseMust(1, errors.New("unsupported method"), nil)
	}
}

func getStringArgs(params []string) ([]string, error) {
	topics := make([]string, len(params))

	for i, t := range params {
		topics[i] = t
	}

	return topics, nil
}

func (h *Hub) handleSubscribe(req Request) {
	topics, err := getStringArgs(req.Streams)
	if err != nil {
		req.client.send <- responseMust(err, nil)
	}

	for _, t := range topics {
		topic, ok := h.PublicTopics[t]
		if !ok {
			topic = NewTopic(h)
		}

		message := make(map[string]string)
		message["message"] = "subscribed"
		message["streams"] = t

		req.client.send <- responseMust(nil, message)
		topic.subscribe(req.client)
	}
}

func (h *Hub) handleUnsubscribe(req Request) {
	topics, err := getStringArgs(req.Streams)
	if err != nil {
		req.client.send <- responseMust(err, nil)
	}

	fmt.Println(topics)
	for _, t := range topics {
		topic, ok := h.PublicTopics[t]
		if !ok {
			req.client.send <- responseMust(fmt.Errorf("Topic does not exist %s", t), nil)
			return
		}

		topic.unsubscribe(req.client)
	}
}
