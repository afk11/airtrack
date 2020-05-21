package airtrackqa

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/fs"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/afk11/airtrack/pkg/geo/openaip"
	"github.com/afk11/airtrack/pkg/tracker"
	"github.com/pkg/errors"
)

type NearestAirport struct {
	Lat float64 `help:"latitude"`
	Lon float64 `help:"longitude"`
}

func (na *NearestAirport) Run(ctx *Context) error {
	fmt.Printf("%f %f\n", na.Lat, na.Lon)

	cfg := ctx.Config
	if len(cfg.Airports.Directories) == 0 {
		return errors.New("no airport directories configured")
	}

	nearestAirports := geo.NewNearestAirportGeocoder(tracker.DefaultGeoHashLength)
	files, err := fs.ScanDirectoriesForFiles("aip", cfg.Airports.Directories)
	for _, file := range files {
		airports, err := openaip.ReadAirportsFromFile(file)
		if err != nil {
			return errors.Wrapf(err, "error reading openaip file: %s", file)
		}
		err = nearestAirports.Register(airports)
		if err != nil {
			return errors.Wrapf(err, "error reading openaip file: %s", file)
		}
		fmt.Printf("found %d airports in file %s\n", len(airports), file)
	}

	place, distance, err := nearestAirports.ReverseGeocode(na.Lat, na.Lon)
	if err != nil {
		return err
	} else if place == "" {
		fmt.Printf("place not found")
	} else {
		fmt.Printf("place: %s\n", place)
		fmt.Printf("distance: %f\n", distance)
	}
	return nil
}
