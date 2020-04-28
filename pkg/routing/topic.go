package routing

import (
	"container/ring"
	"encoding/json"

	msg "github.com/openware/rango/pkg/message"
	"github.com/rs/zerolog/log"
)

const TopicBufferSize = 10

type Topic struct {
	hub     *Hub
	clients map[IClient]struct{}

	buffer *ring.Ring
}

func NewTopic(h *Hub) *Topic {
	return &Topic{
		clients: make(map[IClient]struct{}),
		hub:     h,
		buffer:  ring.New(TopicBufferSize),
	}
}

func eventMust(method string, data interface{}) []byte {
	ev, err := msg.PackOutgoingEvent(method, data)
	if err != nil {
		log.Panic().Msg(err.Error())
	}

	return ev
}

func contains(list []string, el string) bool {
	for _, l := range list {
		if l == el {
			return true
		}
	}
	return false
}

func (t *Topic) len() int {
	return len(t.clients)
}

func (t *Topic) broadcast(message *Event) {
	t.buffer.Value = message
	t.buffer = t.buffer.Next()

	body, err := json.Marshal(map[string]interface{}{
		message.Topic: message.Body,
	})

	if err != nil {
		log.Error().Msgf("Fail to JSON marshal: %s", err.Error())
		return
	}

	for client := range t.clients {
		client.Send(string(body))
	}
}

func (t *Topic) subscribe(c IClient) {
	if _, ok := t.clients[c]; ok {
		return
	}
	t.clients[c] = struct{}{}
}

func (t *Topic) unsubscribe(c IClient) {
	delete(t.clients, c)
}
