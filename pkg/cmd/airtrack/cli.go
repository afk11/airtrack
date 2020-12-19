package airtrack

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type (
	// ProjectListCmd - prints a list of projects
	ProjectListCmd struct {
		Config string `help:"Configuration file path"`
	}
)

// Run - triggers up migrations
func (c *ProjectListCmd) Run() error {
	// Call the Run() method of the selected parsed command.
	cfg, err := config.ReadConfigFromFile(c.Config)
	if err != nil {
		panic(err)
	}

	loc, err := cfg.GetTimeLocation()
	if err != nil {
		return err
	}

	dbURL, err := cfg.Database.DataSource(loc)
	if err != nil {
		return errors.Wrapf(err, "creating database connection parameters")
	}
	dbConn, err := sqlx.Connect(cfg.Database.Driver, dbURL)
	if err != nil {
		return errors.Wrapf(err, "creating database connection")
	}
	dbh := db.NewDatabase(dbConn, goqu.Dialect(cfg.Database.Driver))
	fmt.Println(dbh)
	return nil
}
