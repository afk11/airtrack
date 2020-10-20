package airtrack

import (
	"bufio"
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db/migrations"
	"github.com/pkg/errors"
	"os"
)

type (
	// MigrateUpCmd - executes migrate up
	MigrateUpCmd struct {
		Config string `help:"Configuration file path"`
		Force  bool   `help:"Proceed with task without user confirmation'"`
	}
	// MigrateDownCmd - executes migrate down
	MigrateDownCmd struct {
		Config string `help:"Configuration file path"`
		Force  bool   `help:"Proceed with task without user confirmation'"`
	}
	// MigrateStepsCmd - executes n migration steps
	MigrateStepsCmd struct {
		Config string `help:"Configuration file path"`
		Force  bool   `help:"Proceed with task without user confirmation'"`
		N      int    `help:"how many migrations up (if positive), or down (if negative)"`
	}
)

// Run - triggers up migrations
func (c *MigrateUpCmd) Run() error {
	// Call the Run() method of the selected parsed command.
	cfg, err := config.ReadConfigFromFile(c.Config)
	if err != nil {
		panic(err)
	}

	loc, err := cfg.GetTimeLocation()
	if err != nil {
		return err
	}

	m, err := migrations.InitMigrations(&cfg.Database, loc)
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

// Run - triggers down migrations
func (c *MigrateDownCmd) Run() error {
	// Call the Run() method of the selected parsed command.
	cfg, err := config.ReadConfigFromFile(c.Config)
	if err != nil {
		panic(err)
	}
	loc, err := cfg.GetTimeLocation()
	if err != nil {
		return err
	}

	m, err := migrations.InitMigrations(&cfg.Database, loc)
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

// Run - triggers migration n steps
func (c *MigrateStepsCmd) Run() error {
	if c.N == 0 {
		return errors.Errorf("cannot set n=0 (stay where we are)")
	}
	// Call the Run() method of the selected parsed command.
	cfg, err := config.ReadConfigFromFile(c.Config)
	if err != nil {
		panic(err)
	}
	loc, err := cfg.GetTimeLocation()
	if err != nil {
		return err
	}

	m, err := migrations.InitMigrations(&cfg.Database, loc)
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

func prompt(action string) (bool, error) {
	var reader = bufio.NewReader(os.Stdin)
	fmt.Printf("Proceed with %s (y/N):  \n", action)
	r, _, err := reader.ReadRune()
	if err != nil {
		return false, err
	}
	return r == 'y', nil
}
