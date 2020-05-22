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
		openaipFile, err := openaip.ParseFile(file)
		if err != nil {
			return errors.Wrapf(err, "error reading openaip file: %s", file)
		}
		acRecords, err := openaip.ExtractAirports(openaipFile)
		if err != nil {
			return errors.Wrapf(err, "converting openaip record: %s", file)
		}
		err = nearestAirports.Register(acRecords)
		if err != nil {
			return errors.Wrapf(err, "error reading openaip file: %s", file)
		}
		fmt.Printf("found %d openaipFile in file %s\n", len(acRecords), file)
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
