package routing

import (
	"encoding/json"

	msg "github.com/openware/rango/pkg/message"
	"github.com/rs/zerolog/log"
)

type Topic struct {
	hub     *Hub
	clients map[IClient]struct{}
}

func NewTopic(h *Hub) *Topic {
	return &Topic{
		clients: make(map[IClient]struct{}),
		hub:     h,
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

func (t *Topic) broadcastRaw(topic, msgBody string) {
	for client := range t.clients {
		client.Send(msgBody)
	}
}

func (t *Topic) subscribe(c IClient) bool {
	if _, ok := t.clients[c]; ok {
		return false
	}
	t.clients[c] = struct{}{}

	return true
}

func (t *Topic) unsubscribe(c IClient) bool {
	_, ok := t.clients[c]
	delete(t.clients, c)

	return ok
}
