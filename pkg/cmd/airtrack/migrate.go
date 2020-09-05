package airtrack

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db/migrations"
	"github.com/afk11/airtrack/pkg/db/migrations_postgres"
	"github.com/afk11/airtrack/pkg/db/migrations_sqlite3"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database"
	"github.com/golang-migrate/migrate/database/mysql"
	"github.com/golang-migrate/migrate/database/postgres"
	"github.com/golang-migrate/migrate/database/sqlite3"
	"github.com/golang-migrate/migrate/source"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
)

type (
	MigrateUpCmd struct {
		Force bool `help:"Proceed with task without user confirmation'"`
	}
	MigrateDownCmd struct {
		Force bool `help:"Proceed with task without user confirmation'"`
	}
	MigrateStepsCmd struct {
		Force bool `help:"Proceed with task without user confirmation'"`
		N     int  `help:"how many migrations up (if positive), or down (if negative)"`
	}
)

func (c *MigrateUpCmd) Run(ctx *Context) error {
	loc, err := ctx.Config.GetTimeLocation()
	if err != nil {
		return err
	}

	m, err := initMigrations(&ctx.Config.Database, loc)
	if err != nil {
		return err
	}
	if !c.Force {
		c, err := prompt("migration to latest")
		if err != nil {
			return err
		} else if !c {
			return errors.Errorf("task cancelled by user")
		}
	}
	err = m.Up() // run your migrations and handle the errors above of course
	if err != nil {
		return err
	}
	return nil
}

func (c *MigrateDownCmd) Run(ctx *Context) error {
	loc, err := ctx.Config.GetTimeLocation()
	if err != nil {
		return err
	}

	m, err := initMigrations(&ctx.Config.Database, loc)
	if err != nil {
		return err
	}
	if !c.Force {
		c, err := prompt("rollback")
		if err != nil {
			return err
		} else if !c {
			return errors.Errorf("task cancelled by user")
		}
	}
	err = m.Down() // run your migrations and handle the errors above of course
	if err != nil {
		return err
	}
	return nil
}

func (c *MigrateStepsCmd) Run(ctx *Context) error {
	if c.N == 0 {
		return errors.Errorf("cannot set n=0 (stay where we are)")
	}
	loc, err := ctx.Config.GetTimeLocation()
	if err != nil {
		return err
	}

	m, err := initMigrations(&ctx.Config.Database, loc)
	if err != nil {
		return err
	}
	if !c.Force {
		var action string
		if c.N > 0 {
			action = "migrate %d steps forward"
		} else {
			action = "rollback %d stages"
		}
		c, err := prompt(fmt.Sprintf(action, c.N))
		if err != nil {
			return err
		} else if !c {
			return errors.Errorf("task cancelled by user")
		}
	}
	err = m.Steps(c.N) // run your migrations and handle the errors above of course
	if err != nil {
		return err
	}
	return nil
}

func initMigrations(dbConf *config.Database, loc *time.Location) (*migrate.Migrate, error) {
	var db *sql.DB
	var driver database.Driver
	var src source.Driver
	var err error
	switch dbConf.Driver {
	case config.DatabaseDriverMySQL, config.DatabaseDriverPostgresql:
		u, err := dbConf.NetworkDatabaseUrl(loc)
		if err != nil {
			return nil, err
		}
		if dbConf.Driver == config.DatabaseDriverMySQL {
			sep := "?"
			if strings.Contains(u, "?") {
				sep = "&"
			}
			u = u + sep + "multiStatements=true"
		}

		db, err = sql.Open(dbConf.Driver, u)
		if err != nil {
			return nil, err
		}
		var s *bindata.AssetSource
		if dbConf.Driver == config.DatabaseDriverPostgresql {
			s = bindata.Resource(migrations_postgres.AssetNames(),
				func(name string) ([]byte, error) {
					return migrations_postgres.Asset(name)
				})
			driver, err = postgres.WithInstance(db, &postgres.Config{})
		} else {
			s = bindata.Resource(migrations.AssetNames(),
				func(name string) ([]byte, error) {
					return migrations.Asset(name)
				})
			driver, err = mysql.WithInstance(db, &mysql.Config{})
		}
		if err != nil {
			return nil, err
		}
		src, err = bindata.WithInstance(s)
		if err != nil {
			return nil, err
		}
	case config.DatabaseDriverSqlite3:
		u, err := dbConf.Sqlite3Url(loc)
		if err != nil {
			return nil, err
		}
		db, err = sql.Open("sqlite3", u)
		if err != nil {
			return nil, err
		}
		driver, err = sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			return nil, err
		}
		s := bindata.Resource(migrations_sqlite3.AssetNames(),
			func(name string) ([]byte, error) {
				return migrations_sqlite3.Asset(name)
			})
		src, err = bindata.WithInstance(s)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported database driver `" + dbConf.Driver + "`")
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		src,
		dbConf.Database,
		driver,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func prompt(action string) (bool, error) {
	var reader = bufio.NewReader(os.Stdin)
	fmt.Printf("Proceed with %s (y/N):  \n", action)
	r, _, err := reader.ReadRune()
	if err != nil {
		return false, err
	}
	return r == 'y', nil
}
