package tracker

import (
	"fmt"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/afk11/airtrack/pkg/test"
	"testing"
	"time"
)

func TestTracker(t *testing.T) {
	dbConf := test.MustLoadTestDbConfig()
	tz := test.MustLoadTestTimeZone()
	database := fmt.Sprintf(dbConf.DatabaseFmt, 0)
	dbConn, err := db.NewConn(dbConf.Driver, dbConf.Username, dbConf.Password, dbConf.Host, dbConf.Port, database, tz.Tz)
	if err != nil {
		t.Errorf("error creating db: %s", err)
	}
	tr, err := New(dbConn, Options{
		SightingTimeout:         time.Second * 30,
		OnGroundUpdateThreshold: 1,
	})
	if err != nil {
		t.Errorf("error creating tracker: %s", err)
	}
	c := make(chan *pb.Message)
	tr.Start(c)
	err = tr.Stop()
	if err != nil {
		t.Errorf("error stopping tracker: %s", err)
	}
}
