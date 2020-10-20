package tracker

import (
	"errors"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/pb"
	assert "github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMap_GetProjectAircraft_UnknownProject(t *testing.T) {
	settings := &config.MapSettings{}
	m, err := NewAircraftMap(settings)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	err = m.GetProjectAircraft("unknown", func(i int64, aircrafts []*JSONAircraft) error {
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, ErrUnknownProject, err)
}
func TestMap_GetProjectAircraft_ReceivesError(t *testing.T) {
	settings := &config.MapSettings{}
	m, err := NewAircraftMap(settings)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	cfgP1 := config.Project{
		Name: "project-one",
	}
	p1, err := InitProject(cfgP1)
	assert.NoError(t, err)
	assert.NotNil(t, p1)

	pl := NewMapProjectStatusListener(m)
	pl.Activated(p1)

	expected := errors.New("it happened")
	err = m.GetProjectAircraft(p1.Name, func(i int64, aircrafts []*JSONAircraft) error {
		return expected
	})
	assert.Error(t, err)
	assert.Equal(t, expected, err)
}
func TestMap_DontMap(t *testing.T) {
	settings := &config.MapSettings{}
	m, err := NewAircraftMap(settings)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	cfgP1 := config.Project{
		Name: "project-one",
		Map: &config.ProjectMapSettings{
			Disabled: true,
		},
	}
	p1, err := InitProject(cfgP1)
	assert.NoError(t, err)
	assert.NotNil(t, p1)
	assert.False(t, p1.ShouldMap)

	pl := NewMapProjectStatusListener(m)
	paul := NewMapProjectAircraftUpdateListener(m)

	assert.Equal(t, 0, len(m.projects))
	pl.Activated(p1)
	assert.Equal(t, 0, len(m.projects))

	s1 := &Sighting{
		State: pb.State{
			Icao:     "424242",
			CallSign: "AF1",
			Squawk:   "7700",
		},
	}
	paul.NewAircraft(p1, s1)
	assert.Equal(t, 0, len(m.ac))
	paul.UpdatedAircraft(p1, s1)
	assert.Equal(t, 0, len(m.ac))
	paul.LostAircraft(p1, s1)
	assert.Equal(t, 0, len(m.ac))

	pl.Deactivated(p1)
	assert.Equal(t, 0, len(m.projects))
}
func TestMapNew(t *testing.T) {
	settings := &config.MapSettings{}
	m, err := NewAircraftMap(settings)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, DefaultHistoryInterval, m.historyInterval)
	assert.Equal(t, ":8080", m.s.Addr)

	settings = &config.MapSettings{
		Port: 9912,
	}
	m, err = NewAircraftMap(settings)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, ":9912", m.s.Addr)

	settings = &config.MapSettings{
		HistoryInterval: 120,
	}
	m, err = NewAircraftMap(settings)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, 120*time.Second, m.historyInterval)
}
func TestMapStartAndStop(t *testing.T) {
	settings := &config.MapSettings{}
	m, err := NewAircraftMap(settings)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	m.Serve()
	err = m.Stop()
	assert.NoError(t, err)
}
func TestMap(t *testing.T) {
	settings := &config.MapSettings{}
	m, err := NewAircraftMap(settings)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	assert.Equal(t, 0, len(m.projects))
	cfgP1 := config.Project{
		Name: "project-one",
	}
	cfgP2 := config.Project{
		Name: "project-two",
	}
	p1, err := InitProject(cfgP1)
	assert.NoError(t, err)
	assert.NotNil(t, p1)
	p2, err := InitProject(cfgP2)
	assert.NoError(t, err)
	assert.NotNil(t, p2)

	pl := NewMapProjectStatusListener(m)
	paul := NewMapProjectAircraftUpdateListener(m)
	pl.Activated(p1)

	assert.Equal(t, 1, len(m.projects))
	pi1, ok := m.projects[cfgP1.Name]
	assert.True(t, ok)
	assert.NotNil(t, pi1)
	assert.Equal(t, cfgP1.Name, pi1.name)
	assert.Equal(t, 0, len(pi1.aircraft))

	pl.Activated(p2)
	assert.Equal(t, 2, len(m.projects))
	pi2, ok := m.projects[cfgP2.Name]
	assert.True(t, ok)
	assert.NotNil(t, pi2)
	assert.Equal(t, cfgP2.Name, pi2.name)
	assert.Equal(t, 0, len(pi2.aircraft))

	s1 := &Sighting{
		State: pb.State{
			Icao:     "424242",
			CallSign: "AF1",
			Squawk:   "7700",
		},
	}
	s2 := &Sighting{
		State: pb.State{
			Icao:     "080800",
			CallSign: "USPS231",
			Squawk:   "0542",
		},
	}
	paul.NewAircraft(p1, s1)
	assert.Equal(t, 1, len(pi1.aircraft))
	assert.Equal(t, s1.State.Icao, pi1.aircraft[0])
	assert.Equal(t, 1, len(m.ac))
	ac1, ok := m.ac[s1.State.Icao]
	assert.True(t, ok)
	assert.NotNil(t, ac1)
	assert.Equal(t, s1.State.Icao, ac1.Hex)
	assert.Equal(t, int64(1), ac1.Messages)
	assert.Equal(t, s1.State.CallSign, ac1.Flight)
	assert.Equal(t, s1.State.Squawk, ac1.Squawk)
	assert.Equal(t, int64(1), ac1.referenceCount)
	assert.NoError(t, m.GetProjectAircraft(p1.Name, func(i int64, aircraft []*JSONAircraft) error {
		assert.Equal(t, 1, len(aircraft))
		assert.Equal(t, ac1.Hex, aircraft[0].Hex)
		return nil
	}))

	paul.NewAircraft(p1, s2)
	assert.Equal(t, 2, len(pi1.aircraft))
	assert.Equal(t, s1.State.Icao, pi1.aircraft[0])
	assert.Equal(t, 2, len(m.ac))
	ac2, ok := m.ac[s2.State.Icao]
	assert.True(t, ok)
	assert.NotNil(t, ac2)
	assert.Equal(t, s2.State.Icao, ac2.Hex)
	assert.Equal(t, int64(1), ac2.Messages)
	assert.Equal(t, s2.State.CallSign, ac2.Flight)
	assert.Equal(t, s2.State.Squawk, ac2.Squawk)
	assert.Equal(t, int64(1), ac2.referenceCount)
	assert.NoError(t, m.GetProjectAircraft(p1.Name, func(i int64, aircraft []*JSONAircraft) error {
		assert.Equal(t, 2, len(aircraft))
		assert.Equal(t, ac1.Hex, aircraft[0].Hex)
		assert.Equal(t, ac2.Hex, aircraft[1].Hex)
		return nil
	}))

	paul.NewAircraft(p2, s2)
	assert.Equal(t, int64(2), ac2.referenceCount)
	assert.NoError(t, m.GetProjectAircraft(p2.Name, func(i int64, aircraft []*JSONAircraft) error {
		assert.Equal(t, 1, len(aircraft))
		assert.Equal(t, ac2.Hex, aircraft[0].Hex)
		return nil
	}))

	s2.State.Squawk = "4333"
	paul.UpdatedAircraft(p2, s2)
	assert.Equal(t, s2.State.Squawk, ac2.Squawk)

	s2.State.HaveLocation = true
	s2.State.Latitude = 1.234567
	s2.State.Longitude = 9.87654321
	s2.State.AltitudeBarometric = 32000
	paul.UpdatedAircraft(p2, s2)
	assert.InDelta(t, s2.State.Latitude, ac2.Latitude, 100)
	assert.InDelta(t, s2.State.Longitude, ac2.Longitude, 100)
	assert.Equal(t, s2.State.AltitudeBarometric, ac2.BarometricAltitude)

	paul.LostAircraft(p1, s2)
	assert.Equal(t, 1, len(pi1.aircraft))
	assert.Equal(t, 2, len(m.ac))
	assert.Equal(t, int64(1), ac2.referenceCount)
	_, check := m.ac[s2.State.Icao]
	assert.True(t, check)
	assert.NoError(t, m.GetProjectAircraft(p1.Name, func(i int64, aircraft []*JSONAircraft) error {
		assert.Equal(t, 1, len(aircraft))
		assert.Equal(t, ac1.Hex, aircraft[0].Hex)
		return nil
	}))

	paul.LostAircraft(p1, s1)
	assert.Equal(t, 0, len(pi1.aircraft))
	assert.Equal(t, 1, len(m.ac))
	assert.Equal(t, int64(0), ac1.referenceCount)
	_, check = m.ac[s1.State.Icao]
	assert.False(t, check)
	assert.NoError(t, m.GetProjectAircraft(p1.Name, func(i int64, aircraft []*JSONAircraft) error {
		assert.Equal(t, 0, len(aircraft))
		return nil
	}))

	paul.LostAircraft(p2, s2)
	assert.Equal(t, 0, len(pi2.aircraft))
	assert.Equal(t, 0, len(m.ac))
	assert.Equal(t, int64(0), ac2.referenceCount)
	_, check = m.ac[s2.State.Icao]
	assert.False(t, check)
	assert.NoError(t, m.GetProjectAircraft(p2.Name, func(i int64, aircraft []*JSONAircraft) error {
		assert.Equal(t, 0, len(aircraft))
		return nil
	}))

	pl.Deactivated(p1)
	assert.Equal(t, 1, len(m.projects))
	_, ok = m.projects[cfgP1.Name]
	assert.False(t, ok)
	pl.Deactivated(p2)
	assert.Equal(t, 0, len(m.projects))
	_, ok = m.projects[cfgP2.Name]
	assert.False(t, ok)

}
