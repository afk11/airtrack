package db

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"io/ioutil"
	"time"
)

const (
	EmailPending            = 1
	EmailFailed             = 0
	KmlPlainTextContentType = 0
	KmlGzipContentType      = 1
)

type (
	// Project database record. Created the first time
	// a project is used.
	Project struct {
		Id         uint64    `db:"id"`
		Identifier string    `db:"identifier"`
		Label      *string   `db:"label"`
		CreatedAt  time.Time `db:"created_at"`
		UpdatedAt  time.Time `db:"updated_at"`
		// todo: why is project here? should it be deleted?
		DeletedAt *time.Time `db:"deleted_at"`
	}
	// Session database record. Created each time a project
	// is used.
	Session struct {
		Id                    uint64     `db:"id"`
		Identifier            string     `db:"identifier"`
		ProjectId             uint64     `db:"project_id"`
		ClosedAt              *time.Time `db:"closed_at"`
		CreatedAt             time.Time  `db:"created_at"`
		UpdatedAt             time.Time  `db:"updated_at"`
		DeletedAt             *time.Time `db:"deleted_at"`
		WithSquawks           bool       `db:"with_squawks"`
		WithTransmissionTypes bool       `db:"with_transmission_types"`
		WithCallSigns         bool       `db:"with_callsigns"`
	}
	// Aircraft database record. There will be one of these per icao.
	Aircraft struct {
		Id        uint64    `db:"id"`
		Icao      string    `db:"icao"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	// Sighting database record. Created every time an aircraft is
	// watched by a project in a session.
	Sighting struct {
		Id         uint64  `db:"id"`
		ProjectId  uint64  `db:"project_id"`
		SessionId  uint64  `db:"session_id"`
		AircraftId uint64  `db:"aircraft_id"`
		CallSign   *string `db:"callsign"`

		CreatedAt time.Time  `db:"created_at"`
		UpdatedAt time.Time  `db:"updated_at"`
		ClosedAt  *time.Time `db:"closed_at"`

		TransmissionTypes uint8   `db:"transmission_types"`
		Squawk            *string `db:"squawk"`
	}
	// SightingCallsign database record. Created for the first callsign
	// and for newly adopted callsign.
	SightingCallSign struct {
		Id         uint64    `db:"id"`
		SightingId uint64    `db:"sighting_id"`
		CallSign   string    `db:"callsign"`
		ObservedAt time.Time `db:"observed_at"`
	}
	// SightingKml database record. Created once per sighting (as it closes). May be
	// updated in future if the sighting is reopened.
	SightingKml struct {
		Id          uint64 `db:"id"`
		SightingId  uint64 `db:"sighting_id"`
		ContentType int32  `db:"content_type"`
		Kml         []byte `db:"kml"`
	}
	// SightingLocation database record. Created each time a sightings location is
	// updated.
	SightingLocation struct {
		Id         uint64    `db:"id"`
		SightingId uint64    `db:"sighting_id"`
		TimeStamp  time.Time `db:"timestamp"`
		Altitude   int64     `db:"altitude"`
		Latitude   float64   `db:"latitude"`
		Longitude  float64   `db:"longitude"`
	}
	// SightingSquawk database record. Created for the first squawk, and for
	// newly adopted squawks.
	SightingSquawk struct {
		Id         uint64    `db:"id"`
		SightingId uint64    `db:"sighting_id"`
		Squawk     string    `db:"squawk"`
		ObservedAt time.Time `db:"observed_at"`
	}
	// Email database record. Contains the encoded job, as well as information
	// relating to it's pending status. Will be deleted if successfully processed,
	// otherwise will be left in the failed state.
	Email struct {
		Id         uint64     `db:"id"`
		Status     int32      `db:"status"`
		Retries    int32      `db:"retries"`
		RetryAfter *time.Time `db:"retry_after"`
		CreatedAt  time.Time  `db:"created_at"`
		UpdatedAt  time.Time  `db:"updated_at"`
		Job        []byte
	}
)

// UpdateKml - updates the sighting_kml record
func (k *SightingKml) UpdateKml(kml []byte) error {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(kml); err != nil {
		return err
	} else if err := w.Close(); err != nil {
		return err
	}
	k.ContentType = KmlGzipContentType
	k.Kml = b.Bytes()
	return nil
}

// DecodedKml - decodes SightingKml.Kml based on SightingKml.ContentType.
// Returns the KML file, or an error if one occurred.
func (k *SightingKml) DecodedKml() ([]byte, error) {
	if k.ContentType == KmlGzipContentType {
		r, err := gzip.NewReader(bytes.NewBuffer(k.Kml))
		if err != nil {
			return nil, errors.Wrapf(err, "creating gzip reader for kml")
		}
		decomp, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, errors.Wrapf(err, "decompressing gzipped kml")
		}
		return decomp, nil
	}
	return k.Kml, nil
}

// CheckRowsUpdated verifies the sql.Result affected the expected number
// of rows. Returns an error a different number of rows was affected.
func CheckRowsUpdated(res sql.Result, expectAffected int64) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	} else if affected != expectAffected {
		return errors.Errorf("expected %d rows affected, got %d", expectAffected, affected)
	}
	return nil
}

// Database - interface for the database queries
type Database interface {
	// Transaction takes a function to execute inside a transaction context.
	// If the function returns an error, the transaction is rolled back.
	Transaction(f func(tx *sqlx.Tx) error) error

	// CreateProject creates a Project record.
	CreateProject(projectName string, now time.Time) (sql.Result, error)
	// GetProject searches for a Project by its name. If the project exists
	// it will be returned. Otherwise an error will be returned.
	GetProject(projectName string) (*Project, error)

	// CreateSession creates a new Session for a particular project.
	CreateSession(project *Project, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool) (sql.Result, error)
	// GetSessionByIdentifier searches for a Session belonging to the provided project.
	// If the Session exists it will be returned. Otherwise an error is returned.
	GetSessionByIdentifier(site *Project, identifier string) (*Session, error)
	// CloseSession marks the Session as closed. The sql.Result is returned
	// if the query was successful, otherwise an error is returned.
	CloseSession(session *Session) (sql.Result, error)

	// GetAircraftByIcao searches for an Aircraft using it's hex ICAO. If the
	// Aircraft exists it will be returned. Otherwise an error is returned.
	GetAircraftByIcao(icao string) (*Aircraft, error)
	// GetAircraftById searches for an Aircraft using it's ID. If the
	// Aircraft exists it will be returned. Otherwise an error is returned.
	GetAircraftById(id uint64) (*Aircraft, error)
	// CreateAircraft creates an Aircraft for the specified icao.
	CreateAircraft(icao string) (sql.Result, error)

	// CreateSighting creates a sighting for ac on the provided Session. A sql.Result
	// is returned if the query was successful. Otherwise an error is returned.
	CreateSighting(session *Session, ac *Aircraft) (sql.Result, error)
	// CreateSightingTx creates a sighting for ac on the provided Session, executing
	// the query with the provided transaction. A sql.Result is returned if the query
	// was successful. Otherwise an error is returned.
	CreateSightingTx(tx *sqlx.Tx, session *Session, ac *Aircraft) (sql.Result, error)
	// GetSightingById searches for a Sighting with the provided ID. The Sighting is returned
	// if one was found. Otherwise an error is returned.
	GetSightingById(sightingId uint64) (*Sighting, error)
	// GetSightingById searches for a Sighting with the provided ID, executing the query
	// with the provided transaction. The Sighting is returned if one was found. Otherwise
	// an error is returned.
	GetSightingByIdTx(tx *sqlx.Tx, sightingId uint64) (*Sighting, error)
	// GetLastSighting searches for a Sighting for ac in the provided Session. The Sighting
	// is returned if one was found. Otherwise an error is returned.
	GetLastSighting(session *Session, ac *Aircraft) (*Sighting, error)
	// GetLastSightingTx searches for a Sighting for ac in the provided Session, executing
	// the query with the provided transaction. The Sighting is returned if one was found.
	// Otherwise an error is returned.
	GetLastSightingTx(tx *sqlx.Tx, session *Session, ac *Aircraft) (*Sighting, error)
	// ReopenSighting updates the provided Sighting to mark it as open. A sql.Result
	// is returned if the query was successful. Otherwise an error is returned.
	ReopenSighting(sighting *Sighting) (sql.Result, error)
	// ReopenSightingTx updates the provided Sighting to mark it as open, executing
	// the query with the provided transaction. A sql.Result is returned if the query
	// was successful. Otherwise an error is returned.
	ReopenSightingTx(tx *sqlx.Tx, sighting *Sighting) (sql.Result, error)
	// CloseSightingBatch takes a list of sightings and marks them as closed in a batch.
	// If successful, nil is returned. Otherwise an error is returned.
	CloseSightingBatch(sightings []*Sighting) error
	// UpdateSightingCallsignTx updates the Sighting.Callsign to the provided callsign.
	// A sql.Result is returned if the query is successful. Otherwise an error is returned.
	UpdateSightingCallsignTx(tx *sqlx.Tx, sighting *Sighting, callsign string) (sql.Result, error)
	// UpdateSightingSquawkTx updates the Sighting.Squawk to the provided squawk.
	// A sql.Result is returned if the query is successful. Otherwise an error is returned.
	UpdateSightingSquawkTx(tx *sqlx.Tx, sighting *Sighting, squawk string) (sql.Result, error)

	// CreateNewSightingCallSignTx inserts a new SightingCallSign for a sighting, executing
	// the query on the provided tx. A sql.Result is returned if the query was successful.
	// Otherwise an error is returned.
	CreateNewSightingCallSignTx(tx *sqlx.Tx, sighting *Sighting, callsign string, observedAt time.Time) (sql.Result, error)
	// CreateNewSightingSquawkTx inserts a new SightingSquaqk for a sighting, executing
	// the query on the provided tx. A sql.Result is returned if the query was successful.
	// Otherwise an error is returned.
	CreateNewSightingSquawkTx(tx *sqlx.Tx, sighting *Sighting, squawk string, observedAt time.Time) (sql.Result, error)

	// CreateSightingLocation inserts a new SightingCallSign for a sighting. A sql.Result is
	// returned if the query was successful. Otherwise an error is returned.
	CreateSightingLocation(sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error)
	// CreateSightingLocationTx inserts a new SightingLocation for a sighting, executing
	// the query on the provided tx. A sql.Result is returned if the query was successful.
	// Otherwise an error is returned.
	CreateSightingLocationTx(tx *sqlx.Tx, sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error)
	// LoadLocationHistory searches for SightingLocation records for the provided Sighting.
	// lastId should initially be zero, and in subsequent calls the ID of the last processed
	// row should be used instead. At most batchSize results will be returned. If the query
	// succeeds, the rows are returned (and must be closed by the caller). If unsuccessful
	// an error is returned.
	LoadLocationHistory(sighting *Sighting, lastId int64, batchSize int64) (*sqlx.Rows, error)
	// WalkLocationHistoryBatch searches for SightingLocation records by loading at most
	// batchSize results at a time, invoking f with each batch of results. An error is returned
	// if we were unsuccessful.
	WalkLocationHistoryBatch(sighting *Sighting, batchSize int64, f func([]SightingLocation)) error
	// GetFullLocationHistory queries for all SightingLocation records. Internally, use batchSize
	// to limit the number of rows returned by each query. If successful, the full location history
	// is returned. Otherwise, an error will be returned.
	GetFullLocationHistory(sighting *Sighting, batchSize int64) ([]SightingLocation, error)

	// GetSightingKml queries for a SightingKml record for the provided Sighting. The SightingKml
	// is returned if found, otherwise an error is returned.
	GetSightingKml(sighting *Sighting) (*SightingKml, error)
	// UpdateSightingKml commits an updated KML field to the database. A sql.Result is returned
	// if the query was successful, otherwise an error is returned.
	UpdateSightingKml(sightingKml *SightingKml) (sql.Result, error)
	// CreateSightingKmlContent creates a SightingKml record for the provided Sighting. A sql.Result is
	// returned if the query was successful, otherwise an error is returned.
	CreateSightingKmlContent(sighting *Sighting, kmlData []byte) (sql.Result, error)

	// CreateEmailJobTx inserts a new Email record, executing the query on the provided tx. A
	// sql.Result is returned if the query was successful, otherwise an error is returned.
	CreateEmailJobTx(tx *sqlx.Tx, createdAt time.Time, content []byte) (sql.Result, error)
	// GetPendingEmailJobs searches for non-failed emails with a retryTime less than or equal to
	// currentTime. A list of Email records is returned if successful, otherwise an error is
	// returned.
	GetPendingEmailJobs(currentTime time.Time) ([]Email, error)
	// DeleteCompletedEmailTx deletes the specified job, excecuting the query on the provided tx.
	// A sql.Result is returned if the query was successful, otherwise an error is returned.
	DeleteCompletedEmailTx(tx *sqlx.Tx, job Email) (sql.Result, error)
	// MarkEmailFailedTx sets job's status to failed, executing the query on the provided tx.
	// A sql.Result is returned if the query was successful, otherwise an error is returned.
	MarkEmailFailedTx(tx *sqlx.Tx, job *Email) (sql.Result, error)
	// RetryEmailAfterTx updates the job records retryAfter to the provided retryAfter value.
	// A sql.Result is returned if the query was successful, otherwise an error is returned.
	RetryEmailAfterTx(tx *sqlx.Tx, job *Email, retryAfter time.Time) (sql.Result, error)
}

// DatabaseImpl - Implements Database.
type DatabaseImpl struct {
	db      *sqlx.DB
	dialect goqu.DialectWrapper
}

// NewDatabase creates a DatabaseImpl with the provided db and dialect.
func NewDatabase(db *sqlx.DB, dialect goqu.DialectWrapper) *DatabaseImpl {
	return &DatabaseImpl{db: db, dialect: dialect}
}

// Transaction - see Database.Transaction
func (d *DatabaseImpl) Transaction(f func(tx *sqlx.Tx) error) error {
	return NewTxExecer(d.db, f).Exec()
}

// CreateProject - see Database.CreateProject
func (d *DatabaseImpl) CreateProject(siteName string, now time.Time) (sql.Result, error) {
	q := d.dialect.
		Insert("project").
		Prepared(true).
		Cols("identifier", "created_at", "updated_at").
		Vals(goqu.Vals{siteName, now, now})
	s, p, err := q.ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// GetProject - see Database.GetProject
func (d *DatabaseImpl) GetProject(siteName string) (*Project, error) {
	q := d.dialect.
		From("project").
		Prepared(true).
		Where(goqu.C("identifier").Eq(siteName))
	s, p, err := q.ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	site := Project{}
	err = row.StructScan(&site)
	if err != nil {
		return nil, err
	}
	return &site, err
}

// CreateSession - see Database.CreateSession
// todo: pass in current time
func (d *DatabaseImpl) CreateSession(project *Project, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Insert("session").
		Prepared(true).
		Cols("project_id", "identifier", "with_squawks", "with_transmission_types",
			"with_callsigns", "created_at", "updated_at", "closed_at").
		Vals(goqu.Vals{project.Id, identifier, withSquawks, withTxTypes, withCallSigns, now, now, nil}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetSessionByIdentifier - see Database.GetSessionByIdentifier
func (d *DatabaseImpl) GetSessionByIdentifier(site *Project, identifier string) (*Session, error) {
	s, p, err := d.dialect.
		From("session").
		Prepared(true).
		Where(goqu.Ex{
			"project_id": site.Id,
			"identifier": identifier,
		}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	session := &Session{}
	err = row.StructScan(session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// CloseSession - see Database.CloseSession
// todo: pass in time.Now()
func (d *DatabaseImpl) CloseSession(session *Session) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Update("session").
		Prepared(true).
		Set(goqu.Ex{
			"closed_at": now,
		}).
		Where(goqu.C("id").Eq(session.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	session.ClosedAt = &now
	return res, nil
}

// GetAircraftByIcao - see Database.GetAircraftByIcao
func (d *DatabaseImpl) GetAircraftByIcao(icao string) (*Aircraft, error) {
	s, p, err := d.dialect.
		From("aircraft").
		Prepared(true).
		Where(goqu.C("icao").Eq(icao)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	aircraft := &Aircraft{}
	err = row.StructScan(aircraft)
	if err != nil {
		return nil, err
	}
	return aircraft, nil
}

// GetAircraftById - see Database.GetAircraftById
func (d *DatabaseImpl) GetAircraftById(id uint64) (*Aircraft, error) {
	s, p, err := d.dialect.
		From("aircraft").
		Prepared(true).
		Where(goqu.C("id").Eq(id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	aircraft := &Aircraft{}
	err = row.StructScan(aircraft)
	if err != nil {
		return nil, err
	}
	return aircraft, nil
}

// CreateAircraft - see Database.CreateAircraft
// todo: pass in current time
func (d *DatabaseImpl) CreateAircraft(icao string) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Insert("aircraft").
		Prepared(true).
		Cols("icao", "created_at", "updated_at").
		Vals(goqu.Vals{icao, now, now}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// CreateSighting - see Database.CreateSighting
// todo: pass in current time
func (d *DatabaseImpl) CreateSighting(session *Session, ac *Aircraft) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Insert("sighting").
		Prepared(true).
		Cols("project_id", "session_id", "aircraft_id", "created_at", "updated_at").
		Vals(goqu.Vals{session.ProjectId, session.Id, ac.Id, &now, &now}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// CreateSightingTx - see Database.CreateSightingTx
// todo: pass in current time
func (d *DatabaseImpl) CreateSightingTx(tx *sqlx.Tx, session *Session, ac *Aircraft) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Insert("sighting").
		Prepared(true).
		Cols("project_id", "session_id", "aircraft_id", "created_at", "updated_at").
		Vals(goqu.Vals{session.ProjectId, session.Id, ac.Id, &now, &now}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

// ReopenSighting - see Database.ReopenSighting
func (d *DatabaseImpl) ReopenSighting(sighting *Sighting) (sql.Result, error) {
	s, p, err := d.dialect.
		Update("session").
		Prepared(true).
		Set(goqu.Ex{
			"closed_at": nil,
		}).
		Where(goqu.C("id").Eq(sighting.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	sighting.ClosedAt = nil
	return res, nil
}

// ReopenSightingTx - see Database.ReopenSightingTx
func (d *DatabaseImpl) ReopenSightingTx(tx *sqlx.Tx, sighting *Sighting) (sql.Result, error) {
	s, p, err := d.dialect.
		Update("session").
		Prepared(true).
		Set(goqu.Ex{
			"closed_at": nil,
		}).
		Where(goqu.C("id").Eq(sighting.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	sighting.ClosedAt = nil
	return res, nil
}

// GetLastSighting - see Database.GetLastSighting
func (d *DatabaseImpl) GetLastSighting(session *Session, ac *Aircraft) (*Sighting, error) {
	s, p, err := d.dialect.
		From("sighting").
		Prepared(true).
		Where(goqu.Ex{
			"session_id":  session.Id,
			"aircraft_id": ac.Id,
		}).
		Order(goqu.C("id").Desc()).
		Limit(1).
		ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	sighting := &Sighting{}
	err = row.StructScan(sighting)
	if err != nil {
		return nil, err
	}
	return sighting, nil
}

// GetLastSightingTx - see Database.GetLastSightingTx
func (d *DatabaseImpl) GetLastSightingTx(tx *sqlx.Tx, session *Session, ac *Aircraft) (*Sighting, error) {
	s, p, err := d.dialect.
		From("sighting").
		Prepared(true).
		Where(goqu.Ex{
			"session_id":  session.Id,
			"aircraft_id": ac.Id,
		}).
		Order(goqu.C("id").Desc()).
		Limit(1).
		ToSQL()
	if err != nil {
		return nil, err
	}
	row := tx.QueryRowx(s, p...)
	sighting := &Sighting{}
	err = row.StructScan(sighting)
	if err != nil {
		return nil, err
	}
	return sighting, nil
}

// UpdateSightingCallsignTx - see Database.UpdateSightingCallsignTx
func (d *DatabaseImpl) UpdateSightingCallsignTx(tx *sqlx.Tx, sighting *Sighting, callsign string) (sql.Result, error) {
	s, p, err := d.dialect.
		Update("sighting").
		Prepared(true).
		Set(goqu.Ex{
			"callsign": callsign,
		}).
		Where(goqu.C("id").Eq(sighting.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	sighting.CallSign = &callsign
	return res, nil
}

// UpdateSightingSquawkTx - see Database.UpdateSightingSquawkTx
func (d *DatabaseImpl) UpdateSightingSquawkTx(tx *sqlx.Tx, sighting *Sighting, squawk string) (sql.Result, error) {
	s, p, err := d.dialect.
		Update("sighting").
		Prepared(true).
		Set(goqu.Ex{
			"squawk": squawk,
		}).
		Where(goqu.C("id").Eq(sighting.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	sighting.Squawk = &squawk
	return res, nil
}

// CloseSightingBatch - see Database.CloseSightingBatch
func (d *DatabaseImpl) CloseSightingBatch(sightings []*Sighting) error {
	if len(sightings) == 0 {
		return nil
	}
	n := len(sightings)
	closedAt := time.Now()
	err := d.Transaction(func(tx *sqlx.Tx) error {
		var ids []uint64
		for i := 0; i < n; i++ {
			ids = append(ids, sightings[i].Id)
		}

		s, p, err := d.dialect.
			Update("sighting").
			Prepared(true).
			Set(goqu.Ex{
				"closed_at": closedAt,
			}).
			Where(goqu.C("id").In(ids)).
			ToSQL()
		if err != nil {
			return err
		}
		res, err := tx.Exec(s, p...)
		if err != nil {
			return err
		}
		return CheckRowsUpdated(res, int64(n))
	})
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		sightings[i].ClosedAt = &closedAt
	}
	return nil
}

// GetSightingById - see Database.GetSightingById
func (d *DatabaseImpl) GetSightingById(sightingId uint64) (*Sighting, error) {
	s, p, err := d.dialect.
		From("sighting").
		Prepared(true).
		Where(goqu.C("id").Eq(sightingId)).
		Limit(1).
		ToSQL()
	if err != nil {
		return nil, err
	}
	// index: aircraft_id, session_id
	row := d.db.QueryRowx(s, p...)
	sighting := &Sighting{}
	err = row.StructScan(sighting)
	if err != nil {
		return nil, err
	}
	return sighting, nil
}

// GetSightingByIdTx - see Database.GetSightingByIdTx
func (d *DatabaseImpl) GetSightingByIdTx(tx *sqlx.Tx, sightingId uint64) (*Sighting, error) {
	s, p, err := d.dialect.
		From("sighting").
		Prepared(true).
		Where(goqu.C("id").Eq(sightingId)).
		Limit(1).
		ToSQL()
	if err != nil {
		return nil, err
	}
	// index: aircraft_id, session_id
	row := tx.QueryRowx(s, p...)
	sighting := &Sighting{}
	err = row.StructScan(sighting)
	if err != nil {
		return nil, err
	}
	return sighting, nil
}

// CreateNewSightingCallSignTx - see Database.CreateNewSightingCallSignTx
func (d *DatabaseImpl) CreateNewSightingCallSignTx(tx *sqlx.Tx, sighting *Sighting, callsign string, observedAt time.Time) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert("sighting_callsign").
		Prepared(true).
		Cols("sighting_id", "callsign", "observed_at").
		Vals(goqu.Vals{sighting.Id, callsign, observedAt}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

// CreateSightingLocation - see Database.CreateSightingLocation
func (d *DatabaseImpl) CreateSightingLocation(sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert("sighting_location").
		Prepared(true).
		Cols("sighting_id", "timestamp", "altitude", "latitude", "longitude").
		Vals(goqu.Vals{sightingId, t, altitude, lat, long}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	// index: aircraft_id, session_id
	return d.db.Exec(s, p...)
}

// CreateSightingLocationTx - see Database.CreateSightingLocationTx
func (d *DatabaseImpl) CreateSightingLocationTx(tx *sqlx.Tx, sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert("sighting_location").
		Prepared(true).
		Cols("sighting_id", "timestamp", "altitude", "latitude", "longitude").
		Vals(goqu.Vals{sightingId, t, altitude, lat, long}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	// index: aircraft_id, session_id
	return tx.Exec(s, p...)
}

// LoadLocationHistory - see Database.LoadLocationHistory
func (d *DatabaseImpl) LoadLocationHistory(sighting *Sighting, lastId int64, batchSize int64) (*sqlx.Rows, error) {
	s, p, err := d.dialect.
		From("sighting_location").
		Prepared(true).
		Where(goqu.Ex{
			"sighting_id": sighting.Id,
			"id":          goqu.Op{"gt": lastId},
		}).
		Limit(uint(batchSize)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := d.db.Queryx(s, p...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// WalkLocationHistoryBatch - see Database.WalkLocationHistoryBatch
func (d *DatabaseImpl) WalkLocationHistoryBatch(sighting *Sighting, batchSize int64, f func([]SightingLocation)) error {
	lastId := int64(-1)
	batch := make([]SightingLocation, 0, batchSize)
	for {
		// res - needs closing
		res, err := d.LoadLocationHistory(sighting, lastId, batchSize)
		if err != nil {
			return errors.Wrap(err, "failed to fetch location history")
		}

		for res.Next() {
			location := SightingLocation{}
			err = res.StructScan(&location)
			if err != nil {
				_ = res.Close()
				return errors.Wrap(err, "scanning location record into memory")
			}
			batch = append(batch, location)
		}

		_ = res.Close()
		if len(batch) == 0 {
			return nil
		}

		f(batch)
		lastId = int64(batch[len(batch)-1].Id)
		batch = make([]SightingLocation, 0, batchSize)
	}
}

// GetFullLocationHistory - see Database.GetFullLocationHistory
func (d *DatabaseImpl) GetFullLocationHistory(sighting *Sighting, batchSize int64) ([]SightingLocation, error) {
	var h []SightingLocation
	err := d.WalkLocationHistoryBatch(sighting, batchSize, func(location []SightingLocation) {
		h = append(h, location...)
	})
	if err != nil {
		return nil, err
	}
	return h, nil
}

// CreateNewSightingSquawkTx - see Database.CreateNewSightingSquawkTx
func (d *DatabaseImpl) CreateNewSightingSquawkTx(tx *sqlx.Tx, sighting *Sighting, squawk string, observedAt time.Time) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert("sighting_squawk").
		Prepared(true).
		Cols("sighting_id", "squawk", "observed_at").
		Vals(goqu.Vals{sighting.Id, squawk, observedAt}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

// GetSightingKml - see Database.GetSightingKml
func (d *DatabaseImpl) GetSightingKml(sighting *Sighting) (*SightingKml, error) {
	s, p, err := d.dialect.
		From("sighting_kml").
		Prepared(true).
		Where(goqu.Ex{
			"sighting_id": sighting.Id,
		}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	sightingKml := &SightingKml{}
	err = row.StructScan(sightingKml)
	if err != nil {
		return nil, err
	}
	return sightingKml, nil
}

// UpdateSightingKml - see Database.UpdateSightingKml
func (d *DatabaseImpl) UpdateSightingKml(sightingKml *SightingKml) (sql.Result, error) {
	s, p, err := d.dialect.
		Update("sighting_kml").
		Prepared(true).
		Set(goqu.Ex{
			"kml":          sightingKml.Kml,
			"content_type": sightingKml.ContentType,
		}).
		Where(goqu.C("id").Eq(sightingKml.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	} else if err = CheckRowsUpdated(res, 1); err != nil {
		return nil, err
	}
	return res, nil
}

// CreateSightingKmlContent - see Database.CreateSightingKmlContent
func (d *DatabaseImpl) CreateSightingKmlContent(sighting *Sighting, kmlData []byte) (sql.Result, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(kmlData)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}

	s, p, err := d.dialect.
		Insert("sighting_kml").
		Prepared(true).
		Cols("sighting_id", "content_type", "kml").
		Vals(goqu.Vals{sighting.Id, KmlGzipContentType, b.Bytes()}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// CreateEmailJobTx - see Database.CreateEmailJobTx
func (d *DatabaseImpl) CreateEmailJobTx(tx *sqlx.Tx, createdAt time.Time, content []byte) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert("email").
		Prepared(true).
		Cols("status", "retries", "created_at", "updated_at", "retry_after", "job").
		Vals(goqu.Vals{EmailPending, 0, createdAt, createdAt, nil, content}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

// GetPendingEmailJobs - see Database.GetPendingEmailJobs
// Does not return sql.ErrNoRows
func (d *DatabaseImpl) GetPendingEmailJobs(now time.Time) ([]Email, error) {
	s, p, err := d.dialect.
		From("email").
		Prepared(true).
		Where(goqu.C("status").Eq(EmailPending)).
		Where(goqu.Or(
			goqu.C("retry_after").Eq(nil),
			goqu.C("retry_after").Lte(now))).
		Order(goqu.C("id").Desc()).
		ToSQL()
	if err != nil {
		return nil, err
	}

	var jobs []Email
	rows, err := d.db.Queryx(s, p...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		job := Email{}
		err := rows.StructScan(&job)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// DeleteCompletedEmailTx - see Database.DeleteCompletedEmailTx
func (d *DatabaseImpl) DeleteCompletedEmailTx(tx *sqlx.Tx, job Email) (sql.Result, error) {
	s, p, err := d.dialect.
		Delete("email").
		Prepared(true).
		Where(goqu.C("id").Eq(job.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

// MarkEmailFailedTx - see Database.MarkEmailFailedTx
func (d *DatabaseImpl) MarkEmailFailedTx(tx *sqlx.Tx, job *Email) (sql.Result, error) {
	s, p, err := d.dialect.
		Update("email").
		Prepared(true).
		Set(goqu.Ex{
			"retry_after": 0,
			"status":      EmailFailed,
		}).
		Where(goqu.C("id").Eq(job.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	ts := time.Unix(0, 0)
	job.RetryAfter = &ts
	job.Status = EmailFailed
	return res, nil
}

// RetryEmailAfterTx - see Database.RetryEmailAfterTx
func (d *DatabaseImpl) RetryEmailAfterTx(tx *sqlx.Tx, job *Email, retryAfter time.Time) (sql.Result, error) {
	s, p, err := d.dialect.
		Update("email").
		Prepared(true).
		Set(goqu.Ex{
			"retry_after": retryAfter,
			"retries":     job.Retries + 1,
		}).
		Where(goqu.C("id").Eq(job.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	job.RetryAfter = &retryAfter
	job.Retries = job.Retries + 1
	return res, nil
}
