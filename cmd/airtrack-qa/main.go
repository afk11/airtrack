package main

import (
	"github.com/afk11/airtrack/pkg/cmd/airtrackqa"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/alecthomas/kong"
)

var cli struct {
	Config                      string                                 `help:"Configuration file path"`
	Email                       airtrackqa.TestEmail                   `cmd help:"test email"`
	NearestAirport              airtrackqa.NearestAirport              `cmd help:"find nearest airport"`
	AirportFileStats            airtrackqa.OpenAipAirportStats         `cmd help:"print stats for openaip airport file"`
	EmptyKml                    airtrackqa.EmptyKml                    `cmd help:"compare kml files"`
	DumpMigration               airtrackqa.DumpMigration               `cmd help:"dump a migration file"`
	MictronicsOperatorCountryQA airtrackqa.MictronicsOperatorCountryQA `cmd help:"analyse countries in mictronics operators database"`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	cfg, err := config.ReadConfigFromFile(cli.Config)
	err = ctx.Run(&airtrackqa.Context{Config: cfg})
	ctx.FatalIfErrorf(err)
}
