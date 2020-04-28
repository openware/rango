package routing

import (
	"encoding/json"
	"errors"
	"strings"

	msg "github.com/openware/rango/pkg/message"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

type Request struct {
	client IClient
	msg.Request
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Register Requests from the clients.
	Requests chan Request

	// Unregister requests from clients.
	Unregister chan IClient

	// List of clients registered to public topics
	PublicTopics map[string]*Topic

	// List of clients registered to private topics
	PrivateTopics map[string]map[string]*Topic
}

type Event struct {
	Scope  string                 // global, public, private
	Stream string                 // channel routing key
	Type   string                 // event type
	Topic  string                 // topic routing key (stream.type)
	Body   map[string]interface{} // event json body
}

func NewHub() *Hub {
	return &Hub{
		Requests:      make(chan Request),
		Unregister:    make(chan IClient),
		PublicTopics:  make(map[string]*Topic, 100),
		PrivateTopics: make(map[string]map[string]*Topic, 1000),
	}
}

func getTopic(scope, stream, typ string) string {
	if scope == "private" {
		return typ
	}
	return stream + "." + typ
}

func (h *Hub) ListenWebsocketEvents() {
	for {
		select {
		case req := <-h.Requests:
			h.handleRequest(req)

		case client := <-h.Unregister:
			log.Info().Msgf("Unregistering client %s", client.GetUID())
			h.unsubscribeAll(client)
			client.Close()
		}
	}
}

func (h *Hub) ListenAMQP(q <-chan amqp.Delivery) {
	for {
		delivery := <-q
		if log.Logger.GetLevel() <= zerolog.TraceLevel {
			log.Trace().Msgf("AMQP msg received: %s -> %s", delivery.RoutingKey, delivery.Body)
		}
		s := strings.Split(delivery.RoutingKey, ".")

		o := make(map[string]interface{})
		err := json.Unmarshal(delivery.Body, &o)

		if err != nil {
			log.Error().Msgf("JSON parse error: %s, msg: %s", err.Error(), delivery.Body)
		}

		switch len(s) {
		case 2:
			msg := Event{
				Scope:  s[0],
				Stream: "",
				Type:   s[1],
				Topic:  getTopic(s[0], s[0], s[1]),
				Body:   o,
			}

			h.routeMessage(&msg)

		case 3:
			msg := Event{
				Scope:  s[0],
				Stream: s[1],
				Type:   s[2],
				Topic:  getTopic(s[0], s[1], s[2]),
				Body:   o,
			}

			h.routeMessage(&msg)

		default:
			log.Error().Msgf("Bad routing key: %s", delivery.RoutingKey)
		}
		delivery.Ack(true)
	}
}

func (h *Hub) routeMessage(msg *Event) {
	log.Info().Msgf("Routing message %v", msg)
	switch msg.Scope {
	case "public", "global":
		topic, ok := h.PublicTopics[msg.Topic]
		if ok {
			topic.broadcast(msg)
		} else {
			if log.Logger.GetLevel() <= zerolog.DebugLevel {
				log.Debug().Msgf("No public registration to %s", msg.Topic)
				log.Debug().Msgf("Public topics: %v", h.PublicTopics)
			}
		}

	case "private":
		uid := msg.Stream
		uTopic, ok := h.PrivateTopics[uid]
		if ok {
			topic, ok := uTopic[msg.Topic]
			if ok {
				topic.broadcast(msg)
				break
			}
		}
		if log.Logger.GetLevel() <= zerolog.DebugLevel {
			log.Debug().Msgf("No private registration to %s", msg.Topic)
			log.Debug().Msgf("Public topics: %v", h.PrivateTopics)
		}

	default:
		log.Error().Msgf("Invalid message scope %s", msg.Scope)
	}

}

func (h *Hub) unsubscribeAll(client IClient) {
	for _, topic := range h.PublicTopics {
		topic.unsubscribe(client)
	}
}

func responseMust(e error, r interface{}) string {
	res, err := msg.PackOutgoingResponse(e, r)
	if err != nil {
		log.Panic().Msg("responseMust failed:" + err.Error())
		panic(err.Error())
	}

	return string(res)
}

func isPrivateStream(s string) bool {
	return strings.Count(s, ".") == 0
}

func (h *Hub) handleRequest(req Request) {
	switch req.Method {
	case "subscribe":
		h.handleSubscribe(req)
	case "unsubscribe":
		h.handleUnsubscribe(req)
	default:
		req.client.Send(responseMust(errors.New("unsupported method"), nil))
	}
}

func (h *Hub) handleSubscribe(req Request) {
	for _, t := range req.Streams {
		if isPrivateStream(t) {
			uid := req.client.GetUID()
			if uid == "" {
				log.Error().Msgf("Anonymous user tries to subscribe to private stream %s", t)
				continue
			}

			uTopics, ok := h.PrivateTopics[uid]
			if !ok {
				uTopics = make(map[string]*Topic, 3)
				h.PrivateTopics[uid] = uTopics
			}

			topic, ok := uTopics[t]
			if !ok {
				topic = NewTopic(h)
				uTopics[t] = topic
			}

			topic.subscribe(req.client)
			req.client.SubscribePrivate(t)
		} else {
			topic, ok := h.PublicTopics[t]
			if !ok {
				topic = NewTopic(h)
				h.PublicTopics[t] = topic
			}

			topic.subscribe(req.client)
			req.client.SubscribePublic(t)
		}
	}

	log.Debug().Msgf("Public topics: %v", h.PublicTopics)

	req.client.Send(responseMust(nil, map[string]interface{}{
		"message": "subscribed",
		"streams": req.client.GetSubscriptions(),
	}))
}

func (h *Hub) handleUnsubscribe(req Request) {
	for _, t := range req.Streams {
		if isPrivateStream(t) {
			uid := req.client.GetUID()
			if uid == "" {
				continue
			}
			uTopics, ok := h.PrivateTopics[uid]
			if !ok {
				continue
			}

			topic := uTopics[t]
			if ok {
				topic.unsubscribe(req.client)
				if topic.len() == 0 {
					delete(uTopics, t)
				}
				req.client.UnsubscribePrivate(t)
			}

			uTopics, ok = h.PrivateTopics[uid]
			if ok && len(uTopics) == 0 {
				delete(h.PrivateTopics, uid)
			}

		} else {
			topic, ok := h.PublicTopics[t]
			if ok {
				topic.unsubscribe(req.client)
				if topic.len() == 0 {
					delete(h.PublicTopics, t)
				}
				req.client.UnsubscribePublic(t)
			}
		}
	}

	req.client.Send(responseMust(nil, map[string]interface{}{
		"message": "unsubscribed",
		"streams": req.client.GetSubscriptions(),
	}))
}
