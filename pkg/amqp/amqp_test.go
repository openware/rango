package amqp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_privateTopicOnly(t *testing.T) {
	assert.Equal(t, true, privateTopicOnly("rango.instance.private-312978312897367"))
	assert.Equal(t, true, privateTopicOnly("rango.instance.private-1"))
	assert.Equal(t, false, privateTopicOnly("rango.instance.private-"))
	assert.Equal(t, false, privateTopicOnly("rango.instance.312978312897367"))
}
