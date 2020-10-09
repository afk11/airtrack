package kml

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/pkg/errors"
	"time"
)

const (
	openDoc = `<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2" xmlns:gx="http://www.google.com/kml/ext/2.2">
<Document>`

	closeDoc = `
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

// locationPlacemark generates XML for a location placemark
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

// WriterOptions contains some preprocessed information
// about the flight
type WriterOptions struct {
	RouteName        string
	RouteDescription string

	SourceName        string
	SourceDescription string

	DestinationName        string
	DestinationDescription string
}

// Writer processes locations into a KML file
type Writer struct {
	opt   WriterOptions
	first *db.SightingLocation
	last  *db.SightingLocation
	when  string
	coord string
}

// NewWriter returns a new Writer initialized with opt
func NewWriter(opt WriterOptions) *Writer {
	return &Writer{
		opt: opt,
	}
}

// Write processes the new locationData and appends it to internal state
func (w *Writer) Write(locationData []db.SightingLocation) {
	for i := range locationData {
		w.when += "            <when>" + locationData[i].TimeStamp.Format(time.RFC3339) + "</when>\n"
		w.coord += fmt.Sprintf("            <gx:coord>%f %f %d</gx:coord>\n",
			locationData[i].Longitude, locationData[i].Latitude, locationData[i].Altitude)
	}

	if len(locationData) > 0 {
		if w.first == nil {
			w.first = &locationData[0]
		}
		w.last = &locationData[len(locationData)-1]
	}
}

// Final returns the final result of the writer, or an error if one occurred.
func (w *Writer) Final() (string, error) {
	if w.first == nil || w.last == nil {
		return "", errors.New("missing location information")
	}
	return openDoc +
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
		closeDoc, nil
}
