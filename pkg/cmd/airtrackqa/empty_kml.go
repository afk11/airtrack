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
			Altitude:  100,
			Latitude:  100,
			Longitude: 100,
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
	w.Add(history)
	_, _, data := w.Final()

	fmt.Printf("%s\n", data)
	return nil
}
