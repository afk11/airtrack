package cup

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/coord"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/pkg/errors"
	"strconv"
)

func FromCupCsvRecord(c []string) (*geo.AirportRecord, error) {
	if len(c) != 11 {
		fmt.Println(c)
		return nil, errors.Errorf("expected 7 fields in CUP format record (found %d)", len(c))
	}

	lat, lon, err := coord.DMSToDecimalLocation(c[3], c[4])
	if err != nil {
		return nil, errors.Wrap(err, "invalid location")
	}

	elevLen := len(c[5])
	elevSuffix := c[5][elevLen-1]
	if elevSuffix != 'm' {
		return nil, errors.New("unexpected suffix for elevation")
	}
	elevation, err := strconv.ParseFloat(c[5][:elevLen-1], 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid elevation")
	}

	r := new(geo.AirportRecord)
	r.Name = c[0]
	r.Code = c[1]
	r.CountryCode = c[2]
	r.Latitude = lat
	r.Longitude = lon
	r.Elevation = elevation
	r.Style = c[6]
	return r, nil
}
