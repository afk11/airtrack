package airtrackqa

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/fs"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/afk11/airtrack/pkg/geo/openaip"
	"github.com/afk11/airtrack/pkg/tracker"
	"github.com/pkg/errors"
)

// NearestAirport - takes lat and lon and tries to find the
// nearest airport to that location
type NearestAirport struct {
	// Lat - the latitude
	Lat float64 `help:"latitude"`
	// Lon - the longitude
	Lon float64 `help:"longitude"`
}

// Run takes the ctx, parses location files, and attempts to reverse
// geocode the location
func (na *NearestAirport) Run(ctx *Context) error {
	fmt.Printf("%f %f\n", na.Lat, na.Lon)

	cfg := ctx.Config
	if len(cfg.Airports.OpenAIPDirectories) == 0 {
		return errors.New("no airport directories configured")
	}

	nearestAirports := geo.NewNearestAirportGeocoder(tracker.DefaultGeoHashLength)
	files, err := fs.ScanDirectoriesForFiles("aip", cfg.Airports.OpenAIPDirectories)
	for _, file := range files {
		openaipFile, err := openaip.ParseFile(file)
		if err != nil {
			return errors.Wrapf(err, "error reading openaip file: %s", file)
		}
		acRecords, err := openaip.ExtractOpenAIPRecords(openaipFile)
		if err != nil {
			return errors.Wrapf(err, "converting openaip record: %s", file)
		}
		err = nearestAirports.Register(acRecords)
		if err != nil {
			return errors.Wrapf(err, "error reading openaip file: %s", file)
		}
		fmt.Printf("found %d openaipFile in file %s\n", len(acRecords), file)
	}

	place, distance := nearestAirports.ReverseGeocode(na.Lat, na.Lon)
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
