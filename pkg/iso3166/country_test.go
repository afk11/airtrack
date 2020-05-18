package iso3166

import (
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestCountry_Name(t *testing.T) {
	c := Country{
		name: "Zimbabwe",
	}
	assert.Equal(t, "Zimbabwe", c.Name())
}
