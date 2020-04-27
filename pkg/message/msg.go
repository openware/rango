package message

import (
	"encoding/json"
	"fmt"
)

type Request struct {
	Method  string
	Streams []string
}

type NewResponse struct {
	Resp map[string]interface{}
}

func Response(err error, message interface{}) ([]byte, error) {
	// TODO: Response method
	res := make(map[string]interface{}, 3)
	if err != nil {
		fmt.Println(err)
		res["unsuccess"] = err.Error()
	} else {
		res["success"] = message
	}
	return json.Marshal(res)
}

func Event(channel string, data interface{}) ([]byte, error) {
	// TODO: Event method
	resp := make(map[string]interface{})
	resp[channel] = data
	return json.Marshal(resp)
}
