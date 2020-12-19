package main

import (
	"github.com/afk11/airtrack/pkg/cmd/airtrack"
	"github.com/alecthomas/kong"
)

var cli struct {
	Version airtrack.VersionCmd `cmd:"" help:"Prints version information"`

	Track airtrack.TrackCmd `cmd:"" help:"Track aircraft"`

	Migrate struct {
		Up    airtrack.MigrateUpCmd    `cmd:"" help:"Migrate to latest database migration"`
		Down  airtrack.MigrateDownCmd  `cmd:"" help:"Rollback all migrations"`
		Steps airtrack.MigrateStepsCmd `cmd:"" help:"Migrate n steps forward if positive, or rollback n if negative"`
	} `cmd:"" help:"Database management functions"`

	Cli struct {
		Project struct {
			List airtrack.ProjectListCmd `cmd:"" help:"List projects"`
		} `cmd:"" help:"Project-related tasks"`
	} `cmd:"" help:"Console interface to airtrack"`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
