package routing

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openware/rango/pkg/metrics"
	"github.com/openware/rango/pkg/msg"
	"github.com/rs/zerolog/log"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// FIXME: IClient looks very wrong.
type IClient interface {
	Send(string)
	Close()
	GetUID() string
	GetPublicSubscriptions() []interface{}
	GetPrivateSubscriptions() []interface{}
	SubscribePublic(string)
	SubscribePrivate(string)
	UnsubscribePublic(string)
	UnsubscribePrivate(string)
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// User ID if authorized
	UID string

	pubSub  []interface{}
	privSub []interface{}

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// NewClient handles websocket requests from the peer.
func NewClient(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Msg("Websocket upgrade failed: " + err.Error())
		return
	}
	client := &Client{
		hub:     hub,
		conn:    conn,
		send:    make(chan []byte, 256),
		UID:     r.Header.Get("JwtUID"),
		pubSub:  []interface{}{},
		privSub: []interface{}{},
	}

	if client.UID == "" {
		log.Info().Msgf("New anonymous connection")
	} else {
		log.Info().Msgf("New authenticated connection: %s", client.UID)
	}

	metrics.RecordHubClientNew()

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.write()
	go client.read()
}

func (c *Client) Send(s string) {
	c.send <- []byte(s)
}

func (c *Client) Close() {
	close(c.send)
}

func (c *Client) GetUID() string {
	return c.UID
}

func (c *Client) GetPublicSubscriptions() []interface{} {
	return c.pubSub
}

func (c *Client) GetPrivateSubscriptions() []interface{} {
	return c.privSub
}

func (c *Client) SubscribePublic(s string) {
	if !msg.Contains(c.pubSub, s) {
		c.pubSub = append(c.pubSub, s)
	}
}

func (c *Client) SubscribePrivate(s string) {
	if !msg.Contains(c.privSub, s) {
		c.privSub = append(c.privSub, s)
	}
}

func (c *Client) UnsubscribePublic(s string) {
	l := make([]interface{}, len(c.pubSub)-1)
	i := 0
	for _, el := range c.pubSub {
		if s != el {
			l[i] = el
			i++
		}
	}
	c.pubSub = l
}

func (c *Client) UnsubscribePrivate(s string) {
	l := make([]interface{}, len(c.privSub)-1)
	i := 0
	for _, el := range c.privSub {
		if s != el {
			l[i] = el
			i++
		}
	}
	c.privSub = l
}

// read pumps messages from the websocket connection to the hub.
//
// The application runs read in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) read() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Info().Msgf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		if len(message) == 0 {
			continue
		}
		if isDebug() {
			log.Debug().Msgf("Received message %s", message)
		}

		// handle ping
		if string(message) == "ping" {
			c.send <- []byte("pong")
			continue
		}

		req, err := msg.Parse(message)
		if err != nil {
			log.Error().Msgf("fail to parse message: %s", err.Error())

			resp, err := msg.NewResponse(&msg.Msg{ReqID: 0}, "error", []interface{}{err.Error()}).Encode()
			if err == nil {
				c.send <- resp
			}

			continue
		}

		log.Debug().Msgf("Pushing request to hub: %v", req)
		c.hub.Requests <- Request{c, req}
	}
}

// write pumps messages from the hub to the websocket connection.
//
// A goroutine running write is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		metrics.RecordHubClientClose()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
