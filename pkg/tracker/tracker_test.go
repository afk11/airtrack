package tracker

import (
	"database/sql"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/afk11/airtrack/pkg/test"
	"github.com/pkg/errors"
	assert "github.com/stretchr/testify/require"

	"sync"
	"testing"
	"time"
)

var beastSource = &pb.Source{
	Type: pb.Source_BeastServer,
}
var basicOptions = Options{
	SightingTimeout:         time.Second * 30,
	OnGroundUpdateThreshold: 1,
}

func startTracker(database db.Database, c chan *pb.Message, opt Options) *Tracker {
	tr, err := New(database, opt)
	if err != nil {
		panic(err)
	}
	tr.Start(c)
	return tr
}
func doTest(opt Options, proj *Project, testFunc func(tr *Tracker) error) error {
	dbConn, dialect, _, closer := test.InitDBUp()
	defer closer()

	c := make(chan *pb.Message)
	database := db.NewDatabase(dbConn, dialect)
	tr := startTracker(database, c, opt)
	defer tr.Stop()

	err := tr.AddProject(proj)
	if err != nil {
		return errors.Wrapf(err, "failed to add project")
	}

	var wg sync.WaitGroup
	wg.Add(1)
	var gerr error
	go func() {
		defer wg.Done()
		gerr = testFunc(tr)
	}()
	wg.Wait()

	if gerr != nil {
		return gerr
	}

	return nil
}
func TestTracker(t *testing.T) {
	t.Run("startstop", func(t *testing.T) {
		dbConn, dialect, _, closer := test.InitDBUp()
		defer closer()
		c := make(chan *pb.Message)
		database := db.NewDatabase(dbConn, dialect)
		tr := startTracker(database, c, basicOptions)

		err := tr.Stop()
		if err != nil {
			t.Errorf("error stopping tracker: %s", err)
		}
	})

	t.Run("message", func(t *testing.T) {
		opt := basicOptions
		projCfg := config.Project{
			Name: "testproj",
		}
		proj, err := InitProject(projCfg)
		assert.NoError(t, err)
		err = doTest(opt, proj, func(tr *Tracker) error {
			p := pb.Message{Source: beastSource, Icao: "444444"}
			now := time.Now()
			s := tr.getSighting(p.Icao, now)
			defer s.mu.Unlock()
			err := tr.ProcessMessage(proj, s, now, &p)
			if err != nil {
				return errors.Wrap(err, "process message")
			}

			// has global Sighting
			s, ok := tr.sighting["444444"]
			assert.True(t, ok, "should find aircraft sighting after processing message")
			assert.NotNil(t, s)
			assert.Equal(t, p.Icao, s.State.Icao)
			assert.NotNil(t, s.a)
			assert.Equal(t, p.Icao, s.a.Icao)

			// aircraft in DB matches expected values
			ac, err := tr.database.GetAircraftByIcao(p.Icao)
			assert.NoError(t, err, "expecting aircraft to exist")
			assert.NotNil(t, ac, "expecting aircraft to be returned")
			assert.Equal(t, p.Icao, ac.Icao)
			assert.Equal(t, s.a.ID, ac.ID)

			// no sighting yet (until db processing takes place)
			ourSighting, err := tr.database.GetLastSighting(proj.Session, ac)
			assert.Error(t, sql.ErrNoRows, err, "expected last sighting, not error")
			assert.Nil(t, ourSighting, "last sighting should be nil")

			ob, ok := s.observedBy[proj.Session.ID]
			assert.True(t, ok, "should find observation by our project on Sighting")
			assert.Equal(t, proj.Name, ob.project.Name, "observation project should match our project")
			assert.Nil(t, ob.sighting, "sigthing db.Sighting should be nil on init")
			//assert.Equal(t, ourSighting.ID, ob.sighting.ID)

			return nil
		})
		assert.NoError(t, err)
	})
}

func TestProjectObservation(t *testing.T) {
	projCfg := config.Project{
		Name: "testproj",
	}
	proj, err := InitProject(projCfg)
	assert.NoError(t, err)
	proj.Session = &db.Session{ID: 1}
	s1 := &Sighting{
		State: pb.State{
			Icao: "424242",
		},
	}
	s2 := &Sighting{
		State: pb.State{
			Icao: "080800",
		},
	}
	s3 := &Sighting{
		State: pb.State{
			Icao: "421234",
		},
	}
	msgTime := time.Now()
	po1 := NewProjectObservation(proj, s1, msgTime)
	assert.NotNil(t, po1)
	assert.Equal(t, msgTime, po1.firstSeen)
	assert.Equal(t, msgTime, po1.lastSeen)

	assert.False(t, po1.HaveCallSign())
	assert.False(t, po1.HaveSquawk())
	assert.False(t, po1.HaveLocation())
	assert.False(t, po1.HaveAltitudeBarometric())
	assert.False(t, po1.HaveAltitudeGeometric())
	assert.Equal(t, "", po1.CallSign())
	assert.Equal(t, "", po1.Squawk())
	assert.Equal(t, int64(0), po1.AltitudeBarometric())
	assert.Equal(t, int64(0), po1.AltitudeGeometric())
	lat, lon := po1.Location()
	assert.Equal(t, float64(0.0), lat)
	assert.Equal(t, float64(0.0), lon)
	assert.False(t, po1.dirty)

	mta := time.Now()
	assert.NoError(t, po1.SetCallSign("cs1", false, mta))
	assert.True(t, po1.haveCallsign)
	assert.True(t, po1.HaveCallSign())
	assert.Equal(t, "cs1", po1.CallSign())
	assert.Equal(t, 0, len(po1.csLogs))
	assert.False(t, po1.dirty)

	mtb := time.Now().Add(time.Second)
	assert.NoError(t, po1.SetCallSign("cs2", true, mtb))
	assert.True(t, po1.HaveCallSign())
	assert.Equal(t, "cs2", po1.CallSign())
	assert.True(t, po1.dirty)
	assert.Equal(t, 1, len(po1.csLogs))
	assert.Equal(t, "cs2", po1.csLogs[0].callsign)
	assert.Equal(t, mtb, po1.csLogs[0].time)
	assert.Nil(t, po1.csLogs[0].sighting)

	po2 := NewProjectObservation(proj, s2, msgTime)
	assert.NotNil(t, po2)
	assert.Equal(t, msgTime, po2.firstSeen)
	assert.Equal(t, msgTime, po2.lastSeen)
	assert.False(t, po2.dirty)
	assert.False(t, po2.HaveSquawk())

	mtc := time.Now().Add(time.Second * 2)
	assert.NoError(t, po2.SetSquawk("7700", false, mtc))
	assert.Equal(t, "7700", po2.Squawk())
	assert.Equal(t, 0, len(po2.squawkLogs))
	assert.False(t, po2.dirty)
	assert.True(t, po2.HaveSquawk())

	mtd := time.Now().Add(time.Second * 3)
	assert.NoError(t, po2.SetSquawk("7701", true, mtd))
	assert.Equal(t, "7701", po2.Squawk())
	assert.True(t, po2.dirty)
	assert.Equal(t, 1, len(po2.squawkLogs))
	assert.Equal(t, "7701", po2.squawkLogs[0].squawk)
	assert.Equal(t, mtd, po2.squawkLogs[0].time)
	assert.Nil(t, po2.squawkLogs[0].sighting)
	assert.True(t, po2.HaveSquawk())

	po3 := NewProjectObservation(proj, s3, msgTime)
	assert.NotNil(t, po3)
	assert.Equal(t, msgTime, po3.firstSeen)
	assert.Equal(t, msgTime, po3.lastSeen)
	assert.False(t, po3.dirty)
	assert.False(t, po3.haveLocation)
	assert.False(t, po3.HaveLocation())

	assert.NoError(t, po3.SetAltitudeBarometric(10000))
	assert.True(t, po3.HaveAltitudeBarometric())
	assert.Equal(t, int64(10000), po3.AltitudeBarometric())

	mte := time.Now()
	assert.NoError(t, po3.SetLocation(1.00, 2.00, false, mte))
	assert.InDelta(t, 1.00, po3.latitude, 0.00000001)
	assert.InDelta(t, 2.00, po3.longitude, 0.00000001)
	assert.Equal(t, 0, len(po3.locationLogs))
	assert.False(t, po3.dirty)
	assert.True(t, po3.haveLocation)
	assert.True(t, po3.HaveLocation())

	mtf := time.Now().Add(3 * time.Second)
	assert.NoError(t, po3.SetLocation(1.01, 2.01, true, mtf))
	assert.InDelta(t, 1.01, po3.latitude, 0.00000001)
	assert.InDelta(t, 2.01, po3.longitude, 0.00000001)
	assert.Equal(t, 1, len(po3.locationLogs))
	assert.InDelta(t, 1.01, po3.locationLogs[0].lat, 0.00000001)
	assert.InDelta(t, 2.01, po3.locationLogs[0].lon, 0.00000001)
	assert.Equal(t, int64(10000), po3.locationLogs[0].alt)
	assert.Equal(t, mtf, po3.locationLogs[0].time)
	assert.True(t, po3.dirty)
	assert.True(t, po3.haveLocation)
	assert.True(t, po3.HaveLocation())

	assert.NoError(t, po3.SetAltitudeBarometric(10016))
	assert.True(t, po3.HaveAltitudeBarometric())
	assert.Equal(t, int64(10016), po3.AltitudeBarometric())

	mtg := time.Now().Add(7 * time.Second)
	assert.NoError(t, po3.SetLocation(1.02, 2.02, true, mtg))
	assert.InDelta(t, 1.02, po3.latitude, 100)
	assert.InDelta(t, 2.02, po3.longitude, 100)
	assert.Equal(t, 2, len(po3.locationLogs))
	assert.InDelta(t, 1.01, po3.locationLogs[0].lat, 0.00000001)
	assert.InDelta(t, 2.01, po3.locationLogs[0].lon, 0.00000001)
	assert.Equal(t, int64(10000), po3.locationLogs[0].alt)
	assert.Equal(t, mtf, po3.locationLogs[0].time)
	assert.InDelta(t, 1.02, po3.locationLogs[1].lat, 0.00000001)
	assert.InDelta(t, 2.02, po3.locationLogs[1].lon, 0.00000001)
	assert.Equal(t, int64(10016), po3.locationLogs[1].alt)
	assert.Equal(t, mtg, po3.locationLogs[1].time)

	assert.NoError(t, po3.SetAltitudeGeometric(10028))
	assert.True(t, po3.HaveAltitudeGeometric())
	assert.Equal(t, int64(10028), po3.AltitudeGeometric())
}

func TestProjectObservation_LocationUpdateInterval(t *testing.T) {
	var locationUpdateInterval int64 = 3
	projCfg := config.Project{
		Name: "testproj",
		LocationUpdateInterval: &locationUpdateInterval,
	}
	proj, err := InitProject(projCfg)
	assert.NoError(t, err)
	proj.Session = &db.Session{ID: 1}
	s1 := &Sighting{
		State: pb.State{
			Icao: "424242",
		},
	}

	msgTime := time.Now()
	po1 := NewProjectObservation(proj, s1, msgTime)
	assert.NotNil(t, po1)

	// Accepted as there's none there
	lat1 := 1.1112
	lon1 := 52.123123
	assert.NoError(t, po1.SetLocation(lat1, lon1, false, msgTime))
	assert.Equal(t, msgTime, po1.lastLocation)
	assert.Equal(t, lat1, po1.latitude)
	assert.Equal(t, lon1, po1.longitude)

	// Stays the same
	lat2 := 1.1113
	lon2 := 52.123120
	msgTime2 := msgTime.Add(time.Second)
	assert.NoError(t, po1.SetLocation(lat2, lon2, false, msgTime2))
	assert.Equal(t, msgTime, po1.lastLocation)
	assert.Equal(t, lat1, po1.latitude)
	assert.Equal(t, lon1, po1.longitude)

	// 3 seconds have passed, proceed
	lat3 := 1.1115
	lon3 := 52.12318
	msgTime3 := msgTime.Add(time.Second*3)
	assert.NoError(t, po1.SetLocation(lat3, lon3, false, msgTime3))
	assert.Equal(t, msgTime3, po1.lastLocation)
	assert.Equal(t, lat3, po1.latitude)
	assert.Equal(t, lon3, po1.longitude)
}

func TestProjectObservation_LocationUpdateInterval_Zero(t *testing.T) {
	var locationUpdateInterval int64 = 0
	projCfg := config.Project{
		Name: "testproj",
		LocationUpdateInterval: &locationUpdateInterval,
	}
	proj, err := InitProject(projCfg)
	assert.NoError(t, err)
	proj.Session = &db.Session{ID: 1}
	s1 := &Sighting{
		State: pb.State{
			Icao: "424242",
		},
	}

	msgTime := time.Now()
	po1 := NewProjectObservation(proj, s1, msgTime)
	assert.NotNil(t, po1)

	// Accepted as there's none there
	lat1 := 1.1112
	lon1 := 52.123123
	assert.NoError(t, po1.SetLocation(lat1, lon1, false, msgTime))
	assert.Equal(t, msgTime, po1.lastLocation)
	assert.Equal(t, lat1, po1.latitude)
	assert.Equal(t, lon1, po1.longitude)

	// Accepted
	lat2 := 1.1113
	lon2 := 52.123120
	msgTime2 := msgTime.Add(time.Second)
	assert.NoError(t, po1.SetLocation(lat2, lon2, false, msgTime2))
	assert.Equal(t, msgTime2, po1.lastLocation)
	assert.Equal(t, lat2, po1.latitude)
	assert.Equal(t, lon2, po1.longitude)

	// Accepted
	lat3 := 1.1115
	lon3 := 52.12318
	msgTime3 := msgTime.Add(time.Second*3)
	assert.NoError(t, po1.SetLocation(lat3, lon3, false, msgTime3))
	assert.Equal(t, msgTime3, po1.lastLocation)
	assert.Equal(t, lat3, po1.latitude)
	assert.Equal(t, lon3, po1.longitude)
}

func TestProjectObservation_LocationUpdateInterval_NotSet(t *testing.T) {
	projCfg := config.Project{
		Name: "testproj",
	}
	proj, err := InitProject(projCfg)
	assert.NoError(t, err)
	proj.Session = &db.Session{ID: 1}
	s1 := &Sighting{
		State: pb.State{
			Icao: "424242",
		},
	}

	msgTime := time.Now()
	po1 := NewProjectObservation(proj, s1, msgTime)
	assert.NotNil(t, po1)

	// Accepted as there's none there
	lat1 := 1.1112
	lon1 := 52.123123
	assert.NoError(t, po1.SetLocation(lat1, lon1, false, msgTime))
	assert.Equal(t, msgTime, po1.lastLocation)
	assert.Equal(t, lat1, po1.latitude)
	assert.Equal(t, lon1, po1.longitude)

	// Accepted
	lat2 := 1.1113
	lon2 := 52.123120
	msgTime2 := msgTime.Add(time.Second)
	assert.NoError(t, po1.SetLocation(lat2, lon2, false, msgTime2))
	assert.Equal(t, msgTime2, po1.lastLocation)
	assert.Equal(t, lat2, po1.latitude)
	assert.Equal(t, lon2, po1.longitude)

	// Accepted
	lat3 := 1.1115
	lon3 := 52.12318
	msgTime3 := msgTime.Add(time.Second*3)
	assert.NoError(t, po1.SetLocation(lat3, lon3, false, msgTime3))
	assert.Equal(t, msgTime3, po1.lastLocation)
	assert.Equal(t, lat3, po1.latitude)
	assert.Equal(t, lon3, po1.longitude)
}
