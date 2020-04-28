package routing

import (
	"testing"

	"github.com/openware/rango/pkg/message"
)

type MockedClient struct{}

func (c *MockedClient) Send(m []byte) {
}

func (c *MockedClient) Close() {
}

func (c *MockedClient) GetUID() string {
	return "U123456789"
}

func TestHandleSubscribe(t *testing.T) {
	h := NewHub()

	h.handleSubscribe(Request{
		client: &MockedClient{},
		Request: message.Request{
			Streams: []string{""},
		},
	})
}
