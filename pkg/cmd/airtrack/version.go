package airtrack

import (
	"fmt"
)

var version string
var commit string

type (
	// VersionCmd - prints version & build info
	VersionCmd struct{}
)

// Run - prints version & build info
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
