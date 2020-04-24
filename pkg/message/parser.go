package message

import (
	"encoding/json"
	"fmt"
)

func ParseRequest(msg []byte) (Request, error) {
	request, err := Parse(msg)
	if err != nil {
		return request, err
	}

	return request, nil
}

func Parse(msg []byte) (Request, error) {
	var v map[string]interface{}
	var parsed Request

	if err := json.Unmarshal(msg, &v); err != nil {
		return parsed, fmt.Errorf("Could not parse message: %w", err)
	}

	return parsed, nil
}
