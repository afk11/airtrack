package readsb

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseExtendedSquitterTargetStateAndStatus(t *testing.T) {
	msgHex := "3319acc59750f4178d485345ea50285d1d3f8c4deb42"

	IcaoFilterInitOnce()
	ModeACInitOnce()
	ModesChecksumInitOnce(1)

	msgBytes, err := hex.DecodeString(msgHex)
	assert.NoError(t, err)
	decoder := NewDecoder()
	decoder.NumBitsToCorrect(1)

	mm, err := DecodeBinMessage(decoder, msgBytes, 0, true)
	assert.NoError(t, err)
	assert.NotNil(t, mm)
	modes := &ModesMessage{msg: mm}
	assert.Equal(t, "485345", modes.GetIcaoHex())
	isOnGround, err := modes.IsOnGround()
	assert.NoError(t, err)
	assert.False(t, isOnGround)
	assert.Equal(t, 17, modes.GetMessageType())
}
