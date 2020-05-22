package airtrackqa

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/kml"
)

type EmptyKml struct {
}

func (e *EmptyKml) Run(ctx *Context) error {
	history := []db.SightingLocation{
		{
			Latitude:  51.4967107,
			Longitude: -0.0393017,
			Altitude:  100,
		},
		{
			Latitude:  51.4967107,
			Longitude: -0.0393017,
			Altitude:  100,
		},
	}
	w := kml.NewWriter(kml.WriterOptions{
		RouteName:        "Route",
		RouteDescription: "Route description..",

		SourceName:        "Source",
		SourceDescription: "Source description..",

		DestinationName:        "Destination",
		DestinationDescription: "Destination description..",
	})
	w.Write(history)
	data, err := w.Final()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", data)
	return nil
}
