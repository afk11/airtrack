package cup

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/afk11/airtrack/pkg/coord"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
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

func ParseFile(file string) ([][]string, error) {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "reading openaip file")
	}
	f, err := Parse(bytes.NewBuffer(contents))
	if err != nil {
		return nil, errors.Wrapf(err, "parsing openaip file")
	}
	return f, nil
}
func Parse(contents io.Reader) ([][]string, error) {
	r := csv.NewReader(contents)
	r.Comment = '*'

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if strings.Join(records[0], ",") != "name,code,country,lat,lon,elev,style,rwdir,rwlen,freq,desc" {
		return nil, errors.New("missing first row with column titles, or invalid titles found.")
	}

	return records[1:], nil
}

func ExtractCupRecords(records [][]string) ([]geo.AirportRecord, error) {
	var airports []geo.AirportRecord
	// we unmarshal our byteArray which contains our
	// xmlFiles content into 'aip' which we defined above
	for _, airport := range records {
		acRecord, err := cup.FromCupCsvRecord(airport)
		if err != nil {
			return nil, err
		}
		airports = append(airports, *acRecord)
	}
	return airports, nil
}
