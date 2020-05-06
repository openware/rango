package msg

import (
	"encoding/json"
)

const (
	// Request type code
	Request = 1

	// Response type code
	Response = 2

	// Event type code
	Event = 3
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
func (m *Msg) Encode() []byte {
	s, err := json.Marshal([]interface{}{
		m.Type,
		m.ReqID,
		m.Method,
		m.Args,
	})
	if err != nil {
		return []byte{}
	}
	return s
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
