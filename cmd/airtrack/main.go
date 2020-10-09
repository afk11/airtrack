package main

import (
	"github.com/afk11/airtrack/pkg/cmd/airtrack"
	"github.com/alecthomas/kong"
)

var cli struct {
	Track airtrack.TrackCmd `cmd help:"Track aircraft"`

	Migrate struct {
		Up    airtrack.MigrateUpCmd    `cmd help:"Migrate to latest database migration"`
		Down  airtrack.MigrateDownCmd  `cmd help:"Rollback all migrations"`
		Steps airtrack.MigrateStepsCmd `cmd help:"Migrate n steps forward if positive, or rollback n if negative"`
	} `cmd help:"Database management functions"`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
