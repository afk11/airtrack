package cup

import (
	"fmt"
	assert "github.com/stretchr/testify/require"
	"strings"
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

func TestParse(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		sample := `
***
* This would appear to be a comment
***

name,code,country,lat,lon,elev,style,rwdir,rwlen,freq,desc
"Alpha","4242",AR,4554.240S,06733.540W,50.0m,5,70,790m,"",""
"Beta","6969",AR,5136.540S,07213.320W,270.0m,5,50,1200m,"aaa","zzz"`
		records, err := Parse(strings.NewReader(sample))
		assert.NoError(t, err)
		assert.Equal(t, 2, len(records))
		assert.Equal(t, "Alpha", records[0][0])
		assert.Equal(t, "4242", records[0][1])
		assert.Equal(t, "AR", records[0][2])
		assert.Equal(t, "50.0m", records[0][5])
		assert.Equal(t, "5", records[0][6])
		assert.Equal(t, "70", records[0][7])
		assert.Equal(t, "790m", records[0][8])
		assert.Equal(t, "", records[0][9])
		assert.Equal(t, "", records[0][10])

		assert.Equal(t, "Beta", records[1][0])
		assert.Equal(t, "6969", records[1][1])
		assert.Equal(t, "AR", records[1][2])
		assert.Equal(t, "270.0m", records[1][5])
		assert.Equal(t, "5", records[1][6])
		assert.Equal(t, "50", records[1][7])
		assert.Equal(t, "1200m", records[1][8])
		assert.Equal(t, "aaa", records[1][9])
		assert.Equal(t, "zzz", records[1][10])
	})
	t.Run("no comment", func(t *testing.T) {
		sample := `
name,code,country,lat,lon,elev,style,rwdir,rwlen,freq,desc
"Alpha","4242",AR,4554.240S,06733.540W,50.0m,5,70,790m,"",""
"Beta","6969",AR,5136.540S,07213.320W,270.0m,5,50,1200m,"aaa","zzz"`
		records, err := Parse(strings.NewReader(sample))
		assert.NoError(t, err)
		assert.Equal(t, 2, len(records))
		assert.Equal(t, "Alpha", records[0][0])
		assert.Equal(t, "4242", records[0][1])
		assert.Equal(t, "AR", records[0][2])
		assert.Equal(t, "50.0m", records[0][5])
		assert.Equal(t, "5", records[0][6])
		assert.Equal(t, "70", records[0][7])
		assert.Equal(t, "790m", records[0][8])
		assert.Equal(t, "", records[0][9])
		assert.Equal(t, "", records[0][10])

		assert.Equal(t, "Beta", records[1][0])
		assert.Equal(t, "6969", records[1][1])
		assert.Equal(t, "AR", records[1][2])
		assert.Equal(t, "270.0m", records[1][5])
		assert.Equal(t, "5", records[1][6])
		assert.Equal(t, "50", records[1][7])
		assert.Equal(t, "1200m", records[1][8])
		assert.Equal(t, "aaa", records[1][9])
		assert.Equal(t, "zzz", records[1][10])
	})
}
