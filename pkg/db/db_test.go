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

func TestProject(t *testing.T) {
	loc := test.MustLoadTestTimeZone()
	dbConn, dialect, _, closer := test.InitDBUp()
	defer closer()

	projName := "testProj"
	database := NewDatabase(dbConn, dialect)
	p, err := database.GetProject(projName)
	assert.Error(t, err, "error is expected")
	assert.Nil(t, p, "project not expected")
	assert.Equal(t, sql.ErrNoRows, err, "expected SQL ErrNoRows")

	createdAt := time.Now().In(loc)
	_, err = database.CreateProject(projName, createdAt)
	assert.NoError(t, err)

	p, err = database.GetProject(projName)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, projName, p.Identifier)
	assert.Nil(t, p.Label)
	assert.Equal(t, createdAt.Unix(), p.CreatedAt.Unix())
	assert.Equal(t, createdAt.Unix(), p.UpdatedAt.Unix())
	assert.Nil(t, p.DeletedAt)
}
func ProjectDuplicateName(t *testing.T) {
	loc := test.MustLoadTestTimeZone()
	dbConn, dialect, _, closer := test.InitDBUp()
	defer closer()

	projName := "testProj"
	database := NewDatabase(dbConn, dialect)

	createdAt := time.Now().In(loc)
	_, err := database.CreateProject(projName, createdAt)
	assert.NoError(t, err)

	_, err = database.CreateProject(projName, createdAt)
	assert.Error(t, err)
	isUniqueViolation, err := IsUniqueConstraintViolation(err)
	assert.NoError(t, err)
	assert.True(t, isUniqueViolation)
}
func TestSession(t *testing.T) {
	loc := test.MustLoadTestTimeZone()
	dbConn, dialect, _, closer := test.InitDBUp()
	defer closer()

	projName := "testProj"
	database := NewDatabase(dbConn, dialect)

	createdAt := time.Now().In(loc)
	_, err := database.CreateProject(projName, createdAt)
	assert.NoError(t, err)

	p, err := database.GetProject(projName)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	identUUID, err := uuid.NewRandom()
	assert.NoError(t, err)
	ident := identUUID.String()

	sess, err := database.GetSessionByIdentifier(p, ident)
	assert.Error(t, err, "error is expected")
	assert.Nil(t, sess)
	assert.Equal(t, sql.ErrNoRows, err, "expected SQL ErrNoRows")

	_, err = database.CreateSession(p, ident, false, false, false)
	assert.NoError(t, err)
	sess, err = database.GetSessionByIdentifier(p, ident)
	assert.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, ident, sess.Identifier)
	assert.False(t, sess.WithCallSigns)
	assert.False(t, sess.WithSquawks)
	assert.False(t, sess.WithTransmissionTypes)

	_, err = database.CreateSession(p, ident+"1", true, false, false)
	assert.NoError(t, err)
	sess1, err := database.GetSessionByIdentifier(p, ident+"1")
	assert.NoError(t, err)
	assert.NotNil(t, sess1)
	assert.True(t, sess1.WithSquawks)
	assert.False(t, sess1.WithTransmissionTypes)
	assert.False(t, sess1.WithCallSigns)

	_, err = database.CreateSession(p, ident+"2", false, true, false)
	assert.NoError(t, err)
	sess2, err := database.GetSessionByIdentifier(p, ident+"2")
	assert.NoError(t, err)
	assert.NotNil(t, sess2)
	assert.False(t, sess2.WithSquawks)
	assert.True(t, sess2.WithTransmissionTypes)
	assert.False(t, sess2.WithCallSigns)

	_, err = database.CreateSession(p, ident+"3", false, false, true)
	assert.NoError(t, err)
	sess3, err := database.GetSessionByIdentifier(p, ident+"3")
	assert.NoError(t, err)
	assert.NotNil(t, sess3)
	assert.False(t, sess3.WithSquawks)
	assert.False(t, sess3.WithTransmissionTypes)
	assert.True(t, sess3.WithCallSigns)

	closeTime := createdAt.Add(time.Second * 6)
	_, err = database.CloseSession(sess, closeTime)
	assert.NoError(t, err)
	_, err = database.CloseSession(sess1, closeTime)
	assert.NoError(t, err)
	_, err = database.CloseSession(sess2, closeTime)
	assert.NoError(t, err)
	_, err = database.CloseSession(sess3, closeTime)
	assert.NoError(t, err)

	assert.Equal(t, closeTime.Unix(), sess.ClosedAt.Unix())
	assert.Equal(t, closeTime.Unix(), sess1.ClosedAt.Unix())
	assert.Equal(t, closeTime.Unix(), sess2.ClosedAt.Unix())
	assert.Equal(t, closeTime.Unix(), sess3.ClosedAt.Unix())
}
func TestSessionDuplicateIdentifier(t *testing.T) {
	loc := test.MustLoadTestTimeZone()
	dbConn, dialect, _, closer := test.InitDBUp()
	defer closer()

	projName := "testProj"
	database := NewDatabase(dbConn, dialect)

	createdAt := time.Now().In(loc)
	_, err := database.CreateProject(projName, createdAt)
	assert.NoError(t, err)

	p, err := database.GetProject(projName)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	identUUID, err := uuid.NewRandom()
	assert.NoError(t, err)
	ident := identUUID.String()

	_, err = database.CreateSession(p, ident, false, false, true)
	assert.NoError(t, err)
	_, err = database.CreateSession(p, ident, false, false, true)
	assert.Error(t, err)
	isUniqueViolation, err := IsUniqueConstraintViolation(err)
	assert.NoError(t, err)
	assert.True(t, isUniqueViolation)
}
func TestAircraft(t *testing.T) {
	dbConn, dialect, _, closer := test.InitDBUp()
	defer closer()
	database := NewDatabase(dbConn, dialect)

	icao := "7f80ff"
	_, err := database.GetAircraftByIcao(icao)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)

	n := time.Now()
	_, err = database.CreateAircraft(icao, n)
	assert.NoError(t, err)

	a, err := database.GetAircraftByIcao(icao)
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, icao, a.Icao)
	assert.Equal(t, n.Unix(), a.CreatedAt.Unix())
	assert.Equal(t, n.Unix(), a.UpdatedAt.Unix())

	a, err = database.GetAircraftByID(a.ID)
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, icao, a.Icao)
}
func TestSighting(t *testing.T) {
	loc := test.MustLoadTestTimeZone()

	dbConn, dialect, _, closer := test.InitDBUp()
	defer closer()
	database := NewDatabase(dbConn, dialect)

	projName := "testProj"
	icao := "123456"
	createdAt := time.Now()
	_, err := database.CreateAircraft(icao, createdAt)
	assert.NoError(t, err)

	a, err := database.GetAircraftByIcao(icao)
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, icao, a.Icao)
	assert.Equal(t, createdAt.Unix(), a.CreatedAt.Unix())
	assert.Equal(t, createdAt.Unix(), a.UpdatedAt.Unix())

	_, err = database.CreateProject(projName, createdAt)
	assert.NoError(t, err)
	p, err := database.GetProject(projName)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	identUUID, err := uuid.NewRandom()
	assert.NoError(t, err)
	ident := identUUID.String()

	_, err = database.CreateSession(p, ident, false, false, false)
	assert.NoError(t, err)
	sess, err := database.GetSessionByIdentifier(p, ident)
	assert.NoError(t, err)
	assert.NotNil(t, sess)

	sighting, err := database.GetLastSighting(sess, a)
	assert.Error(t, sql.ErrNoRows, err)
	assert.Nil(t, sighting)

	_, err = database.CreateSighting(sess, a, createdAt)
	assert.NoError(t, err)
	sighting, err = database.GetLastSighting(sess, a)
	assert.NoError(t, err)
	assert.NotNil(t, sighting)
	assert.Nil(t, sighting.ClosedAt)
	assert.Equal(t, sess.ID, sighting.SessionID)
	assert.Equal(t, a.ID, sighting.AircraftID)
	assert.Equal(t, createdAt.Unix(), sighting.CreatedAt.Unix())
	assert.Equal(t, createdAt.Unix(), sighting.UpdatedAt.Unix())
	assert.Nil(t, sighting.CallSign)
	assert.Nil(t, sighting.Squawk)
	assert.Equal(t, uint8(0), sighting.TransmissionTypes)

	// again
	sightingByID, err := database.GetSightingByID(sighting.ID)
	assert.NoError(t, err)
	assert.Equal(t, sighting.ID, sightingByID.ID)
	// again
	assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
		s, err := database.GetSightingByIDTx(tx, sighting.ID)
		assert.NoError(t, err)
		assert.NotNil(t, s)
		assert.Equal(t, sighting.ID, s.ID)
		return nil
	}))
	// again
	assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
		s, err := database.GetLastSightingTx(tx, sess, a)
		assert.NoError(t, err)
		assert.NotNil(t, s)
		assert.Equal(t, sighting.ID, s.ID)
		return nil
	}))
	// todo: test GetLastSighting with multiple sightings

	now := time.Now().In(loc)
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
		cs, err := database.GetLastSightingCallSignTx(tx, sighting)
		assert.NoError(t, err)
		assert.NotNil(t, cs)
		assert.Equal(t, callsign, cs.CallSign)
		assert.Equal(t, now.Unix(), cs.ObservedAt.Unix())
		assert.Equal(t, sighting.ID, cs.SightingID)
		return nil
	}))

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
		sk, err := database.GetLastSightingSquawkTx(tx, sighting)
		assert.NoError(t, err)
		assert.NotNil(t, sk)
		assert.Equal(t, squawk, sk.Squawk)
		assert.Equal(t, now.Unix(), sk.ObservedAt.Unix())
		assert.Equal(t, sighting.ID, sk.SightingID)
		return nil
	}))

	// close and reopen
	err = database.CloseSightingBatch([]*Sighting{sighting}, now.Add(time.Second*1))
	assert.NoError(t, err)
	assert.NotNil(t, sighting.ClosedAt)
	_, err = database.ReopenSighting(sighting)
	assert.NoError(t, err)
	assert.Nil(t, sighting.ClosedAt)

	// close and reopen tx
	err = database.CloseSightingBatch([]*Sighting{sighting}, now.Add(time.Second*3))
	assert.NoError(t, err)
	assert.NotNil(t, sighting.ClosedAt)
	assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
		_, err = database.ReopenSightingTx(tx, sighting)
		assert.NoError(t, err)
		assert.Nil(t, sighting.ClosedAt)
		return nil
	}))

	err = database.CloseSightingBatch([]*Sighting{sighting}, now.Add(time.Second*5))
	assert.NoError(t, err)
	assert.NotNil(t, sighting.ClosedAt)
}
func TestSightingLocation(t *testing.T) {
	dbConn, dialect, _, closer := test.InitDBUp()
	defer closer()
	database := NewDatabase(dbConn, dialect)

	projName := "testProj"
	icao := "123456"
	createdAt := time.Now()
	_, err := database.CreateAircraft(icao, createdAt)
	assert.NoError(t, err)

	a, err := database.GetAircraftByIcao(icao)
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, icao, a.Icao)

	_, err = database.CreateProject(projName, createdAt)
	assert.NoError(t, err)
	p, err := database.GetProject(projName)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	identUUID, err := uuid.NewRandom()
	assert.NoError(t, err)
	ident := identUUID.String()

	_, err = database.CreateSession(p, ident, false, false, false)
	assert.NoError(t, err)
	sess, err := database.GetSessionByIdentifier(p, ident)
	assert.NoError(t, err)
	assert.NotNil(t, sess)

	_, err = database.CreateSighting(sess, a, createdAt)
	assert.NoError(t, err)
	sighting, err := database.GetLastSighting(sess, a)
	assert.NoError(t, err)
	assert.NotNil(t, sighting)

	// expect
	empty, err := database.LoadLocationHistory(sighting, 0, 100)
	assert.NoError(t, err)
	assert.NotNil(t, empty)
	assert.False(t, empty.Next())

	now := time.Now()
	lat1 := 1.9876523
	lon1 := 1.234789
	alt1 := int64(10000)
	_, err = database.CreateSightingLocation(sighting.ID, now, alt1, lat1, lon1)
	assert.NoError(t, err)

	lat2 := 1.9899523
	lon2 := 1.238889
	alt2 := int64(10015)
	assert.NoError(t, database.Transaction(func(tx *sqlx.Tx) error {
		_, err = database.CreateSightingLocationTx(tx, sighting.ID, now, alt2, lat2, lon2)
		assert.NoError(t, err)
		return nil
	}))

	history, err := database.GetFullLocationHistory(sighting, 100)
	assert.NoError(t, err)
	assert.NotNil(t, history)
	assert.Equal(t, 2, len(history))
	assert.InDelta(t, lat1, history[0].Latitude, 0.0000001)
	assert.InDelta(t, lon1, history[0].Longitude, 0.0000001)
	assert.Equal(t, alt1, history[0].Altitude)
	assert.InDelta(t, lat2, history[1].Latitude, 0.0000001)
	assert.InDelta(t, lon2, history[1].Longitude, 0.0000001)
	assert.Equal(t, alt2, history[1].Altitude)

	sightingKml, err := database.GetSightingKml(sighting)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, sightingKml)

	kStr := []byte("not-quite-kml")

	_, err = database.CreateSightingKmlContent(sighting, kStr)
	assert.NoError(t, err)
	sightingKml, err = database.GetSightingKml(sighting)
	assert.NoError(t, err)
	assert.NotNil(t, sightingKml)
	assert.Equal(t, sighting.ID, sightingKml.SightingID)

	decoded, err := sightingKml.DecodedKml()
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(kStr, decoded))

	err = sightingKml.UpdateKml([]byte("different kml"))
	assert.NoError(t, err)

	_, err = database.UpdateSightingKml(sightingKml)
	assert.NoError(t, err)

	sightingKml, err = database.GetSightingKml(sighting)
	assert.NoError(t, err)
	assert.NotNil(t, sightingKml)
	decoded, err = sightingKml.DecodedKml()
	assert.NoError(t, err)
	assert.True(t, bytes.Equal([]byte("different kml"), decoded))

	err = database.CloseSightingBatch([]*Sighting{sighting}, time.Now().Add(time.Second))
	assert.NoError(t, err)
	assert.NotNil(t, sighting.ClosedAt)
}
func TestEmail(t *testing.T) {
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
		assert.Equal(t, int32(EmailFailed), rows[0].Status)
		return nil
	}))
}
