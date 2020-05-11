package msg

import (
	"encoding/json"
	"errors"
)

const (
	// Request type code
	Request = 1

	// Response type code
	Response = 2

	// EventPublic is public event type code
	EventPublic = 3

	// EventPrivate is private event type code
	EventPrivate = 4
)

// Msg represent websocket messages, it could be either a request, a response or an event
type Msg struct {
	Type   uint8
	ReqID  uint64
	Method string
	Args   []interface{}
}

// NewResponse build a response object
func NewResponse(req *Msg, method string, args []interface{}) *Msg {
	return &Msg{
		Type:   Response,
		ReqID:  req.ReqID,
		Method: method,
		Args:   args,
	}
}

// Encode msg into json
func (m *Msg) Encode() ([]byte, error) {
	switch m.Type {
	case Response, Request:
		return json.Marshal([]interface{}{
			m.Type,
			m.ReqID,
			m.Method,
			m.Args,
		})

	case EventPrivate, EventPublic:
		return json.Marshal([]interface{}{
			m.Type,
			m.Method,
			m.Args,
		})

	default:
		return nil, errors.New("invalid type")
	}
}

// Convss2is converts a string slice to interface slice more details: https://golang.org/doc/faq#convert_slice_of_interface)
func Convss2is(a []string) []interface{} {
	s := make([]interface{}, len(a))
	for i, v := range a {
		s[i] = v
	}
	return s
}

func Contains(haystack []interface{}, niddle interface{}) bool {
	for _, el := range haystack {
		if el == niddle {
			return true
		}
	}
	return false
}
