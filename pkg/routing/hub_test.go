package routing

import (
	"testing"

	"github.com/openware/rango/pkg/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockedClient struct {
	mock.Mock
}

func (c *MockedClient) Send(m string) {
	c.Called(m)
}

func (c *MockedClient) Close() {
}

func (c *MockedClient) GetUID() string {
	args := c.Called()
	return args.String(0)
}

func (c *MockedClient) GetSubscriptions() []string {
	args := c.Called()
	return args.Get(0).([]string)
}

func (c *MockedClient) SubscribePublic(s string) {
	c.Called(s)
}

func (c *MockedClient) SubscribePrivate(s string) {
	c.Called(s)
}

func (c *MockedClient) UnsubscribePublic(s string) {
	c.Called(s)
}

func (c *MockedClient) UnsubscribePrivate(s string) {
	c.Called(s)
}

func setup(c *MockedClient, streams []string) *Hub {
	h := NewHub()
	h.handleSubscribe(Request{
		client: c,
		Request: message.Request{
			Streams: streams,
		},
	})
	return h
}

func teardown(h *Hub, c *MockedClient, streams []string) {
	h.handleUnsubscribe(Request{
		client: c,
		Request: message.Request{
			Streams: streams,
		},
	})
}

func TestAnonymous(t *testing.T) {

	t.Run("subscribe to a public single stream", func(t *testing.T) {
		c := &MockedClient{}

		streams := []string{
			"eurusd.trades",
		}

		c.On("GetUID").Return("")
		c.On("GetSubscriptions").Return(streams).Once()
		c.On("SubscribePublic", streams[0]).Return().Once()
		c.On("Send", `{"success":{"message":"subscribed","streams":["`+streams[0]+`"]}}`).Return()

		h := setup(c, streams)
		assert.Equal(t, 1, len(h.PublicTopics))
		assert.Equal(t, 0, len(h.PrivateTopics))

		c.On("UnsubscribePublic", streams[0]).Return()
		c.On("GetSubscriptions").Return([]string{}).Once()
		c.On("Send", `{"success":{"message":"unsubscribed","streams":[]}}`).Return()

		teardown(h, c, streams)
		assert.Equal(t, 0, len(h.PublicTopics))
		assert.Equal(t, 0, len(h.PrivateTopics))
	})

	t.Run("subscribe to multiple public streams", func(t *testing.T) {
		c := &MockedClient{}
		streams := []string{
			"eurusd.trades",
			"eurusd.updates",
		}

		c.On("GetUID").Return("")
		c.On("GetSubscriptions").Return(streams).Once()
		c.On("SubscribePublic", "eurusd.trades").Return()
		c.On("SubscribePublic", "eurusd.updates").Return()
		c.On("Send", `{"success":{"message":"subscribed","streams":["eurusd.trades","eurusd.updates"]}}`).Return()

		h := setup(c, []string{
			"eurusd.trades",
			"eurusd.updates",
		})

		assert.Equal(t, 2, len(h.PublicTopics))
		assert.Equal(t, 0, len(h.PrivateTopics))

		c.On("UnsubscribePublic", streams[0]).Return().Once()
		c.On("UnsubscribePublic", streams[1]).Return().Once()
		c.On("GetSubscriptions").Return([]string{}).Once()
		c.On("Send", `{"success":{"message":"unsubscribed","streams":[]}}`).Return()

		teardown(h, c, streams)
		assert.Equal(t, 0, len(h.PublicTopics))
		assert.Equal(t, 0, len(h.PrivateTopics))

	})

	t.Run("subscribe to a private single stream", func(t *testing.T) {
		c := MockedClient{}

		c.On("GetUID").Return("")
		c.On("GetSubscriptions").Return([]string{})
		c.On("SubscribePrivate", "trades").Return()
		c.On("Send", `{"success":{"message":"subscribed","streams":[]}}`).Return()

		h := setup(&c, []string{
			"trades",
		})

		assert.Equal(t, 0, len(h.PublicTopics))
		assert.Equal(t, 0, len(h.PrivateTopics))
	})
}
func TestAuthenticated(t *testing.T) {
	t.Run("subscribe to a private single stream", func(t *testing.T) {
		c := &MockedClient{}

		c.On("GetUID").Return("UIDABC00001")
		c.On("GetSubscriptions").Return([]string{"trades"}).Once()
		c.On("SubscribePrivate", "trades").Return()
		c.On("Send", `{"success":{"message":"subscribed","streams":["trades"]}}`).Return()

		h := setup(c, []string{
			"trades",
		})
		assert.Equal(t, 0, len(h.PublicTopics))
		assert.Equal(t, 1, len(h.PrivateTopics))

		c.On("UnsubscribePrivate", "trades").Return().Once()
		c.On("GetSubscriptions").Return([]string{}).Once()
		c.On("Send", `{"success":{"message":"unsubscribed","streams":[]}}`).Return()

		teardown(h, c, []string{"trades"})
		assert.Equal(t, 0, len(h.PublicTopics))
		assert.Equal(t, 0, len(h.PrivateTopics))
	})

	t.Run("subscribe to multiple private streams", func(t *testing.T) {
		c := &MockedClient{}

		c.On("GetSubscriptions").Return([]string{"trades", "orders"}).Once()
		c.On("GetUID").Return("UIDABC00001")
		c.On("SubscribePrivate", "trades").Return()
		c.On("SubscribePrivate", "orders").Return()
		c.On("Send", `{"success":{"message":"subscribed","streams":["trades","orders"]}}`).Return()

		h := setup(c, []string{"trades", "orders"})
		assert.Equal(t, 0, len(h.PublicTopics))
		assert.Equal(t, 1, len(h.PrivateTopics))

		uTopics, ok := h.PrivateTopics["UIDABC00001"]
		require.True(t, ok)
		assert.Equal(t, 2, len(uTopics))

		c.On("UnsubscribePrivate", "trades").Return().Once()
		c.On("UnsubscribePrivate", "orders").Return().Once()
		c.On("GetSubscriptions").Return([]string{}).Once()
		c.On("Send", `{"success":{"message":"unsubscribed","streams":[]}}`).Return()

		teardown(h, c, []string{"trades", "orders"})
		assert.Equal(t, 0, len(h.PublicTopics))
		assert.Equal(t, 0, len(h.PrivateTopics))

	})

	t.Run("subscribe to multiple private and public streams", func(t *testing.T) {
		c := &MockedClient{}

		c.On("GetSubscriptions").Return([]string{"trades", "orders", "eurusd.updates"}).Once()
		c.On("GetUID").Return("UIDABC00001")
		c.On("SubscribePrivate", "trades").Return()
		c.On("SubscribePrivate", "orders").Return()
		c.On("SubscribePublic", "eurusd.updates").Return()
		c.On("Send", `{"success":{"message":"subscribed","streams":["trades","orders","eurusd.updates"]}}`).Return()

		h := setup(c, []string{"trades", "orders", "eurusd.updates"})
		assert.Equal(t, 1, len(h.PublicTopics))
		assert.Equal(t, 1, len(h.PrivateTopics))

		uTopics, ok := h.PrivateTopics["UIDABC00001"]
		require.True(t, ok)
		assert.Equal(t, 2, len(uTopics))

		c.On("UnsubscribePrivate", "trades").Return().Once()
		c.On("UnsubscribePrivate", "orders").Return().Once()
		c.On("UnsubscribePublic", "eurusd.updates").Return().Once()
		c.On("GetSubscriptions").Return([]string{}).Once()
		c.On("Send", `{"success":{"message":"unsubscribed","streams":[]}}`).Return()

		teardown(h, c, []string{"trades", "orders", "eurusd.updates"})
		assert.Equal(t, 0, len(h.PublicTopics))
		assert.Equal(t, 0, len(h.PrivateTopics))
	})
}
