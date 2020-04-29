package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:     hub,
		send:    make(chan []byte, 256),
		UID:     "UIDABC001",
		pubSub:  []string{},
		privSub: []string{},
	}

	assert.Equal(t, "UIDABC001", client.GetUID())
	assert.Equal(t, []string{}, client.GetSubscriptions())

	client.SubscribePublic("a.x")
	assert.Equal(t, []string{"a.x"}, client.GetSubscriptions())
	assert.Equal(t, []string{"a.x"}, client.pubSub)
	assert.Equal(t, []string{}, client.privSub)

	client.SubscribePublic("a.y")
	assert.Equal(t, []string{"a.x", "a.y"}, client.GetSubscriptions())
	assert.Equal(t, []string{"a.x", "a.y"}, client.pubSub)
	assert.Equal(t, []string{}, client.privSub)

	client.UnsubscribePublic("a.y")
	assert.Equal(t, []string{"a.x"}, client.GetSubscriptions())
	assert.Equal(t, []string{"a.x"}, client.pubSub)
	assert.Equal(t, []string{}, client.privSub)

	client.SubscribePrivate("b")
	assert.Equal(t, []string{"a.x", "b"}, client.GetSubscriptions())
	assert.Equal(t, []string{"a.x"}, client.pubSub)
	assert.Equal(t, []string{"b"}, client.privSub)

	client.SubscribePrivate("c")
	assert.Equal(t, []string{"a.x", "b", "c"}, client.GetSubscriptions())
	assert.Equal(t, []string{"a.x"}, client.pubSub)
	assert.Equal(t, []string{"b", "c"}, client.privSub)

	client.UnsubscribePrivate("b")
	assert.Equal(t, []string{"a.x", "c"}, client.GetSubscriptions())
	assert.Equal(t, []string{"a.x"}, client.pubSub)
	assert.Equal(t, []string{"c"}, client.privSub)

	client.UnsubscribePrivate("c")
	assert.Equal(t, []string{"a.x"}, client.GetSubscriptions())
	assert.Equal(t, []string{"a.x"}, client.pubSub)
	assert.Equal(t, []string{}, client.privSub)

	client.UnsubscribePublic("a.x")
	assert.Equal(t, []string{}, client.GetSubscriptions())
	assert.Equal(t, []string{}, client.pubSub)
	assert.Equal(t, []string{}, client.privSub)
}

func TestParseStreamsFromURI(t *testing.T) {
	assert.Equal(t, []string{}, parseStreamsFromURI("/?"))
	assert.Equal(t, []string{}, parseStreamsFromURI(""))
	assert.Equal(t, []string{"aaa", "bbb"}, parseStreamsFromURI("/?stream=aaa&stream=bbb"))
	assert.Equal(t, []string{"aaa", "bbb"}, parseStreamsFromURI("/?stream=aaa,bbb"))
}
