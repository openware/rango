package message

import (
	"encoding/json"
)

type Request struct {
	Type   string
	ID     uint32
	Method string
	Params []interface{}
}

func Response(id uint32, err error, result interface{}) ([]byte, error) {
	// TODO: Response method
	res := make([]interface{}, 4)
	return json.Marshal(res)
}

func Event(method string, data interface{}) ([]byte, error) {
	// TODO: Event method
	res := make([]interface{}, 3)

	return json.Marshal(res)
}
