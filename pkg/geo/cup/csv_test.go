package cup

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFromCupCsvRecord(t *testing.T) {
	c := []string{
		"DUBLIN", "EIDW", "IE", "5325.278N", "00616.206W", "74.0m", "5", "95", "2637m", "118.600", "",
	}
	record, err := FromCupCsvRecord(c)
	assert.NoError(t, err)
	assert.Equal(t, c[0], record.Name)
	assert.Equal(t, c[1], record.Code)
	assert.Equal(t, c[2], record.CountryCode)
	assert.Equal(t, "53.416744", fmt.Sprintf("%f", record.Latitude)[0:9])
	assert.Equal(t, "6.266724", fmt.Sprintf("%f", record.Longitude)[0:8])
	assert.Equal(t, 74.0, record.Elevation)
}
