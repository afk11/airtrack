package airtrackqa

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/geo/openaip"
	"github.com/pkg/errors"
	"io/ioutil"
)

type OpenAipAirportStats struct {
	File string `help:"File path to openaip file"`
}

func (o *OpenAipAirportStats) Run(ctx *Context) error {
	fmt.Printf("%s\n", o.File)

	contents, err := ioutil.ReadFile(o.File)
	if err != nil {
		return errors.Wrapf(err, "reading openaip file")
	}
	f, err := openaip.Parse(contents)
	if err != nil {
		return errors.Wrapf(err, "parsing openaip file")
	}

	airportTypeToCount := make(map[string]int64)
	hasIcao := map[bool]int64{
		true:  0,
		false: 0,
	}
	for _, airport := range f.Waypoints.Airports {
		_, ok := airportTypeToCount[airport.Type]
		if !ok {
			airportTypeToCount[airport.Type] = 0
		}
		airportTypeToCount[airport.Type]++
		hasIcao[airport.Icao != ""]++
	}
	fmt.Println("TOTAL FOR EACH AIRPORT TYPE")
	for airportType, count := range airportTypeToCount {
		fmt.Printf("%s %d\n", airportType, count)
	}
	fmt.Println()

	fmt.Println("TOTAL WITH ICAO FIELD")
	fmt.Printf("%t %d\n", true, hasIcao[true])
	fmt.Printf("%t %d\n", false, hasIcao[false])
	fmt.Println()

	return nil
}
