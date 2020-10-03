package openaip

import (
	"encoding/xml"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/pkg/errors"
	"io/ioutil"
	"strconv"
)

type (
	// Elevation - containing elevation value and units
	Elevation struct {
		// XMLName - entity name
		XMLName xml.Name `xml:"ELEV"`
		// Unit - unit as string
		Unit string `xml:"UNIT,attr"`
		// Value: altitude (todo: MASL? what type?)
		Value float64 `xml:",chardata"`
	}
	// Geolocation - containing position information for the airport
	Geolocation struct {
		// XMLName - entity name
		XMLName   xml.Name  `xml:"GEOLOCATION"`
		Latitude  string    `xml:"LAT"`
		Longitude string    `xml:"LON"`
		Elevation Elevation `xml:"ELEV"`
	}
	// Airport - main structure containing aircraft information
	Airport struct {
		// XMLName - entity name
		XMLName     xml.Name    `xml:"AIRPORT"`
		Type        string      `xml:"TYPE,attr"`
		Identifier  string      `xml:"IDENTIFIER"`
		Country     string      `xml:"COUNTRY"`
		Name        string      `xml:"NAME"`
		Icao        string      `xml:"ICAO"`
		Geolocation Geolocation `xml:"GEOLOCATION"`
	}
	// Waypoints - contains a list of Airports.
	Waypoints struct {
		// XMLName - entity name
		XMLName  xml.Name  `xml:"WAYPOINTS"`
		Airports []Airport `xml:"AIRPORT"`
	}
	// File - main structure of openaip file
	File struct {
		// XMLName - entity name
		XMLName    xml.Name  `xml:"OPENAIP"`
		Version    string    `xml:"VERSION,attr"`
		DataFormat string    `xml:"DATAFORMAT,attr"`
		Waypoints  Waypoints `xml:"WAYPOINTS"`
	}
)

// ParseFile takes an openaip filepath and returns the decoded File if
// if successful. If not, an error is returned.
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

// Parse decodes contents and returns the decoded File if successful.
// If not, an error is returned.
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

// convertAirportToAirportRecord converts from the XML structure to
// an AirportRecord for use with the NearestAirportGeocoder
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

// ExtractOpenAIPRecords takes aip and converts all the airports
// into geo.AirportRecords. The list is returned if successful, and
// if not an error is returned.
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
