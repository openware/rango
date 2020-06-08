package routing

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	msg "github.com/openware/rango/pkg/message"
	"github.com/openware/rango/pkg/metrics"
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

	// Storage for incremental objects
	IncrementalObjects map[string]*IncrementalObject

	mutex sync.Mutex
}

type Event struct {
	Scope  string      // global, public, private
	Stream string      // channel routing key
	Type   string      // event type
	Topic  string      // topic routing key (stream.type)
	Body   interface{} // event json body
}

type IncrementalObject struct {
	Snapshot   string
	Increments []string
}

func NewHub() *Hub {
	return &Hub{
		Requests:           make(chan Request),
		Unregister:         make(chan IClient),
		PublicTopics:       make(map[string]*Topic, 100),
		PrivateTopics:      make(map[string]map[string]*Topic, 1000),
		IncrementalObjects: make(map[string]*IncrementalObject, 5),
	}
}

func isIncrementObject(s string) bool {
	return strings.HasSuffix(s, "-inc")
}

func isSnapshotObject(s string) bool {
	return strings.HasSuffix(s, "-snap")
}

func isDebug() bool {
	return log.Logger.GetLevel() <= zerolog.DebugLevel
}

func isTrace() bool {
	return log.Logger.GetLevel() <= zerolog.TraceLevel
}

func getTopic(scope, stream, typ string) string {
	if isSnapshotObject(typ) {
		typ = strings.Replace(typ, "-snap", "-inc", 1)
	}
	if scope == "private" {
		return typ
	}
	return stream + "." + typ
}

func (h *Hub) ListenWebsocketEvents() {
	for {
		select {
		case req := <-h.Requests:
			h.handleRequest(&req)

		case client := <-h.Unregister:
			log.Info().Msgf("Unregistering client (%s)", client.GetUID())
			h.unsubscribeAll(client)
			client.Close()
		}
	}
}

// ReceiveMsg handles AMQP messages
func (h *Hub) ReceiveMsg(delivery amqp.Delivery) {
	if isTrace() {
		log.Trace().Msgf("AMQP msg received: %s -> %s", delivery.RoutingKey, delivery.Body)
	}
	s := strings.Split(delivery.RoutingKey, ".")

	var o interface{}
	err := json.Unmarshal(delivery.Body, &o)

	if err != nil {
		log.Error().Msgf("JSON parse error: %s, msg: %s", err.Error(), delivery.Body)
		return
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
}

func (h *Hub) handleSnapshot(msg *Event) (string, error) {
	topic := msg.Stream + "." + msg.Type
	body, err := json.Marshal(map[string]interface{}{
		topic: msg.Body,
	})

	if err != nil {
		return "", err
	}

	o, ok := h.IncrementalObjects[msg.Topic]
	if !ok {
		o = &IncrementalObject{}
		h.IncrementalObjects[msg.Topic] = o
	}
	o.Snapshot = string(body)
	o.Increments = []string{}

	return string(body), nil
}

func (h *Hub) handleIncrement(msg *Event) (string, error) {
	body, err := json.Marshal(map[string]interface{}{
		msg.Topic: msg.Body,
	})

	if err != nil {
		return "", err
	}

	o, ok := h.IncrementalObjects[msg.Topic]
	if !ok {
		return "", fmt.Errorf("No snapshot received before the increment for topic %s, ignoring", msg.Topic)
	}
	o.Increments = append(o.Increments, string(body))
	return string(body), nil

}

func (h *Hub) routeMessage(msg *Event) {
	if isTrace() {
		log.Trace().Msgf("Routing message %v", msg)
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()

	switch msg.Scope {
	case "public", "global":
		topic, ok := h.PublicTopics[msg.Topic]

		switch {
		case isIncrementObject(msg.Type):
			rm, err := h.handleIncrement(msg)
			if err != nil {
				log.Error().Msgf("handleIncrement failed: %s", err.Error())
				return
			}
			if ok {
				topic.broadcastRaw(msg.Topic, rm)
			}
			return
		case isSnapshotObject(msg.Type):
			_, err := h.handleSnapshot(msg)
			if err != nil {
				log.Error().Msgf("handleSnapshot failed: %s", err.Error())
				return
			}
			return
		}

		if ok {
			topic.broadcast(msg)
		} else {
			if isTrace() {
				log.Trace().Msgf("No public registration to %s", msg.Topic)
				log.Trace().Msgf("Public topics: %v", h.PublicTopics)
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
		if isTrace() {
			log.Trace().Msgf("No private registration to %s", msg.Topic)
			log.Trace().Msgf("Private topics: %v", h.PrivateTopics)
		}

	default:
		log.Error().Msgf("Invalid message scope %s", msg.Scope)
	}

}

func (h *Hub) unsubscribeAll(client IClient) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for t, topic := range h.PublicTopics {
		if topic.unsubscribe(client) {
			metrics.RecordHubUnsubscription("public", t)
		}
		if topic.len() == 0 {
			delete(h.PublicTopics, t)
		}
	}

	uid := client.GetUID()
	topics, ok := h.PrivateTopics[uid]
	if !ok {
		return
	}

	for t, topic := range topics {
		if topic.unsubscribe(client) {
			metrics.RecordHubUnsubscription("private", t)
		}
		if topic.len() == 0 {
			delete(topics, t)
		}
	}

	if len(topics) == 0 {
		delete(h.PrivateTopics, uid)
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

func (h *Hub) handleRequest(req *Request) {
	switch req.Method {
	case "subscribe":
		h.handleSubscribe(req)
	case "unsubscribe":
		h.handleUnsubscribe(req)
	default:
		req.client.Send(responseMust(errors.New("unsupported method"), nil))
	}
}

func (h *Hub) handleSubscribe(req *Request) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for _, t := range req.Streams {
		if isPrivateStream(t) {
			uid := req.client.GetUID()
			if uid == "" {
				log.Error().Msgf("Anonymous user tried to subscribe to private stream %s", t)
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

			if topic.subscribe(req.client) {
				metrics.RecordHubSubscription("private", t)
				req.client.SubscribePrivate(t)
			}
		} else {
			topic, ok := h.PublicTopics[t]
			if !ok {
				topic = NewTopic(h)
				h.PublicTopics[t] = topic
			}

			if topic.subscribe(req.client) {
				metrics.RecordHubSubscription("public", t)
				req.client.SubscribePublic(t)
			}

			if isIncrementObject(t) {
				o, ok := h.IncrementalObjects[t]
				if ok && o.Snapshot != "" {
					req.client.Send(o.Snapshot)
					for _, inc := range o.Increments {
						req.client.Send(inc)
					}
				}
			}
		}
	}

	req.client.Send(responseMust(nil, map[string]interface{}{
		"message": "subscribed",
		"streams": req.client.GetSubscriptions(),
	}))
}

func (h *Hub) handleUnsubscribe(req *Request) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

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

			topic, ok := uTopics[t]
			if ok {
				if topic.unsubscribe(req.client) {
					metrics.RecordHubUnsubscription("private", t)
					req.client.UnsubscribePrivate(t)
				}

				if topic.len() == 0 {
					delete(uTopics, t)
				}
			}

			uTopics, ok = h.PrivateTopics[uid]
			if ok && len(uTopics) == 0 {
				delete(h.PrivateTopics, uid)
			}

		} else {
			topic, ok := h.PublicTopics[t]
			if ok {
				if topic.unsubscribe(req.client) {
					metrics.RecordHubUnsubscription("public", t)
					req.client.UnsubscribePublic(t)
				}

				if topic.len() == 0 {
					delete(h.PublicTopics, t)
				}
			}
		}
	}

	req.client.Send(responseMust(nil, map[string]interface{}{
		"message": "unsubscribed",
		"streams": req.client.GetSubscriptions(),
	}))
}
