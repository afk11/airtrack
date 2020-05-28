package tracker

import (
	"github.com/afk11/airtrack/pkg/config"
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestInitProject(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                []string{string(TrackCallSigns), string(TrackSquawks)},
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: []string{"map_produced", "spotted_in_flight"},
		},
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
	assert.True(t, p.IsFeatureEnabled(TrackCallSigns))
	assert.True(t, p.IsFeatureEnabled(TrackSquawks))
	assert.False(t, p.IsFeatureEnabled(TrackTakeoff))
	assert.False(t, p.IsFeatureEnabled(TrackTakeoff))
	assert.True(t, p.IsEmailNotificationEnabled(MapProduced))
	assert.True(t, p.IsEmailNotificationEnabled(SpottedInFlight))
	assert.False(t, p.IsEmailNotificationEnabled(TakeoffComplete))
	assert.False(t, p.IsEmailNotificationEnabled(TakeoffStart))
	assert.NotNil(t, p.Program)
}
