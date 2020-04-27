package main

import (
	"fmt"
	"log"

	msg "github.com/openware/rango/pkg/message"
	"github.com/openware/rango/pkg/upstream"
)

type Request struct {
	client *Client
	msg.Request
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	// Inbound messages from the clients.
	messages chan upstream.Msg

	// Register requests from the clients.
	requests chan Request

	// Unregister requests from clients.
	unregister chan *Client

	// List of clients registered to topics
	topics map[string]*Topic
}

func newHub() *Hub {
	return &Hub{
		requests:   make(chan Request),
		unregister: make(chan *Client),
		topics:     make(map[string]*Topic),
		messages:   make(chan upstream.Msg),
	}
}

func (h *Hub) run() {
	for {
		select {
		case req := <-h.requests:
			h.handleRequest(req)

		case client := <-h.unregister:
			h.unsubscribeAll(client)
			close(client.send)

		case message := <-h.messages:
			topic, ok := h.topics[message.Channel]
			if !ok {
				topic = NewTopic(h)
				h.topics[message.Channel] = topic
			}

			topic.broadcast(message)
		}
	}
}

func (h *Hub) unsubscribeAll(client *Client) {
	for _, topic := range h.topics {
		topic.unsubscribe(client)
	}
}

func responseMust(e error, r interface{}) []byte {
	res, err := msg.Response(e, r)
	if err != nil {
		log.Panic(err)
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
		topic, ok := h.topics[t]
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
		topic, ok := h.topics[t]
		if !ok {
			req.client.send <- responseMust(fmt.Errorf("Topic does not exist %s", t), nil)
			return
		}

		topic.unsubscribe(req.client)
	}
}
