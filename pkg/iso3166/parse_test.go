package iso3166

import (
	"bytes"
	asset "github.com/afk11/airtrack/pkg/assets"
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestParseColumnFormat(t *testing.T) {
	data, err := asset.Asset("assets/iso3166_country_codes.txt")
	assert.NoError(t, err)
	assert.NotNil(t, data)
	buf := bytes.NewBuffer(data)
	store, err := ParseColumnFormat(buf)
	assert.NoError(t, err)
	assert.NotNil(t, store)

	bd, found := store.GetCountryCode(AlphaTwoCountryCode("BD"))
	assert.True(t, found)
	assert.Equal(t, "Bangladesh", bd.Name())

	ar, found := store.GetCountryCode(AlphaTwoCountryCode("AR"))
	assert.True(t, found)
	assert.Equal(t, "Argentina", ar.Name())

	sd, found := store.GetCountryCode(AlphaTwoCountryCode("SD"))
	assert.True(t, found)
	assert.Equal(t, "Sudan", sd.Name())

	c, found := store.GetCountryCode(AlphaTwoCountryCode("ZZ"))
	assert.False(t, found)
	assert.Nil(t, c)
}
