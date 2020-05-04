package msg

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
)

func ParseUint64(t interface{}) (uint64, error) {
	vf, ok := t.(float64)

	if !ok {
		return 0, errors.New("expected uint64 got: " + reflect.TypeOf(t).String())
	}

	vu := uint64(vf)
	if float64(vu) != vf {
		return 0, errors.New("expected unsigned integer got: float")
	}
	return vu, nil
}

func ParseUint8(t interface{}) (uint8, error) {
	vf, ok := t.(float64)

	if !ok {
		return 0, errors.New("expected uint8 got: " + reflect.TypeOf(t).String())
	}

	if math.Trunc(vf) != vf {
		return 0, errors.New("expected unsigned integer got: float")
	}
	return uint8(vf), nil
}

func ParseString(t interface{}) (string, error) {
	s, ok := t.(string)

	if !ok {
		return "", errors.New("expected string got: " + reflect.TypeOf(t).String())
	}
	return s, nil
}

func ParseSlice(t interface{}) ([]interface{}, error) {
	s, ok := t.([]interface{})

	if !ok {
		return nil, errors.New("expected array got: " + reflect.TypeOf(t).String())
	}
	return s, nil
}

func ParseSliceOfStrings(t interface{}) ([]string, error) {
	s, err := ParseSlice(t)
	if err != nil {
		return nil, err
	}

	a := make([]string, len(s))
	for i, istr := range s {
		str, ok := istr.(string)
		a[i] = str

		if !ok {
			return nil, errors.New("expected array of string, got unexpected " + reflect.TypeOf(istr).String())
		}
	}
	return a, nil
}

func Parse(msg []byte) (*Msg, error) {
	req := Msg{}

	var v []interface{}
	if err := json.Unmarshal(msg, &v); err != nil {
		return nil, fmt.Errorf("Could not parse message: %w", err)
	}

	if len(v) != 4 {
		return nil, errors.New("message must contains 4 elements")
	}

	t, err := ParseUint8(v[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse type: %w", err)
	}
	if t != Request && t != Response && t != Event {
		return nil, errors.New("message type must be 0, 1 or 2")
	}

	reqID, err := ParseUint64(v[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse request ID: %w", err)
	}

	method, err := ParseString(v[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse method: %w", err)
	}

	args, err := ParseSlice(v[3])
	if err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	req.Type = t
	req.ReqID = reqID
	req.Method = method
	req.Args = args

	return &req, nil
}
