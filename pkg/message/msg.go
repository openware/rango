package message

import (
	"encoding/json"
)

type Request struct {
	Method  string
	Streams []string
}

func PackOutgoingResponse(err error, message interface{}) ([]byte, error) {
	res := make(map[string]interface{}, 1)
	if err != nil {
		res["error"] = err.Error()
	} else {
		res["success"] = message
	}
	return json.Marshal(res)
}

func PackOutgoingEvent(channel string, data interface{}) ([]byte, error) {
	resp := make(map[string]interface{}, 1)
	resp[channel] = data
	return json.Marshal(resp)
}
