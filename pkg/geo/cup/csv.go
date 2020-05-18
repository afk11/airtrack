package cup

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/pkg/errors"
	"strconv"
)

func parseCupCsvLocationParameter(param string, posSuffix byte, negSuffix byte) (byte, float64, error) {
	suffix := param[len(param)-1]
	if suffix != posSuffix && suffix != negSuffix {
		return 0, 0.0, fmt.Errorf("invalid suffix")
	}

	value, err := strconv.ParseFloat(param[:len(param)-1], 64)
	if err != nil {
		return 0, 0.0, errors.Wrap(err, "failed to parse location paramer")
	}
	// convert from milli-degrees to degrees
	value = value / 100
	// negative if necessary
	if suffix == negSuffix {
		value = -value
	}
	return suffix, value, nil
}

func FromCupCsvRecord(c []string) (*geo.AirportRecord, error) {
	if len(c) != 11 {
		fmt.Println(c)
		return nil, errors.Errorf("expected 7 fields in CUP format record (found %d)", len(c))
	}

	_, lat, err := parseCupCsvLocationParameter(c[3], 'N', 'S')
	if err != nil {
		return nil, errors.Wrap(err, "invalid latitude")
	}

	_, lon, err := parseCupCsvLocationParameter(c[4], 'E', 'W')
	if err != nil {
		return nil, errors.Wrap(err, "invalid longitude")
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
