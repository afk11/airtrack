package db

import (
	"bytes"
	"database/sql"
	"github.com/afk11/airtrack/pkg/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	assert "github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSomething(t *testing.T) {
	t.Run("Project", func(t *testing.T) {
		loc := test.MustLoadTestTimeZone()
		dbConn, dialect, _, closer := test.InitDBUp()
		defer closer()

		projName := "testProj"
		database := NewDatabase(dbConn, dialect)
		p, err := database.LoadProject(projName)
		assert.Error(t, err, "error is expected")
		assert.Nil(t, p, "project not expected")
		assert.Equal(t, sql.ErrNoRows, err, "expected SQL ErrNoRows")

		createdAt := time.Now().In(loc.Tz)
		_, err = database.NewProject(projName, createdAt)
		assert.NoError(t, err)

		p, err = database.LoadProject(projName)
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, projName, p.Identifier)
		assert.Nil(t, p.Label)
		assert.Equal(t, createdAt, p.CreatedAt)
		assert.Equal(t, createdAt, p.UpdatedAt)
		assert.Nil(t, p.DeletedAt)
	})
	t.Run("Session", func(t *testing.T) {
		loc := test.MustLoadTestTimeZone()
		dbConn, dialect, _, closer := test.InitDBUp()
		defer closer()

		projName := "testProj"
		database := NewDatabase(dbConn, dialect)

		createdAt := time.Now().In(loc.Tz)
		_, err := database.NewProject(projName, createdAt)
		assert.NoError(t, err)

		p, err := database.LoadProject(projName)
		assert.NoError(t, err)
		assert.NotNil(t, p)

		identUuid, err := uuid.NewRandom()
		assert.NoError(t, err)
		ident := identUuid.String()

		sess, err := database.LoadSessionByIdentifier(p, ident)
		assert.Error(t, err, "error is expected")
		assert.Nil(t, sess)
		assert.Equal(t, sql.ErrNoRows, err, "expected SQL ErrNoRows")

		_, err = database.NewSession(p, ident, false, false, false)
		assert.NoError(t, err)
		sess, err = database.LoadSessionByIdentifier(p, ident)
		assert.NoError(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, ident, sess.Identifier)
		assert.False(t, sess.WithCallSigns)
		assert.False(t, sess.WithSquawks)
		assert.False(t, sess.WithTransmissionTypes)

		_, err = database.NewSession(p, ident+"1", true, false, false)
		assert.NoError(t, err)
		sess1, err := database.LoadSessionByIdentifier(p, ident+"1")
		assert.NoError(t, err)
		assert.NotNil(t, sess1)
		assert.True(t, sess1.WithSquawks)
		assert.False(t, sess1.WithTransmissionTypes)
		assert.False(t, sess1.WithCallSigns)

		_, err = database.NewSession(p, ident+"2", false, true, false)
		assert.NoError(t, err)
		sess2, err := database.LoadSessionByIdentifier(p, ident+"2")
		assert.NoError(t, err)
		assert.NotNil(t, sess2)
		assert.False(t, sess2.WithSquawks)
		assert.True(t, sess2.WithTransmissionTypes)
		assert.False(t, sess2.WithCallSigns)

		_, err = database.NewSession(p, ident+"3", false, false, true)
		assert.NoError(t, err)
		sess3, err := database.LoadSessionByIdentifier(p, ident+"3")
		assert.NoError(t, err)
		assert.NotNil(t, sess3)
		assert.False(t, sess3.WithSquawks)
		assert.False(t, sess3.WithTransmissionTypes)
		assert.True(t, sess3.WithCallSigns)

		_, err = database.CloseSession(sess)
		assert.NoError(t, err)
		_, err = database.CloseSession(sess1)
		assert.NoError(t, err)
		_, err = database.CloseSession(sess2)
		assert.NoError(t, err)
		_, err = database.CloseSession(sess3)
		assert.NoError(t, err)

		// todo: test value of ClosedAt
		assert.NotNil(t, sess.ClosedAt)
		assert.NotNil(t, sess1.ClosedAt)
		assert.NotNil(t, sess2.ClosedAt)
		assert.NotNil(t, sess3.ClosedAt)
	})
	t.Run("Aircraft", func(t *testing.T) {
		dbConn, dialect, _, closer := test.InitDBUp()
		defer closer()
		database := NewDatabase(dbConn, dialect)

		icao := "7f80ff"
		_, err := database.LoadAircraftByIcao(icao)
		assert.Error(t, err)
		assert.Equal(t, sql.ErrNoRows, err)

		_, err = database.CreateAircraft(icao)
		assert.NoError(t, err)

		a, err := database.LoadAircraftByIcao(icao)
		assert.NoError(t, err)
		assert.NotNil(t, a)
		assert.Equal(t, icao, a.Icao)
		// todo: test value of aircraft created_at
	})
	t.Run("Sighting", func(t *testing.T) {
		dbConn, dialect, _, closer := test.InitDBUp()
		defer closer()
		database := NewDatabase(dbConn, dialect)

		projName := "testProj"
		icao := "123456"
		createdAt := time.Now()
		_, err := database.CreateAircraft(icao)
		assert.NoError(t, err)

		a, err := database.LoadAircraftByIcao(icao)
		assert.NoError(t, err)
		assert.NotNil(t, a)
		assert.Equal(t, icao, a.Icao)

		_, err = database.NewProject(projName, createdAt)
		assert.NoError(t, err)
		p, err := database.LoadProject(projName)
		assert.NoError(t, err)
		assert.NotNil(t, p)

		identUuid, err := uuid.NewRandom()
		assert.NoError(t, err)
		ident := identUuid.String()

		_, err = database.NewSession(p, ident, false, false, false)
		assert.NoError(t, err)
		sess, err := database.LoadSessionByIdentifier(p, ident)
		assert.NoError(t, err)
		assert.NotNil(t, sess)

		sighting, err := database.LoadLastSighting(sess, a)
		assert.Error(t, sql.ErrNoRows, err)
		assert.Nil(t, sighting)

		_, err = database.CreateSighting(sess, a)
		assert.NoError(t, err)
		sighting, err = database.LoadLastSighting(sess, a)
		assert.NoError(t, err)
		assert.NotNil(t, sighting)
		assert.Nil(t, sighting.ClosedAt)
		assert.Equal(t, sess.Id, sighting.SessionId)
		assert.Equal(t, a.Id, sighting.AircraftId)
		assert.Nil(t, sighting.CallSign)
		assert.Nil(t, sighting.Squawk)
		assert.Equal(t, uint8(0), sighting.TransmissionTypes)

		// again
		sightingById, err := database.LoadSightingById(sighting.Id)
		assert.NoError(t, err)
		assert.Equal(t, sighting.Id, sightingById.Id)
		// again
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			s, err := database.LoadSightingByIdTx(tx, sighting.Id)
			assert.NoError(t, err)
			assert.NotNil(t, s)
			assert.Equal(t, sighting.Id, s.Id)
			return nil
		}))
		// again
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			s, err := database.LoadLastSightingTx(tx, sess, a)
			assert.NoError(t, err)
			assert.NotNil(t, s)
			assert.Equal(t, sighting.Id, s.Id)
			return nil
		}))
		// todo: test LoadLastSighting with multiple sightings

		now := time.Now()
		callsign := "UPS1234"
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err := database.UpdateSightingCallsignTx(tx, sighting, callsign)
			assert.NoError(t, err)
			return nil
		}))
		assert.NotNil(t, sighting.CallSign)
		assert.Equal(t, callsign, *sighting.CallSign)
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err := database.CreateNewSightingCallSignTx(tx, sighting, callsign, now)
			assert.NoError(t, err)
			return nil
		}))
		// todo: should load and check callsign

		squawk := "7700"
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err := database.UpdateSightingSquawkTx(tx, sighting, squawk)
			assert.NoError(t, err)
			return nil
		}))
		assert.NotNil(t, sighting.Squawk)
		assert.Equal(t, squawk, *sighting.Squawk)
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err := database.CreateNewSightingSquawkTx(tx, sighting, squawk, now)
			assert.NoError(t, err)
			return nil
		}))
		// todo: should load and check squawk

		// close and reopen
		err = database.CloseSightingBatch([]*Sighting{sighting})
		assert.NoError(t, err)
		assert.NotNil(t, sighting.ClosedAt)
		_, err = database.ReopenSighting(sighting)
		assert.NoError(t, err)
		assert.Nil(t, sighting.ClosedAt)

		// close and reopen tx
		err = database.CloseSightingBatch([]*Sighting{sighting})
		assert.NoError(t, err)
		assert.NotNil(t, sighting.ClosedAt)
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err = database.ReopenSightingTx(tx, sighting)
			assert.NoError(t, err)
			assert.Nil(t, sighting.ClosedAt)
			return nil
		}))

		err = database.CloseSightingBatch([]*Sighting{sighting})
		assert.NoError(t, err)
		assert.NotNil(t, sighting.ClosedAt)
	})
	t.Run("SightingLocation", func(t *testing.T) {
		dbConn, dialect, _, closer := test.InitDBUp()
		defer closer()
		database := NewDatabase(dbConn, dialect)

		projName := "testProj"
		icao := "123456"
		createdAt := time.Now()
		_, err := database.CreateAircraft(icao)
		assert.NoError(t, err)

		a, err := database.LoadAircraftByIcao(icao)
		assert.NoError(t, err)
		assert.NotNil(t, a)
		assert.Equal(t, icao, a.Icao)

		_, err = database.NewProject(projName, createdAt)
		assert.NoError(t, err)
		p, err := database.LoadProject(projName)
		assert.NoError(t, err)
		assert.NotNil(t, p)

		identUuid, err := uuid.NewRandom()
		assert.NoError(t, err)
		ident := identUuid.String()

		_, err = database.NewSession(p, ident, false, false, false)
		assert.NoError(t, err)
		sess, err := database.LoadSessionByIdentifier(p, ident)
		assert.NoError(t, err)
		assert.NotNil(t, sess)

		_, err = database.CreateSighting(sess, a)
		assert.NoError(t, err)
		sighting, err := database.LoadLastSighting(sess, a)
		assert.NoError(t, err)
		assert.NotNil(t, sighting)

		// expect
		empty, err := database.GetLocationHistory(sighting, 0, 100)
		assert.NoError(t, err)
		assert.NotNil(t, empty)
		assert.False(t, empty.Next())

		now := time.Now()
		lat1 := 1.9876523
		lon1 := 1.234789
		alt1 := int64(10000)
		_, err = database.InsertSightingLocation(sighting.Id, now, alt1, lat1, lon1)
		assert.NoError(t, err)

		lat2 := 1.9899523
		lon2 := 1.238889
		alt2 := int64(10015)
		_, err = database.InsertSightingLocation(sighting.Id, now, alt2, lat2, lon2)
		assert.NoError(t, err)

		history, err := database.GetFullLocationHistory(sighting, 100)
		assert.NoError(t, err)
		assert.NotNil(t, history)
		assert.Equal(t, 2, len(history))
		assert.Equal(t, lat1, history[0].Latitude)
		assert.Equal(t, lon1, history[0].Longitude)
		assert.Equal(t, alt1, history[0].Altitude)
		assert.Equal(t, lat2, history[1].Latitude)
		assert.Equal(t, lon2, history[1].Longitude)
		assert.Equal(t, alt2, history[1].Altitude)

		sightingKml, err := database.LoadSightingKml(sighting)
		assert.Error(t, err)
		assert.Equal(t, sql.ErrNoRows, err)
		assert.Nil(t, sightingKml)

		kStr := []byte("not-quite-kml")

		_, err = database.CreateSightingKml(sighting, kStr)
		assert.NoError(t, err)
		sightingKml, err = database.LoadSightingKml(sighting)
		assert.NoError(t, err)
		assert.NotNil(t, sightingKml)
		assert.Equal(t, sighting.Id, sightingKml.SightingId)

		decoded, err := sightingKml.DecodedKml()
		assert.NoError(t, err)
		assert.True(t, bytes.Equal(kStr, decoded))

		err = sightingKml.UpdateKml([]byte("different kml"))
		assert.NoError(t, err)

		_, err = database.UpdateSightingKml(sightingKml)
		assert.NoError(t, err)

		sightingKml, err = database.LoadSightingKml(sighting)
		assert.NoError(t, err)
		assert.NotNil(t, sightingKml)
		decoded, err = sightingKml.DecodedKml()
		assert.NoError(t, err)
		assert.True(t, bytes.Equal([]byte("different kml"), decoded))

		err = database.CloseSightingBatch([]*Sighting{sighting})
		assert.NoError(t, err)
		assert.NotNil(t, sighting.ClosedAt)
	})
	t.Run("Email", func(t *testing.T) {
		dbConn, dialect, _, closer := test.InitDBUp()
		defer closer()
		database := NewDatabase(dbConn, dialect)

		now := time.Now()
		rows, err := database.GetPendingEmailJobs(now)
		assert.Nil(t, err)
		assert.Nil(t, rows)

		encoded := []byte(`{"to":"testuser@developer.local","subject":"[unittest] 42424242 (THIC4F): spotted in flight","body":"Project: unittest\u003cbr /\u003e\n\n42424242\n\n THIC4F\n\nspotted in flight.\n\u003cbr /\u003e\n\u003cbr /\u003e\n\u003cul\u003e\n    \u003cli\u003eTime: 03 Oct 20 18:40 IST\u003c/li\u003e\n    \u003cli\u003ePlace: \u003ca href=\"https://www.openstreetmap.org/#map=13/0/0\"\u003e0, 0\u003c/a\u003e @ 0 ft\u003c/li\u003e\n\u003c/ul\u003e\n","attachments":null}`)
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err = database.CreateEmailJobTx(tx, now, encoded)
			assert.NoError(t, err)
			return nil
		}))

		rows, err = database.GetPendingEmailJobs(now)
		assert.Nil(t, err)
		assert.NotNil(t, rows)
		assert.Equal(t, 1, len(rows))
		assert.True(t, bytes.Equal(encoded, rows[0].Job))

		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err := database.DeleteCompletedEmailTx(tx, rows[0])
			assert.NoError(t, err)
			return nil
		}))

		rows, err = database.GetPendingEmailJobs(now)
		assert.Nil(t, err)
		assert.Nil(t, rows)

		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err = database.CreateEmailJobTx(tx, now, encoded)
			assert.NoError(t, err)
			return nil
		}))
		rows, err = database.GetPendingEmailJobs(now)
		assert.Nil(t, err)
		assert.NotNil(t, rows)
		later := time.Now().Add(time.Second * 10)
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err = database.RetryEmailAfterTx(tx, &rows[0], later)
			assert.NoError(t, err)
			assert.Equal(t, later, *rows[0].RetryAfter)
			assert.Equal(t, int32(1), rows[0].Retries)
			return nil
		}))
		later = later.Add(time.Second * 10)
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err = database.RetryEmailAfterTx(tx, &rows[0], later)
			assert.NoError(t, err)
			assert.Equal(t, later, *rows[0].RetryAfter)
			assert.Equal(t, int32(2), rows[0].Retries)
			return nil
		}))
		assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
			_, err = database.MarkEmailFailedTx(tx, &rows[0])
			assert.NoError(t, err)
			assert.Equal(t, time.Unix(0, 0), *rows[0].RetryAfter)
			assert.Equal(t, int32(EmailFailed), rows[0].Status)
			return nil
		}))
	})
}
