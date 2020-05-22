package kml

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/pkg/errors"
	"time"
)

const (
	OpenDoc = `<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2" xmlns:gx="http://www.google.com/kml/ext/2.2">
<Document>`

	CloseDoc = `
    <ScreenOverlay>
        <name>FlightAware</name>
        <Icon>
            <href>http://flightaware.com/images/logo_ge.png</href>
        </Icon>
        <overlayXY x="1" y="-1" xunits="fraction" yunits="fraction"/>
        <screenXY x="1" y="0" xunits="fraction" yunits="fraction"/>
        <size x="0" y="0" xunits="fraction" yunits="fraction"/>
    </ScreenOverlay>
</Document>
</kml>`
)

func locationPlacemark(name, desc string, altitude int64, latitude, longitude float64) string {
	// coordinates line: long, lat, alt
	return fmt.Sprintf(`
    <Placemark>
        <name>%s</name>
        <description>%s</description>
        <Point>
            <coordinates>%f,%f,%d</coordinates>
        </Point>
    </Placemark>`, name, desc, longitude, latitude, altitude)
}

func buildPlacemarkFragment(locationData []db.SightingLocation) (string, string) {
	when := ""
	coord := ""
	for _, row := range locationData {
		stamp := row.TimeStamp
		when = when + "            <when>" + stamp.Format(time.RFC3339) + "</when>\n"
		coord = coord + fmt.Sprintf("            <gx:coord>%f %f %d</gx:coord>\n", row.Longitude, row.Latitude, row.Altitude)
	}
	return when, coord
}

type WriterOptions struct {
	RouteName        string
	RouteDescription string

	SourceName        string
	SourceDescription string

	DestinationName        string
	DestinationDescription string
}
type Writer struct {
	opt   WriterOptions
	first *db.SightingLocation
	last  *db.SightingLocation
	when  string
	coord string
}

func NewWriter(opt WriterOptions) *Writer {
	return &Writer{
		opt: opt,
	}
}
func (w *Writer) Write(locationData []db.SightingLocation) {
	when, coord := buildPlacemarkFragment(locationData)
	w.when += when
	w.coord += coord
	if len(locationData) > 0 {
		if w.first == nil {
			w.first = &locationData[0]
		}
		w.last = &locationData[len(locationData)-1]
	}
}
func (w *Writer) Final() (string, error) {
	if w.first == nil || w.last == nil {
		return "", errors.New("missing location information")
	}
	return OpenDoc +
		locationPlacemark(w.opt.SourceName, w.opt.SourceDescription, w.first.Altitude, w.first.Latitude, w.first.Longitude) +
		locationPlacemark(w.opt.DestinationName, w.opt.DestinationDescription, w.last.Altitude, w.last.Latitude, w.last.Longitude) +
		`
    <Placemark>
        <name>` + w.opt.RouteName + `</name>
        <description>` + w.opt.RouteDescription + `</description>
        <gx:Track>
            <extrude>1</extrude>
            <tessellate>1</tessellate>
            <altitudeMode>absolute</altitudeMode>` + "\n" +
		w.when +
		w.coord + `        </gx:Track>
    </Placemark>` +
		CloseDoc, nil
}
