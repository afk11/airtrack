package airtrack

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db/migrations"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/mysql"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/pkg/errors"
	"os"
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
	m, err := initMigrations(&ctx.Config.Database)
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
	m, err := initMigrations(&ctx.Config.Database)
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

	m, err := initMigrations(&ctx.Config.Database)
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

func initMigrations(dbConf *config.Database) (*migrate.Migrate, error) {
	dbUrl := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		dbConf.Username, dbConf.Password,
		dbConf.Host, dbConf.Port, dbConf.Database)
	s := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			fmt.Println(name)
			return migrations.Asset(name)
		})
	d, err := bindata.WithInstance(s)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dbUrl)
	if err != nil {
		return nil, err
	}
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithInstance(
		"go-bindata",
		d,
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
