package airtrackqa

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db/migrations_mysql"
	"github.com/afk11/airtrack/pkg/db/migrations_postgres"
	"github.com/afk11/airtrack/pkg/db/migrations_sqlite3"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/pkg/errors"
)

// DumpMigration - searches for a certain migration and
type DumpMigration struct {
	File   string `help:"migration file to load"`
	Driver string `help:"Database driver"`
}

// Run - searches for a certain migration and
func (e *DumpMigration) Run(ctx *Context) error {
	var s *bindata.AssetSource
	switch e.Driver {
	case config.DatabaseDriverMySQL:
		s = bindata.Resource(migrations_mysql.AssetNames(),
			func(name string) ([]byte, error) {
				return migrations_mysql.Asset(name)
			})
	case config.DatabaseDriverPostgresql:
		s = bindata.Resource(migrations_postgres.AssetNames(),
			func(name string) ([]byte, error) {
				return migrations_postgres.Asset(name)
			})
	case config.DatabaseDriverSqlite3:
		s = bindata.Resource(migrations_sqlite3.AssetNames(),
			func(name string) ([]byte, error) {
				return migrations_sqlite3.Asset(name)
			})
	case "":
		return errors.New("driver was not provided")
	default:
		return errors.New("unknown driver")
	}

	for i := range s.Names {
		if e.File == s.Names[i] {
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
