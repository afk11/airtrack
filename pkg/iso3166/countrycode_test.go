package iso3166

import (
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestCountryCode(t *testing.T) {
	c := AlphaTwoCountryCode("PL")
	assert.Equal(t, "PL", c.String())
}
