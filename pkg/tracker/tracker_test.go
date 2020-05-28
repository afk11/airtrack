package tracker

import (
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/pkg/errors"
	assert "github.com/stretchr/testify/require"

	"github.com/jmoiron/sqlx"
	"sync"
	"testing"
	"time"
)

var sbs1Source = &pb.Source{
	Type: "sbs1",
}
var basicOptions = Options{
	SightingTimeout:         time.Second * 30,
	OnGroundUpdateThreshold: 1,
}

func startTracker(dbConn *sqlx.DB, c chan *pb.Message, opt Options) *Tracker {
	tr, err := New(dbConn, opt)
	if err != nil {
		panic(err)
	}
	tr.Start(c)
	return tr
}
func doTest(opt Options, proj *Project, testFunc func(tr *Tracker) error) error {
	dbConn, _, closer := initDBUp()
	defer closer()

	c := make(chan *pb.Message)
	tr := startTracker(dbConn, c, opt)
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
		dbConn, _, closer := initDBUp()
		defer closer()
		c := make(chan *pb.Message)
		tr := startTracker(dbConn, c, basicOptions)

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
			p := pb.Message{Source: sbs1Source, Icao: "444444"}
			err := tr.ProcessMessage(proj, &p)
			if err != nil {
				return errors.Wrap(err, "process message")
			}

			s, ok := tr.sighting["444444"]
			assert.True(t, ok, "should find aircraft sighting after processing message")
			assert.NotNil(t, s)
			assert.Equal(t, p.Icao, s.State.Icao)
			assert.NotNil(t, s.a)
			assert.Equal(t, p.Icao, s.a.Icao)

			ac, err := db.LoadAircraftByIcao(tr.dbConn, p.Icao)
			assert.NoError(t, err, "expecting aircraft to exist")
			assert.NotNil(t, ac, "expecting aircraft to be returned")
			assert.Equal(t, p.Icao, ac.Icao)
			assert.Equal(t, s.a.Id, ac.Id)

			lastSighting, err := db.LoadLastSighting(tr.dbConn, proj.Session, ac)
			assert.NoError(t, err, "expected last sighting, not error")
			assert.NotNil(t, lastSighting)

			return nil
		})
		assert.NoError(t, err)
	})
}
