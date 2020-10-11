package airtrack

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

var version string
var commit string

type (
	VersionCmd struct{}
)

func (c *VersionCmd) Run() error {
	if version == "" {
		_, err := fmt.Printf("airtrack development version\n")
		if err != nil {
			return err
		}
	} else {
		_, err := fmt.Printf("airtrack version %s\n", version)
		if err != nil {
			return err
		}
	}
	_, err := fmt.Printf("  revision: %s\n", commit)
	if err != nil {
		return err
	}
	return nil
}
