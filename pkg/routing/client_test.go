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
		pubSub:  []interface{}{},
		privSub: []interface{}{},
	}

	assert.Equal(t, "UIDABC001", client.GetUID())
	assert.Equal(t, []interface{}{}, client.GetPublicSubscriptions())
	assert.Equal(t, []interface{}{}, client.GetPrivateSubscriptions())

	client.SubscribePublic("a.x")
	assert.Equal(t, []interface{}{"a.x"}, client.GetPublicSubscriptions())
	assert.Equal(t, []interface{}{"a.x"}, client.pubSub)
	assert.Equal(t, []interface{}{}, client.privSub)

	client.SubscribePublic("a.y")
	assert.Equal(t, []interface{}{"a.x", "a.y"}, client.GetPublicSubscriptions())
	assert.Equal(t, []interface{}{"a.x", "a.y"}, client.pubSub)
	assert.Equal(t, []interface{}{}, client.privSub)

	client.UnsubscribePublic("a.y")
	assert.Equal(t, []interface{}{"a.x"}, client.GetPublicSubscriptions())
	assert.Equal(t, []interface{}{"a.x"}, client.pubSub)
	assert.Equal(t, []interface{}{}, client.privSub)

	client.SubscribePrivate("b")
	assert.Equal(t, []interface{}{"b"}, client.GetPrivateSubscriptions())
	assert.Equal(t, []interface{}{"a.x"}, client.pubSub)
	assert.Equal(t, []interface{}{"b"}, client.privSub)

	client.SubscribePrivate("c")
	assert.Equal(t, []interface{}{"b", "c"}, client.GetPrivateSubscriptions())
	assert.Equal(t, []interface{}{"a.x"}, client.pubSub)
	assert.Equal(t, []interface{}{"b", "c"}, client.privSub)

	client.UnsubscribePrivate("b")
	assert.Equal(t, []interface{}{"c"}, client.GetPrivateSubscriptions())
	assert.Equal(t, []interface{}{"a.x"}, client.pubSub)
	assert.Equal(t, []interface{}{"c"}, client.privSub)

	client.UnsubscribePrivate("c")
	assert.Equal(t, []interface{}{}, client.GetPrivateSubscriptions())
	assert.Equal(t, []interface{}{"a.x"}, client.pubSub)
	assert.Equal(t, []interface{}{}, client.privSub)

	client.UnsubscribePublic("a.x")
	assert.Equal(t, []interface{}{}, client.GetPublicSubscriptions())
	assert.Equal(t, []interface{}{}, client.pubSub)
	assert.Equal(t, []interface{}{}, client.privSub)
}
