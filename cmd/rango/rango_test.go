package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRango_envToMatrix(t *testing.T) {
	env := []string{
		"PREFIXED_HELLO=world",
		"PREFIXED_ONE=TWO,three,four",
		"PREFIXED_FOO=bar",
	}

	matrix := envToMatrix(env, "PREFIXED_")

	assert.Len(t, matrix, 3)
	assert.Equal(t, "world", matrix["hello"][0])
	assert.Equal(t, []string{"TWO", "three", "four"}, matrix["one"])
	assert.Equal(t, "bar", matrix["foo"][0])
}
