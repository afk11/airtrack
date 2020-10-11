package db

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"github.com/doug-martin/goqu/v9"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"io/ioutil"
	"time"
)

const (
	// EmailPending - status of a pending email
	EmailPending = 1
	// EmailFailed - status of a failed email
	EmailFailed = 0

	// KmlPlainTextContentType - content type used when
	// sighting_kml kml record is encoded in plain text KML
	KmlPlainTextContentType = 0
	// KmlGzipContentType - content type used when
	// sighting_kml kml record is gzipped KML
	KmlGzipContentType = 1

	projectTable          = "project"
	sessionTable          = "session"
	aircraftTable         = "aircraft"
	sightingTable         = "sighting"
	sightingLocationTable = "sighting_location"
	sightingCallsignTable = "sighting_callsign"
	sightingSquawkTable   = "sighting_squawk"
	sightingKmlTable      = "sighting_kml"
	emailTable            = "email"
	schemaMigrationsTable = "schema_migrations"
)

type (
	// Project database record. Created the first time
	// a project is used.
	Project struct {
		ID         uint64    `db:"id"`
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
		ID                    uint64     `db:"id"`
		Identifier            string     `db:"identifier"`
		ProjectID             uint64     `db:"project_id"`
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
		ID        uint64    `db:"id"`
		Icao      string    `db:"icao"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	// Sighting database record. Created every time an aircraft is
	// watched by a project in a session.
	Sighting struct {
		ID         uint64  `db:"id"`
		ProjectID  uint64  `db:"project_id"`
		SessionID  uint64  `db:"session_id"`
		AircraftID uint64  `db:"aircraft_id"`
		CallSign   *string `db:"callsign"`

		CreatedAt time.Time  `db:"created_at"`
		UpdatedAt time.Time  `db:"updated_at"`
		ClosedAt  *time.Time `db:"closed_at"`

		TransmissionTypes uint8   `db:"transmission_types"`
		Squawk            *string `db:"squawk"`
	}
	// SightingCallSign database record. Created for the first callsign
	// and for newly adopted callsign.
	SightingCallSign struct {
		ID         uint64    `db:"id"`
		SightingID uint64    `db:"sighting_id"`
		CallSign   string    `db:"callsign"`
		ObservedAt time.Time `db:"observed_at"`
	}
	// SightingKml database record. Created once per sighting (as it closes). May be
	// updated in future if the sighting is reopened.
	SightingKml struct {
		ID          uint64 `db:"id"`
		SightingID  uint64 `db:"sighting_id"`
		ContentType int32  `db:"content_type"`
		Kml         []byte `db:"kml"`
	}
	// SightingLocation database record. Created each time a sightings location is
	// updated.
	SightingLocation struct {
		ID         uint64    `db:"id"`
		SightingID uint64    `db:"sighting_id"`
		TimeStamp  time.Time `db:"timestamp"`
		Altitude   int64     `db:"altitude"`
		Latitude   float64   `db:"latitude"`
		Longitude  float64   `db:"longitude"`
	}
	// SightingSquawk database record. Created for the first squawk, and for
	// newly adopted squawks.
	SightingSquawk struct {
		ID         uint64    `db:"id"`
		SightingID uint64    `db:"sighting_id"`
		Squawk     string    `db:"squawk"`
		ObservedAt time.Time `db:"observed_at"`
	}
	// Email database record. Contains the encoded job, as well as information
	// relating to it's pending status. Will be deleted if successfully processed,
	// otherwise will be left in the failed state.
	Email struct {
		ID         uint64     `db:"id"`
		Status     int32      `db:"status"`
		Retries    int32      `db:"retries"`
		RetryAfter *time.Time `db:"retry_after"`
		CreatedAt  time.Time  `db:"created_at"`
		UpdatedAt  time.Time  `db:"updated_at"`
		Job        []byte
	}
	// SchemaMigrations database record. Contains information
	// about state of database migrations.
	SchemaMigrations struct {
		Version uint64 `db:"version"`
		Dirty   bool   `db:"dirty"`
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

// IsUniqueConstraintViolation returns whether the query failed
// due to violating a unique constraint. It returns an error
// if err is not one of the supported SQL driver error types.
func IsUniqueConstraintViolation(err error) (bool, error) {
	switch e := err.(type) {
	case *mysql.MySQLError:
		return e.Number == 1062, nil
	case *pq.Error:
		return e.Code == "23505", nil
	case sqlite3.Error:
		return e.Code == sqlite3.ErrConstraint, nil
	default:
		return false, errors.New("unsupported error type")
	}
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
	GetSessionByIdentifier(project *Project, identifier string) (*Session, error)
	// CloseSession marks the Session as closed. The sql.Result is returned
	// if the query was successful, otherwise an error is returned.
	CloseSession(session *Session, closedAt time.Time) (sql.Result, error)

	// GetAircraftByIcao searches for an Aircraft using it's hex ICAO. If the
	// Aircraft exists it will be returned. Otherwise an error is returned.
	GetAircraftByIcao(icao string) (*Aircraft, error)
	// GetAircraftById searches for an Aircraft using it's ID. If the
	// Aircraft exists it will be returned. Otherwise an error is returned.
	GetAircraftByID(id uint64) (*Aircraft, error)
	// CreateAircraft creates an Aircraft for the specified icao.
	CreateAircraft(icao string, firstSeenTime time.Time) (sql.Result, error)

	// CreateSighting creates a sighting for ac on the provided Session. A sql.Result
	// is returned if the query was successful. Otherwise an error is returned.
	CreateSighting(session *Session, ac *Aircraft, firstSeenTime time.Time) (sql.Result, error)
	// CreateSightingTx creates a sighting for ac on the provided Session, executing
	// the query with the provided transaction. A sql.Result is returned if the query
	// was successful. Otherwise an error is returned.
	CreateSightingTx(tx *sqlx.Tx, session *Session, ac *Aircraft, firstSeenTime time.Time) (sql.Result, error)
	// GetSightingById searches for a Sighting with the provided ID. The Sighting is returned
	// if one was found. Otherwise an error is returned.
	GetSightingByID(sightingID uint64) (*Sighting, error)
	// GetSightingById searches for a Sighting with the provided ID, executing the query
	// with the provided transaction. The Sighting is returned if one was found. Otherwise
	// an error is returned.
	GetSightingByIDTx(tx *sqlx.Tx, sightingID uint64) (*Sighting, error)
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
	CloseSightingBatch(sightings []*Sighting, closedAt time.Time) error
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
	CreateSightingLocation(sightingID uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error)
	// CreateSightingLocationTx inserts a new SightingLocation for a sighting, executing
	// the query on the provided tx. A sql.Result is returned if the query was successful.
	// Otherwise an error is returned.
	CreateSightingLocationTx(tx *sqlx.Tx, sightingID uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error)
	// LoadLocationHistory searches for SightingLocation records for the provided Sighting.
	// lastID should initially be zero, and in subsequent calls the ID of the last processed
	// row should be used instead. At most batchSize results will be returned. If the query
	// succeeds, the rows are returned (and must be closed by the caller). If unsuccessful
	// an error is returned.
	LoadLocationHistory(sighting *Sighting, lastID int64, batchSize int64) (*sqlx.Rows, error)
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
	// DeleteCompletedEmailTx deletes the specified job, executing the query on the provided tx.
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
func (d *DatabaseImpl) CreateProject(name string, now time.Time) (sql.Result, error) {
	q := d.dialect.
		Insert(projectTable).
		Prepared(true).
		Cols("identifier", "created_at", "updated_at").
		Vals(goqu.Vals{name, now, now})
	s, p, err := q.ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// GetProject - see Database.GetProject
func (d *DatabaseImpl) GetProject(name string) (*Project, error) {
	q := d.dialect.
		From(projectTable).
		Prepared(true).
		Where(goqu.C("identifier").Eq(name))
	s, p, err := q.ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	project := Project{}
	err = row.StructScan(&project)
	if err != nil {
		return nil, err
	}
	return &project, err
}

// CreateSession - see Database.CreateSession
func (d *DatabaseImpl) CreateSession(project *Project, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Insert(sessionTable).
		Prepared(true).
		Cols("project_id", "identifier", "with_squawks", "with_transmission_types",
			"with_callsigns", "created_at", "updated_at", "closed_at").
		Vals(goqu.Vals{project.ID, identifier, withSquawks, withTxTypes, withCallSigns, now, now, nil}).
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
func (d *DatabaseImpl) GetSessionByIdentifier(project *Project, identifier string) (*Session, error) {
	s, p, err := d.dialect.
		From(sessionTable).
		Prepared(true).
		Where(goqu.Ex{
			"project_id": project.ID,
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
func (d *DatabaseImpl) CloseSession(session *Session, closedAt time.Time) (sql.Result, error) {
	s, p, err := d.dialect.
		Update(sessionTable).
		Prepared(true).
		Set(goqu.Ex{
			"closed_at": closedAt,
		}).
		Where(goqu.C("id").Eq(session.ID)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	session.ClosedAt = &closedAt
	return res, nil
}

// GetAircraftByIcao - see Database.GetAircraftByIcao
func (d *DatabaseImpl) GetAircraftByIcao(icao string) (*Aircraft, error) {
	s, p, err := d.dialect.
		From(aircraftTable).
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

// GetAircraftByID - see Database.GetAircraftByID
func (d *DatabaseImpl) GetAircraftByID(id uint64) (*Aircraft, error) {
	s, p, err := d.dialect.
		From(aircraftTable).
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
func (d *DatabaseImpl) CreateAircraft(icao string, firstSeenTime time.Time) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert(aircraftTable).
		Prepared(true).
		Cols("icao", "created_at", "updated_at").
		Vals(goqu.Vals{icao, firstSeenTime, firstSeenTime}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// CreateSighting - see Database.CreateSighting
func (d *DatabaseImpl) CreateSighting(session *Session, ac *Aircraft, firstSeenTime time.Time) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert(sightingTable).
		Prepared(true).
		Cols("project_id", "session_id", "aircraft_id", "created_at", "updated_at").
		Vals(goqu.Vals{session.ProjectID, session.ID, ac.ID, &firstSeenTime, &firstSeenTime}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// CreateSightingTx - see Database.CreateSightingTx
func (d *DatabaseImpl) CreateSightingTx(tx *sqlx.Tx, session *Session, ac *Aircraft, firstSeenTime time.Time) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert(sightingTable).
		Prepared(true).
		Cols("project_id", "session_id", "aircraft_id", "created_at", "updated_at").
		Vals(goqu.Vals{session.ProjectID, session.ID, ac.ID, &firstSeenTime, &firstSeenTime}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

// ReopenSighting - see Database.ReopenSighting
func (d *DatabaseImpl) ReopenSighting(sighting *Sighting) (sql.Result, error) {
	s, p, err := d.dialect.
		Update(sessionTable).
		Prepared(true).
		Set(goqu.Ex{
			"closed_at": nil,
		}).
		Where(goqu.C("id").Eq(sighting.ID)).
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
		Update(sessionTable).
		Prepared(true).
		Set(goqu.Ex{
			"closed_at": nil,
		}).
		Where(goqu.C("id").Eq(sighting.ID)).
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
		From(sightingTable).
		Prepared(true).
		Where(goqu.Ex{
			"session_id":  session.ID,
			"aircraft_id": ac.ID,
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
		From(sightingTable).
		Prepared(true).
		Where(goqu.Ex{
			"session_id":  session.ID,
			"aircraft_id": ac.ID,
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
		Update(sightingTable).
		Prepared(true).
		Set(goqu.Ex{
			"callsign": callsign,
		}).
		Where(goqu.C("id").Eq(sighting.ID)).
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
		Update(sightingTable).
		Prepared(true).
		Set(goqu.Ex{
			"squawk": squawk,
		}).
		Where(goqu.C("id").Eq(sighting.ID)).
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
func (d *DatabaseImpl) CloseSightingBatch(sightings []*Sighting, closedAt time.Time) error {
	if len(sightings) == 0 {
		return nil
	}
	n := len(sightings)
	err := d.Transaction(func(tx *sqlx.Tx) error {
		var ids []uint64
		for i := 0; i < n; i++ {
			ids = append(ids, sightings[i].ID)
		}

		s, p, err := d.dialect.
			Update(sightingTable).
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

// GetSchemaMigration - see Database.GetSchemaMigration
func (d *DatabaseImpl) GetSchemaMigration() (*SchemaMigrations, error) {
	s, p, err := d.dialect.
		From(schemaMigrationsTable).
		Prepared(true).
		ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	sm := &SchemaMigrations{}
	err = row.StructScan(sm)
	if err != nil {
		return nil, err
	}
	return sm, nil
}

// GetSightingByID - see Database.GetSightingByID
func (d *DatabaseImpl) GetSightingByID(sightingID uint64) (*Sighting, error) {
	s, p, err := d.dialect.
		From(sightingTable).
		Prepared(true).
		Where(goqu.C("id").Eq(sightingID)).
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

// GetSightingByIDTx - see Database.GetSightingByIDTx
func (d *DatabaseImpl) GetSightingByIDTx(tx *sqlx.Tx, sightingID uint64) (*Sighting, error) {
	s, p, err := d.dialect.
		From(sightingTable).
		Prepared(true).
		Where(goqu.C("id").Eq(sightingID)).
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
		Insert(sightingCallsignTable).
		Prepared(true).
		Cols("sighting_id", "callsign", "observed_at").
		Vals(goqu.Vals{sighting.ID, callsign, observedAt}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

// GetLastSightingCallSignTx - see Database.GetLastSightingCallSignTx
func (d *DatabaseImpl) GetLastSightingCallSignTx(tx *sqlx.Tx, sighting *Sighting) (*SightingCallSign, error) {
	s, p, err := d.dialect.
		From(sightingCallsignTable).
		Prepared(true).
		Where(goqu.C("sighting_id").Eq(sighting.ID)).
		Order(goqu.C("id").Desc()).
		Limit(1).
		ToSQL()
	if err != nil {
		return nil, err
	}

	callsign := SightingCallSign{}
	row := tx.QueryRowx(s, p...)
	if err = row.StructScan(&callsign); err != nil {
		return nil, err
	}
	return &callsign, nil
}

// CreateSightingLocation - see Database.CreateSightingLocation
func (d *DatabaseImpl) CreateSightingLocation(sightingID uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert(sightingLocationTable).
		Prepared(true).
		Cols("sighting_id", "timestamp", "altitude", "latitude", "longitude").
		Vals(goqu.Vals{sightingID, t, altitude, lat, long}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	// index: aircraft_id, session_id
	return d.db.Exec(s, p...)
}

// CreateSightingLocationTx - see Database.CreateSightingLocationTx
func (d *DatabaseImpl) CreateSightingLocationTx(tx *sqlx.Tx, sightingID uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert(sightingLocationTable).
		Prepared(true).
		Cols("sighting_id", "timestamp", "altitude", "latitude", "longitude").
		Vals(goqu.Vals{sightingID, t, altitude, lat, long}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	// index: aircraft_id, session_id
	return tx.Exec(s, p...)
}

// LoadLocationHistory - see Database.LoadLocationHistory
func (d *DatabaseImpl) LoadLocationHistory(sighting *Sighting, lastID int64, batchSize int64) (*sqlx.Rows, error) {
	s, p, err := d.dialect.
		From(sightingLocationTable).
		Prepared(true).
		Where(goqu.Ex{
			"sighting_id": sighting.ID,
			"id":          goqu.Op{"gt": lastID},
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
	lastID := int64(-1)
	batch := make([]SightingLocation, 0, batchSize)
	for {
		// res - needs closing
		res, err := d.LoadLocationHistory(sighting, lastID, batchSize)
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
		lastID = int64(batch[len(batch)-1].ID)
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
		Insert(sightingSquawkTable).
		Prepared(true).
		Cols("sighting_id", "squawk", "observed_at").
		Vals(goqu.Vals{sighting.ID, squawk, observedAt}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

// GetLastSightingSquawkTx - see Database.GetLastSightingSquawkTx
func (d *DatabaseImpl) GetLastSightingSquawkTx(tx *sqlx.Tx, sighting *Sighting) (*SightingSquawk, error) {
	s, p, err := d.dialect.
		From(sightingSquawkTable).
		Prepared(true).
		Where(goqu.C("sighting_id").Eq(sighting.ID)).
		Order(goqu.C("id").Desc()).
		Limit(1).
		ToSQL()
	if err != nil {
		return nil, err
	}

	squawk := SightingSquawk{}
	row := tx.QueryRowx(s, p...)
	if err = row.StructScan(&squawk); err != nil {
		return nil, err
	}
	return &squawk, nil
}

// GetSightingKml - see Database.GetSightingKml
func (d *DatabaseImpl) GetSightingKml(sighting *Sighting) (*SightingKml, error) {
	s, p, err := d.dialect.
		From(sightingKmlTable).
		Prepared(true).
		Where(goqu.Ex{
			"sighting_id": sighting.ID,
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
		Update(sightingKmlTable).
		Prepared(true).
		Set(goqu.Ex{
			"kml":          sightingKml.Kml,
			"content_type": sightingKml.ContentType,
		}).
		Where(goqu.C("id").Eq(sightingKml.ID)).
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
		Insert(sightingKmlTable).
		Prepared(true).
		Cols("sighting_id", "content_type", "kml").
		Vals(goqu.Vals{sighting.ID, KmlGzipContentType, b.Bytes()}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// CreateEmailJobTx - see Database.CreateEmailJobTx
func (d *DatabaseImpl) CreateEmailJobTx(tx *sqlx.Tx, createdAt time.Time, content []byte) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert(emailTable).
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
		From(emailTable).
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
		Delete(emailTable).
		Prepared(true).
		Where(goqu.C("id").Eq(job.ID)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

// MarkEmailFailedTx - see Database.MarkEmailFailedTx
func (d *DatabaseImpl) MarkEmailFailedTx(tx *sqlx.Tx, job *Email) (sql.Result, error) {
	s, p, err := d.dialect.
		Update(emailTable).
		Prepared(true).
		Set(goqu.Ex{
			"status": EmailFailed,
		}).
		Where(goqu.C("id").Eq(job.ID)).
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
		Update(emailTable).
		Prepared(true).
		Set(goqu.Ex{
			"retry_after": retryAfter,
			"retries":     job.Retries + 1,
		}).
		Where(goqu.C("id").Eq(job.ID)).
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
