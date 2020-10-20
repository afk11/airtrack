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
			s := tr.getSighting(p.Icao)
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
