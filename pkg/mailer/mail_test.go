package mailer

import (
	"bytes"
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestZlib(t *testing.T) {
	data := []byte("Hello world!")
	out, err := zlibEncode(data)
	assert.NoError(t, err)

	in, err := zlibDecode(out)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(in, data))
}
