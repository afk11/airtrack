package airtrackqa

import "github.com/afk11/airtrack/pkg/config"

// Context - some global parameters
type Context struct {
	Debug  bool
	Config *config.Config
}
