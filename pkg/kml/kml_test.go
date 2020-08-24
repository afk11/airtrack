package kml

import (
	"github.com/afk11/airtrack/pkg/db"
	assert "github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	ExpectedKml = `<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2" xmlns:gx="http://www.google.com/kml/ext/2.2">
<Document>
    <Placemark>
        <name>src</name>
        <description>src-desc</description>
        <Point>
            <coordinates>-0.039302,51.496711,100</coordinates>
        </Point>
    </Placemark>
    <Placemark>
        <name>dest</name>
        <description>dest-desc</description>
        <Point>
            <coordinates>-0.039302,51.496711,100</coordinates>
        </Point>
    </Placemark>
    <Placemark>
        <name>route</name>
        <description>route-desc</description>
        <gx:Track>
            <extrude>1</extrude>
            <tessellate>1</tessellate>
            <altitudeMode>absolute</altitudeMode>
            <when>2020-05-22T20:12:49Z</when>
            <when>2020-05-22T20:13:09Z</when>
            <gx:coord>-0.039302 51.496711 100</gx:coord>
            <gx:coord>-0.039302 51.496711 100</gx:coord>
        </gx:Track>
    </Placemark>
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

func TestWriterBasic(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	assert.NoError(t, err)

	First := db.SightingLocation{
		Latitude:  51.4967107,
		Longitude: -0.0393017,
		Altitude:  100,
		TimeStamp: time.Date(2020, 05, 22, 20, 12, 49, 0, loc),
	}
	Second := db.SightingLocation{
		Latitude:  51.4967107,
		Longitude: -0.0393015,
		Altitude:  100,
		TimeStamp: time.Date(2020, 05, 22, 20, 13, 9, 0, loc),
	}

	t.Run("single write", func(t *testing.T) {
		w := NewWriter(WriterOptions{
			RouteName:              "route",
			RouteDescription:       "route-desc",
			SourceName:             "src",
			SourceDescription:      "src-desc",
			DestinationName:        "dest",
			DestinationDescription: "dest-desc",
		})
		w.Write([]db.SightingLocation{First, Second})
		assert.NotNil(t, w.first)
		assert.Equal(t, &First, w.first)
		assert.NotNil(t, w.last)
		assert.Equal(t, &Second, w.last)
		k, err := w.Final()
		assert.NoError(t, err)
		assert.Equal(t, ExpectedKml, k)
	})
	t.Run("multiple writes", func(t *testing.T) {
		w := NewWriter(WriterOptions{
			RouteName:              "route",
			RouteDescription:       "route-desc",
			SourceName:             "src",
			SourceDescription:      "src-desc",
			DestinationName:        "dest",
			DestinationDescription: "dest-desc",
		})
		w.Write([]db.SightingLocation{First})
		w.Write([]db.SightingLocation{Second})
		assert.NotNil(t, w.first)
		assert.Equal(t, &First, w.first)
		assert.NotNil(t, w.last)
		assert.Equal(t, &Second, w.last)
		k, err := w.Final()
		assert.NoError(t, err)
		assert.Equal(t, ExpectedKml, k)
	})
}
