package coord

import (
	"fmt"
	"github.com/pkg/errors"
	"math"
	"strconv"
)

func DMSToDecimalLocation(lat, lon string) (float64, float64, error) {
	_, la, err := parseDMSLat(lat)
	if err != nil {
		return 0, 0, err
	}
	_, lo, err := parseDMSLon(lon)
	if err != nil {
		return 0, 0, err
	}
	return la, lo, nil
}
func DecimalToDMSLocation(lat, lon float64) (string, string, error) {
	// lat
	latDeg := int(math.Abs(lat))
	rem := math.Abs(lat) - float64(latDeg)
	latMin := int(rem * 60)
	latSec := (rem - float64(latMin)/60) * 3600
	var lac byte
	if lat > 0 {
		lac = 'N'
	} else {
		lac = 'S'
	}
	latSecStr := fmt.Sprintf("%02f", latSec)[2:5]
	la := fmt.Sprintf("%02d%02d.%s%c", latDeg, latMin, latSecStr, lac)
	// lon
	lonDeg := int(math.Abs(lon))
	rem = math.Abs(lon) - float64(lonDeg)
	lonMin := int(rem * 60)
	lonSec := (rem - float64(lonMin)/60) * 3600

	var loc byte
	if lon > 0 {
		loc = 'W'
	} else {
		loc = 'E'
	}

	lonSecStr := fmt.Sprintf("%02f", lonSec)[2:5]
	lo := fmt.Sprintf("%03d%02d.%s%c", lonDeg, lonMin, lonSecStr, loc)
	return la, lo, nil
}

// parseDMSLat parses DMS formatted latitude such as 01354.216E
func parseDMSLat(lat string) (byte, float64, error) {
	if len(lat) != 9 {
		return 0, 0, errors.New("latitude should be 9 characters, eg, 0108.382S")
	}
	suffix := lat[len(lat)-1]
	if suffix != 'N' && suffix != 'S' {
		return 0, 0, errors.New("invalid suffix for latitude")
	}
	degrees, err := strconv.ParseFloat(lat[0:2], 64)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "failed to parse latitude degrees")
	}
	minutes, err := strconv.ParseFloat(lat[2:4], 64)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "failed to parse latitude minutes")
	}
	seconds, err := strconv.ParseFloat("0."+lat[5:8], 64)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "failed to parse latitude seconds")
	}
	dec := degrees + minutes/60 + seconds/3600
	if suffix == 'S' {
		dec = -dec
	}
	return suffix, dec, nil
}

// parseDMSLon parse DMS formatted longitude such as 01354.216E
func parseDMSLon(lon string) (byte, float64, error) {
	if len(lon) != 10 {
		return 0, 0, errors.New("longitude should be 10 characters, eg, 0108.382S")
	}
	suffix := lon[len(lon)-1]
	if suffix != 'E' && suffix != 'W' {
		return 0, 0, errors.New("invalid suffix for longitude")
	}
	degrees, err := strconv.ParseFloat(lon[0:3], 64)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "failed to parse longitude degrees")
	}
	minutes, err := strconv.ParseFloat(lon[3:5], 64)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "failed to parse longitude minutes")
	}
	seconds, err := strconv.ParseFloat("0."+lon[6:9], 64)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "failed to parse longitude seconds")
	}
	dec := degrees + minutes/60 + seconds/3600
	if suffix == 'E' {
		dec = -dec
	}
	return suffix, dec, nil
}
