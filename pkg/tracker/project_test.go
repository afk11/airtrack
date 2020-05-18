package tracker

import (
	"github.com/afk11/airtrack/pkg/config"
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestInitProject_OK(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                []string{string(TrackCallSigns), string(TrackSquawks)},
	}
	p, err := InitProject(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, cfg.Name, p.Name)
	assert.Equal(t, cfg.Filter, p.Filter)
	assert.Equal(t, cfg.ReopenSightings, p.ReopenSightings)
	assert.Equal(t, cfg.ReopenSightingsInterval, int(p.ReopenSightingsInterval.Seconds()))
	assert.Equal(t, 2, len(p.Features))
	assert.Equal(t, TrackCallSigns, p.Features[0])
	assert.Equal(t, TrackSquawks, p.Features[1])
	assert.NotNil(t, p.Program)
}
