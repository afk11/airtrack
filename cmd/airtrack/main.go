package main

import (
	"github.com/afk11/airtrack/pkg/cmd/airtrack"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/alecthomas/kong"
)

var cli struct {
	Config   string   `help:"Configuration file path"`
	Projects []string `help:"Projects configuration file (may be repeated, and in addition to main configuration file)"`

	Track       airtrack.TrackCmd    `cmd help:"Track aircraft"`
	GenerateKey airtrack.GenerateKey `cmd help:"Generate an application encryption key"`

	Migrate struct {
		Up    airtrack.MigrateUpCmd    `cmd help:"Migrate to latest database migration"`
		Down  airtrack.MigrateDownCmd  `cmd help:"Rollback all migrations"`
		Steps airtrack.MigrateStepsCmd `cmd help:"Migrate n steps forward if positive, or rollback n if negative"`
	} `cmd help:"Database management functions"`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	cfg, err := config.ReadConfigs(cli.Config, cli.Projects)
	if err != nil {
		panic(err)
	}
	err = ctx.Run(&airtrack.Context{Config: cfg})
	ctx.FatalIfErrorf(err)
}
