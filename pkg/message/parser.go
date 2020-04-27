package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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

	switch v["event"] {
	case "subscribe":
		parsed.Method = "subscribe"
		switch reflect.TypeOf(v["streams"]).Kind() {
		case reflect.Slice:
			streams := reflect.ValueOf(v["streams"])
			for i := 0; i < streams.Len(); i++ {
				parsed.Streams = append(parsed.Streams, streams.Index(i).Interface().(string))
			}
		}
	case "unsubscribe":
		parsed.Method = "unsubscribe"
		switch reflect.TypeOf(v["streams"]).Kind() {
		case reflect.Slice:
			streams := reflect.ValueOf(v["streams"])
			for i := 0; i < streams.Len(); i++ {
				parsed.Streams = append(parsed.Streams, streams.Index(i).Interface().(string))
			}
		}
	default:
		return parsed, errors.New("Could not parse Type: Invalid event")
	}

	return parsed, nil
}
