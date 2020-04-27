package routing

import (
	"container/ring"

	msg "github.com/openware/rango/pkg/message"
	"github.com/openware/rango/pkg/upstream"
	"github.com/rs/zerolog/log"
)

const TopicBufferSize = 10

type Topic struct {
	hub     *Hub
	clients map[*Client]struct{}

	buffer *ring.Ring
}

func NewTopic(h *Hub) *Topic {
	return &Topic{
		clients: make(map[*Client]struct{}),
		hub:     h,
		buffer:  ring.New(TopicBufferSize),
	}
}

func eventMust(method string, data interface{}) []byte {
	ev, err := msg.Event(method, data)
	if err != nil {
		log.Panic().Msg(err.Error())
	}

	return ev
}

func (t *Topic) broadcast(message upstream.Msg) {
	if t == nil {
		panic("Topic not initialized")
	}

	t.buffer.Value = message
	t.buffer = t.buffer.Next()

	for client := range t.clients {
		select {
		case client.send <- eventMust(message.Channel, message.Message):
		default:
			t.hub.Unregister <- client
		}
	}
}

func (t *Topic) subscribe(c *Client) {
	if _, ok := t.clients[c]; ok {
		return
	}
	t.clients[c] = struct{}{}
}

func (t *Topic) unsubscribe(c *Client) {
	delete(t.clients, c)
}
