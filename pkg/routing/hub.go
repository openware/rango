package routing

import (
	"fmt"

	msg "github.com/openware/rango/pkg/message"
	"github.com/openware/rango/pkg/upstream"
	"github.com/rs/zerolog/log"
)

type Request struct {
	client *Client
	msg.Request
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	// Inbound Messages from the clients.
	Messages chan upstream.Msg

	// Register Requests from the clients.
	Requests chan Request

	// Unregister requests from clients.
	Unregister chan *Client

	// List of clients registered to Topics
	Topics map[string]*Topic
}

func NewHub() *Hub {
	return &Hub{
		Requests:   make(chan Request),
		Unregister: make(chan *Client),
		Topics:     make(map[string]*Topic),
		Messages:   make(chan upstream.Msg),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case req := <-h.Requests:
			h.handleRequest(req)

		case client := <-h.Unregister:
			h.unsubscribeAll(client)
			close(client.send)

		case message := <-h.Messages:
			topic, ok := h.Topics[message.Channel]
			if !ok {
				topic = NewTopic(h)
				h.Topics[message.Channel] = topic
			}

			topic.broadcast(message)
		}
	}
}

func (h *Hub) unsubscribeAll(client *Client) {
	for _, topic := range h.Topics {
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
		h.hanldeSubscribe(req)
	case "unsubscribe":
		h.hanldeUnsubscribe(req)
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

func (h *Hub) hanldeSubscribe(req Request) {
	topics, err := getStringArgs(req.Streams)
	if err != nil {
		req.client.send <- responseMust(err, nil)
	}

	for _, t := range topics {
		topic, ok := h.Topics[t]
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

func (h *Hub) hanldeUnsubscribe(req Request) {
	topics, err := getStringArgs(req.Streams)
	if err != nil {
		req.client.send <- responseMust(err, nil)
	}

	fmt.Println(topics)
	for _, t := range topics {
		topic, ok := h.Topics[t]
		if !ok {
			req.client.send <- responseMust(fmt.Errorf("Topic does not exist %s", t), nil)
			return
		}

		topic.unsubscribe(req.client)
	}
}
