package coord

import (
	"fmt"
	assert "github.com/stretchr/testify/require"
	"testing"
)

type coordTestFixture struct {
	dmsLat string
	dmsLon string
}

func TestBackToBack(t *testing.T) {
	fixtures := []coordTestFixture{
		{
			dmsLat: "5325.278N",
			dmsLon: "00616.206W",
		},
		{
			dmsLat: "0204.536N",
			dmsLon: "01129.592E",
		},
		{
			dmsLat: "4554.240S",
			dmsLon: "06733.540W",
		},
	}

	for i, fixture := range fixtures {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			decLat, decLon, err := DMSToDecimalLocation(fixture.dmsLat, fixture.dmsLon)
			assert.NoError(t, err)
			la, lo, err := DecimalToDMSLocation(decLat, decLon)
			assert.NoError(t, err)
			assert.Equal(t, fixture.dmsLat, la)
			assert.Equal(t, fixture.dmsLon, lo)
		})
	}
}
