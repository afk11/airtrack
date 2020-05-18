package iso3166

import (
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestStore_GetCountryCode(t *testing.T) {
	codePT := CountryCode("PT")
	codeSD := CountryCode("SD")
	country := Country{
		name: "Portugal",
	}
	store := emptyStore()
	store.countries[codePT] = country
	_, found := store.GetCountryCode(codeSD)
	assert.False(t, found)
	foundCountry, found := store.GetCountryCode(codePT)
	assert.True(t, found)
	assert.NotNil(t, foundCountry)
	assert.Equal(t, country.name, foundCountry.Name())
}
func TestNew_Errors(t *testing.T) {
	t.Run("invalid country code", func(t *testing.T) {
		rows := [][2]string{
			{"GBallalala", "United Kingdom"},
		}
		s, err := New(rows)
		assert.EqualError(t, err, "country code should be two characters")
		assert.Nil(t, s)
	})
	t.Run("duplicate country code", func(t *testing.T) {
		rows := [][2]string{
			{"GB", "United Kingdom"},
			{"GB", "United Kingdom"},
		}
		s, err := New(rows)
		assert.EqualError(t, err, "cannot use duplicate country codes (GB)")
		assert.Nil(t, s)
	})
}
func TestNew(t *testing.T) {
	rows := [][2]string{
		{"GB", "United Kingdom"},
		{"KZ", "Kazakhstan"},
		{"KI", "Kiribati"},
	}
	store, err := New(rows)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(store.countries))

	c, ok := store.countries["GB"]
	assert.True(t, ok)
	assert.Equal(t, "United Kingdom", c.name)

	c, ok = store.countries["KZ"]
	assert.True(t, ok)
	assert.Equal(t, "Kazakhstan", c.name)

	c, ok = store.countries["KI"]
	assert.True(t, ok)
	assert.Equal(t, "Kiribati", c.name)
}
