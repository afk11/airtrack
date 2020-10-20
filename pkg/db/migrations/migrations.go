package migrations

import (
	"database/sql"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db/migrations_mysql"
	"github.com/afk11/airtrack/pkg/db/migrations_postgres"
	"github.com/afk11/airtrack/pkg/db/migrations_sqlite3"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database"
	"github.com/golang-migrate/migrate/database/mysql"
	"github.com/golang-migrate/migrate/database/postgres"
	"github.com/golang-migrate/migrate/database/sqlite3"
	"github.com/golang-migrate/migrate/source"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/pkg/errors"
	"strings"
	"time"
)

// InitMigrations returns a migrate.Migrate pointer initialized
// for the provided dbConf, or an error if one occurred.
func InitMigrations(dbConf *config.Database, loc *time.Location) (*migrate.Migrate, error) {
	var db *sql.DB
	var driver database.Driver
	var src source.Driver
	var err error
	u, err := dbConf.DataSource(loc)
	if err != nil {
		return nil, err
	}
	switch dbConf.Driver {
	case config.DatabaseDriverMySQL, config.DatabaseDriverPostgresql:
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
			s = bindata.Resource(migrations_mysql.AssetNames(),
				func(name string) ([]byte, error) {
					return migrations_mysql.Asset(name)
				})
			driver, err = mysql.WithInstance(db, &mysql.Config{})
		}
		if err != nil {
			return nil, errors.Wrapf(err, "loading migration resources")
		}
		src, err = bindata.WithInstance(s)
		if err != nil {
			return nil, err
		}
	case config.DatabaseDriverSqlite3:
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
		return nil, errors.Wrapf(err, "init migrations instance")
	}
	return m, nil
}
