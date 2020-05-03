package routing

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openware/rango/pkg/metrics"
	"github.com/openware/rango/pkg/msg"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

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
}

// Request is a container for client message and pointer to the client
type Request struct {
	client IClient
	*msg.Msg
}

// Event contains an event received through AMQP
type Event struct {
	Scope  string      // global, public, private
	Stream string      // channel routing key
	Type   string      // event type
	Topic  string      // topic routing key (stream.type)
	Body   interface{} // event json body
}

// IncrementalObject stores an incremental object built from a snapshot and increments
type IncrementalObject struct {
	Snapshot   string
	Increments []string
}

// NewHub creates a Hub
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
			resp := h.handleRequest(&req)
			req.client.Send(string(resp.Encode()))

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
		if isTrace() {
			log.Trace().Msgf("AMQP msg received: %s -> %s", delivery.RoutingKey, delivery.Body)
		}
		s := strings.Split(delivery.RoutingKey, ".")

		var o interface{}
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

func (h *Hub) handleRequest(req *Request) (resp *msg.Msg) {
	var err error

	switch req.Method {
	case "subscribe":
		err, resp = h.handleSubscribe(req)
	case "unsubscribe":
		err, resp = h.handleUnsubscribe(req)
	default:
		return msg.NewResponse(req.Msg, "error", []interface{}{"Unknown method " + req.Method})
	}

	if err != nil {
		return msg.NewResponse(req.Msg, "error", []interface{}{err.Error()})
	}
	return resp
}

func (h *Hub) handleSubscribePublic(client IClient, streams []string) {
	for _, t := range streams {
		topic, ok := h.PublicTopics[t]
		if !ok {
			topic = NewTopic(h)
			h.PublicTopics[t] = topic
		}

		topic.subscribe(client)
		client.SubscribePublic(t)
		if isIncrementObject(t) {
			o, ok := h.IncrementalObjects[t]
			if ok && o.Snapshot != "" {
				client.Send(o.Snapshot)
				for _, inc := range o.Increments {
					client.Send(inc)
				}
			}
		}
	}
}

func (h *Hub) handleSubscribePrivate(client IClient, uid string, streams []string) {
	for _, t := range streams {
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

		topic.subscribe(client)
		client.SubscribePrivate(t)
	}
}

func (h *Hub) handleSubscribe(req *Request) (error, *msg.Msg) {
	args := req.Msg.Args
	if len(args) != 2 {
		return errors.New("method expects exactly 2 arguments"), nil
	}

	scope, err := msg.ParseString(args[0])
	if err != nil {
		return errors.New("first argument must be a string"), nil
	}

	streams, err := msg.ParseSliceOfStrings(args[1])
	if err != nil {
		return errors.New("second argument must be a list of strings"), nil
	}

	switch scope {
	case "public":
		h.handleSubscribePublic(req.client, streams)

	case "private":
		uid := req.client.GetUID()
		if uid != "" {
			h.handleSubscribePrivate(req.client, uid, streams)
		} else {
			return errors.New("unauthorized"), nil
		}

	default:
		return errors.New("Unexpected scope " + scope), nil
	}

	return nil, msg.NewResponse(req.Msg, "subscribed", msg.Convss2is(req.client.GetSubscriptions()))
}

func (h *Hub) handleUnsubscribe(req *Request) (error, *msg.Msg) {
	args := req.Msg.Args
	if len(args) != 2 {
		return errors.New("method expects exactly 2 arguments"), nil
	}

	scope, ok := args[0].(string)
	if !ok {
		return errors.New("first argument must be a string"), nil
	}

	streams, ok := args[1].([]string)
	if !ok {
		return errors.New("second argument must be a list of strings"), nil
	}
	switch scope {
	case "public":
		h.handleUnsubscribePublic(req.client, streams)

	case "private":
		uid := req.client.GetUID()
		if uid != "" {
			h.handleUnsubscribePrivate(req.client, uid, streams)
		} else {
			return errors.New("unauthorized"), nil
		}

	default:
		return errors.New("Unexpected scope " + scope), nil
	}

	return nil, msg.NewResponse(req.Msg, "unsubscribed", msg.Convss2is(req.client.GetSubscriptions()))
}

func (h *Hub) handleUnsubscribePublic(client IClient, streams []string) {
	for _, t := range streams {
		topic, ok := h.PublicTopics[t]
		if ok {
			if topic.unsubscribe(client) {
				client.UnsubscribePublic(t)
				metrics.RecordHubUnsubscription("public", t)
			}
			if topic.len() == 0 {
				delete(h.PublicTopics, t)
			}
		}
	}
}

func (h *Hub) handleUnsubscribePrivate(client IClient, uid string, streams []string) {
	uTopics, ok := h.PrivateTopics[uid]
	if !ok {
		return
	}

	for _, t := range streams {
		topic := uTopics[t]
		if ok {
			if topic.unsubscribe(client) {
				client.UnsubscribePrivate(t)
				metrics.RecordHubUnsubscription("private", t)
			}
			if topic.len() == 0 {
				delete(uTopics, t)
			}
		}
	}

	if len(uTopics) == 0 {
		delete(h.PrivateTopics, uid)
	}
}
