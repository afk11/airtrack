package iso3166

import (
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestStore_GetCountryCode(t *testing.T) {
	codePT := AlphaTwoCountryCode("PT")
	codeSD := AlphaTwoCountryCode("SD")
	country := Country{
		name: "Portugal",
	}
	store := emptyStore()
	store.alpha2[codePT] = &country
	_, found := store.GetCountryCode(codeSD)
	assert.False(t, found)
	foundCountry, found := store.GetCountryCode(codePT)
	assert.True(t, found)
	assert.NotNil(t, foundCountry)
	assert.Equal(t, country.name, foundCountry.Name())
}
func TestNew_Errors(t *testing.T) {
	t.Run("invalid country code", func(t *testing.T) {
		rows := [][3]string{
			{"GBallalala", "GBR", "United Kingdom"},
		}
		s, err := New(rows)
		assert.EqualError(t, err, "alpha2 country code should be two characters")
		assert.Nil(t, s)
	})
	t.Run("duplicate country code", func(t *testing.T) {
		rows := [][3]string{
			{"GB", "GBR", "United Kingdom"},
			{"GB", "GBR", "United Kingdom"},
		}
		s, err := New(rows)
		assert.EqualError(t, err, "cannot use duplicate alpha2 country codes (GB)")
		assert.Nil(t, s)
	})
}
func TestNew(t *testing.T) {
	rows := [][3]string{
		{"GB", "GBR", "United Kingdom"},
		{"KZ", "KAZ", "Kazakhstan"},
		{"KI", "KIR", "Kiribati"},
	}
	store, err := New(rows)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(store.alpha2))

	c, ok := store.alpha2["GB"]
	assert.True(t, ok)
	assert.Equal(t, "United Kingdom", c.name)

	c, ok = store.alpha2["KZ"]
	assert.True(t, ok)
	assert.Equal(t, "Kazakhstan", c.name)

	c, ok = store.alpha2["KI"]
	assert.True(t, ok)
	assert.Equal(t, "Kiribati", c.name)
}
