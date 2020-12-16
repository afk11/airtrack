package tracker

import (
	"github.com/afk11/airtrack/pkg/config"
	assert "github.com/stretchr/testify/require"
	"testing"
	"time"
)

func featuresToString(features ...Feature) []string {
	s := make([]string, len(features))
	for i := range features {
		s[i] = string(features[i])
	}
	return s
}
func enToString(en ...EmailNotification) []string {
	s := make([]string, len(en))
	for i := range en {
		s[i] = string(en[i])
	}
	return s
}

var allFeatures = []Feature{
	TrackCallSigns, TrackSquawks, TrackTakeoff,
	TrackKmlLocation, TrackTxTypes, GeocodeEndpoints,
}
var allNotifications = []EmailNotification{
	MapProduced, SpottedInFlight, TakeoffFromAirport,
	TakeoffUnknownAirport, TakeoffComplete,
}

func TestInitProject(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                featuresToString(allFeatures...),
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: enToString(allNotifications...),
		},
	}
	p, err := InitProject(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, cfg.Name, p.Name)
	assert.Equal(t, cfg.Filter, p.Filter)
	assert.True(t, p.ShouldMap)

	assert.Equal(t, cfg.ReopenSightings, p.ReopenSightings)
	assert.Equal(t, cfg.ReopenSightingsInterval, int(p.ReopenSightingsInterval.Seconds()))
	assert.Equal(t, DefaultOnGroundUpdateThreshold, p.OnGroundUpdateThreshold)

	assert.Equal(t, len(allFeatures), len(p.Features))
	for i, f := range allFeatures {
		assert.Equal(t, p.Features[i], f)
		assert.True(t, p.IsFeatureEnabled(f))
	}
	assert.Equal(t, len(allNotifications), len(p.EmailNotifications))
	for i, en := range allNotifications {
		assert.Equal(t, p.EmailNotifications[i], en)
		assert.True(t, p.IsEmailNotificationEnabled(en))
	}

	assert.NotNil(t, p.Program)
}
func TestInitProject_LocationUpdateInterval(t *testing.T) {
	for _, updateInterval := range []int64{1, 10, 30} {
		cfg := config.Project{
			Name:                   "myproj",
			LocationUpdateInterval: &updateInterval,
		}
		p, err := InitProject(cfg)
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(updateInterval)*time.Second, p.LocationUpdateInterval)
	}
}
func TestInitProject_FewerOptions(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                featuresToString(TrackKmlLocation),
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: enToString(MapProduced),
		},
	}
	p, err := InitProject(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, cfg.Name, p.Name)
	assert.Equal(t, cfg.Filter, p.Filter)
	assert.True(t, p.ShouldMap)

	assert.Equal(t, cfg.ReopenSightings, p.ReopenSightings)
	assert.Equal(t, cfg.ReopenSightingsInterval, int(p.ReopenSightingsInterval.Seconds()))
	assert.Equal(t, DefaultOnGroundUpdateThreshold, p.OnGroundUpdateThreshold)

	assert.Equal(t, 1, len(p.Features))
	for _, f := range allFeatures {
		if f != p.Features[0] {
			assert.False(t, p.IsFeatureEnabled(f))
		} else {
			assert.True(t, p.IsFeatureEnabled(f))
		}
	}
	assert.Equal(t, 1, len(p.EmailNotifications))
	for _, en := range allNotifications {
		if en != p.EmailNotifications[0] {
			assert.False(t, p.IsEmailNotificationEnabled(en))
		} else {
			assert.True(t, p.IsEmailNotificationEnabled(en))
		}
	}
}
func TestInitProject_InvalidFilter_TypeError(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `aircraft.Icao`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                []string{string(TrackCallSigns), string(TrackSquawks)},
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: []string{"map_produced", "spotted_in_flight"},
		},
	}
	p, err := InitProject(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "type errors in filter expression")
	assert.Contains(t, err.Error(), "undeclared reference to 'aircraft'")
}
func TestInitProject_InvalidFilter_Syntax(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `]_1+"bah}`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                []string{string(TrackCallSigns), string(TrackSquawks)},
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: []string{"map_produced", "spotted_in_flight"},
		},
	}
	p, err := InitProject(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "failed to parse filter expression")
	//assert.Contains(t, err.Error(), "undeclared reference to 'aircraft'")
}

func TestInitProject_MapSettings(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Map: &config.ProjectMapSettings{
			Disabled: false,
		},
		Features: []string{string(TrackCallSigns), string(TrackSquawks)},
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: []string{"map_produced", "spotted_in_flight"},
		},
	}
	p, err := InitProject(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.True(t, p.ShouldMap)

	cfg.Map.Disabled = true
	p, err = InitProject(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.False(t, p.ShouldMap)
}

func TestInitProject_OnGroundUpdateThreshold(t *testing.T) {
	var threshold int64 = 16
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		OnGroundUpdateThreshold: &threshold,
		Features:                []string{string(TrackCallSigns), string(TrackSquawks)},
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: []string{"map_produced", "spotted_in_flight"},
		},
	}
	p, err := InitProject(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, threshold, p.OnGroundUpdateThreshold)
}

func TestInitProject_Disabled(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		Disabled:                true,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                []string{string(TrackCallSigns), string(TrackSquawks)},
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: []string{"map_produced", "spotted_in_flight"},
		},
	}
	p, err := InitProject(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Equal(t, "cannot init disabled project", err.Error())
}

func TestInitProject_InvalidFeature(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                []string{string("invalid-feature")},
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: []string{"map_produced", "spotted_in_flight"},
		},
	}
	p, err := InitProject(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Equal(t, "unknown feature: invalid-feature", err.Error())
}
func TestInitProject_MissingEmail(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                []string{string(TrackCallSigns)},
		Notifications: &config.Notifications{
			Enabled: []string{"map_produced", "spotted_in_flight"},
		},
	}
	p, err := InitProject(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Equal(t, "notifications missing value for email", err.Error())
}
func TestInitProject_InvalidEmailNotification(t *testing.T) {
	cfg := config.Project{
		Name:                    "myproj",
		Filter:                  `msg.Icao == "000000"`,
		ReopenSightings:         true,
		ReopenSightingsInterval: 5 * 60,
		Features:                []string{string(TrackCallSigns)},
		Notifications: &config.Notifications{
			Email:   "test-email@local.localhost",
			Enabled: []string{"invalid-event"},
		},
	}
	p, err := InitProject(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Equal(t, "unknown email notification: invalid-event", err.Error())
}
