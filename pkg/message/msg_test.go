package message

import (
	"errors"
	"fmt"
	"testing"
)

func TestMsg_Response(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		res, err := PackOutgoingResponse(nil, "ok")
		fmt.Println(string(res))

		if err != nil {
			t.Fatal("Should not return error")
		}

		if string(res) != `{"success":"ok"}` {
			t.Fatal("Response invalid")
		}
	})

	t.Run("Some error", func(t *testing.T) {
		res, err := PackOutgoingResponse(errors.New("Some Error"), "ok")
		fmt.Println(string(res))

		if err != nil {
			t.Fatal("Should not return error")
		}

		if string(res) != `{"error":"Some Error"}` {
			t.Fatal("Response invalid")
		}
	})
}

func TestMsg_Event(t *testing.T) {
	res, err := PackOutgoingEvent("someMethod", "Hello")
	fmt.Println(string(res))

	if err != nil {
		t.Fatal("Should not return error")
	}

	if string(res) != `{"someMethod":"Hello"}` {
		t.Fatal("Event invalid")
	}
}
