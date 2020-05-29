package openaip

import (
	"encoding/xml"
	"github.com/afk11/airtrack/pkg/coord"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/afk11/airtrack/pkg/geo/cup"
	"github.com/pkg/errors"
	"io/ioutil"
	"strconv"
)

type (
	Elevation struct {
		XMLName xml.Name `xml:"ELEV"`
		Unit    string   `xml:"UNIT,attr"`
		Value   float64  `xml:",chardata"`
	}
	Geolocation struct {
		XMLName   xml.Name  `xml:"GEOLOCATION"`
		Latitude  string    `xml:"LAT"`
		Longitude string    `xml:"LON"`
		Elevation Elevation `xml:"ELEV"`
	}
	Airport struct {
		XMLName     xml.Name    `xml:"AIRPORT"`
		Type        string      `xml:"TYPE,attr"`
		Identifier  string      `xml:"IDENTIFIER"`
		Country     string      `xml:"COUNTRY"`
		Name        string      `xml:"NAME"`
		Icao        string      `xml:"ICAO"`
		Geolocation Geolocation `xml:"GEOLOCATION"`
	}
	Waypoints struct {
		XMLName  xml.Name  `xml:"WAYPOINTS"`
		Airports []Airport `xml:"AIRPORT"`
	}
	File struct {
		XMLName    xml.Name  `xml:"OPENAIP"`
		Version    string    `xml:"VERSION,attr"`
		DataFormat string    `xml:"DATAFORMAT,attr"`
		Waypoints  Waypoints `xml:"WAYPOINTS"`
	}
)

func ParseFile(file string) (*File, error) {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "reading openaip file")
	}
	f, err := Parse(contents)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing openaip file")
	}
	return f, nil
}

func Parse(contents []byte) (*File, error) {
	// we initialize our Users array
	var aip File
	// we unmarshal our byteArray which contains our
	// xmlFiles content into 'aip' which we defined above
	err := xml.Unmarshal(contents, &aip)
	if err != nil {
		return nil, err
	}
	return &aip, nil
}

func convertAirportToAirportRecord(a *Airport) (geo.AirportRecord, error) {
	lat, err := strconv.ParseFloat(a.Geolocation.Latitude, 64)
	if err != nil {
		return geo.AirportRecord{}, err
	}
	lon, err := strconv.ParseFloat(a.Geolocation.Longitude, 64)
	if err != nil {
		return geo.AirportRecord{}, err
	}
	return geo.AirportRecord{
		Name:        a.Name,
		Code:        a.Icao,
		CountryCode: a.Country,
		Latitude:    lat,
		Longitude:   lon,
		Elevation:   a.Geolocation.Elevation.Value,
	}, nil
}

func convertCupRecordToAirportRecord(rec []string) (geo.AirportRecord, error) {
	lat, lon, err := coord.DMSToDecimalLocation(rec[3], rec[4])
	if err != nil {
		return geo.AirportRecord{}, err
	}
	if rec[5][len(rec[5])-1] != 'm' {
		return geo.AirportRecord{}, errors.New("expected elevation unit to be meters")
	}
	elev, err := strconv.ParseFloat(rec[5][0:len(rec[5])-2], 64)
	if err != nil {
		return geo.AirportRecord{}, err
	}
	return geo.AirportRecord{
		Name:        rec[0],
		Code:        rec[1],
		CountryCode: rec[2],
		Latitude:    lat,
		Longitude:   lon,
		Elevation:   elev,
	}, nil
}

func ExtractOpenAIPRecords(aip *File) ([]geo.AirportRecord, error) {
	var airports []geo.AirportRecord
	// we unmarshal our byteArray which contains our
	// xmlFiles content into 'aip' which we defined above
	for _, airport := range aip.Waypoints.Airports {
		acRecord, err := convertAirportToAirportRecord(&airport)
		if err != nil {
			return nil, err
		}
		airports = append(airports, acRecord)
	}
	return airports, nil
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
