package message

import (
	"errors"
	"fmt"
	"testing"
)

func TestMsg_Response(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		res, err := Response(32, nil, "ok")
		fmt.Println(string(res))

		if err != nil {
			t.Fatal("Should not return error")
		}

		if string(res) != `[1,32,null,"ok"]` {
			t.Fatal("Response invalid")
		}
	})

	t.Run("Some error", func(t *testing.T) {
		res, err := Response(32, errors.New("Some Error"), "ok")
		fmt.Println(string(res))

		if err != nil {
			t.Fatal("Should not return error")
		}

		if string(res) != `[1,32,"Some Error",null]` {
			t.Fatal("Response invalid")
		}
	})
}

func TestMsg_Event(t *testing.T) {
	res, err := Event("someMethod", "Hello")
	fmt.Println(string(res))

	if err != nil {
		t.Fatal("Should not return error")
	}

	if string(res) != `[2,"someMethod","Hello"]` {
		t.Fatal("Event invalid")
	}
}
