package msg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoding(t *testing.T) {
	msg := Msg{
		Type:   Request,
		ReqID:  42,
		Method: "test",
		Args:   []interface{}{"hello", "there"},
	}

	assert.Equal(t, `[0,42,"test",["hello","there"]]`, string(msg.Encode()))
}
