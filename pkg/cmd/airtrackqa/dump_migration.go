package airtrackqa

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/db/migrations"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/pkg/errors"
)

type DumpMigration struct {
	File string `help:"migration file to load"`
}

func (e *DumpMigration) Run(ctx *Context) error {
	s := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	for _, name := range s.Names {
		if e.File == name {
			data, err := s.AssetFunc(e.File)
			if err != nil {
				return errors.Wrapf(err, "reading migration failed")
			}
			fmt.Printf("%s", data)
			fmt.Println()
			return nil
		}
	}
	return errors.Errorf("file not found %s", e.File)
}
