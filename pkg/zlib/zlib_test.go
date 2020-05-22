package zlib

import (
	"bytes"
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestZlib(t *testing.T) {
	data := []byte("Hello world!")
	out, err := Encode(data)
	assert.NoError(t, err)

	in, err := Decode(out)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(in, data))
}
