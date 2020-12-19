package db

import (
	"database/sql"
	"github.com/doug-martin/goqu/v9"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"time"
)

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

// Database - wraps a DB handle. Combines Queries with methods
// associated with the DB handle such as transactions.
type Database interface {
	// Transaction takes a func f to execute inside a transaction context.
	// f receives a new Queries implementation initialized for the transaction.
	// If f returns an error, the transaction is rolled back. Otherwise when
	// f completes, the transaction is committed. An error will be returned
	// if an error occurs at any time, otherwise nil if successful.
	Transaction(f func(tx Queries) error) error

	Queries
}

// Queries - interface exposing database IO operations.
type Queries interface {
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
	// GetSightingByID searches for a Sighting with the provided ID. The Sighting is returned
	// if one was found. Otherwise an error is returned.
	GetSightingByID(sightingID uint64) (*Sighting, error)
	// GetLastSighting searches for a Sighting for ac in the provided Session. The Sighting
	// is returned if one was found. Otherwise an error is returned.
	GetLastSighting(session *Session, ac *Aircraft) (*Sighting, error)
	// ReopenSighting updates the provided Sighting to mark it as open. A sql.Result
	// is returned if the query was successful. Otherwise an error is returned.
	ReopenSighting(sighting *Sighting) (sql.Result, error)
	// CloseSightingBatch takes a list of sightings and marks them as closed in a batch.
	// If successful, nil is returned. Otherwise an error is returned.
	CloseSightingBatch(sightings []*Sighting, closedAt time.Time) error
	// UpdateSightingCallsign updates the Sighting.Callsign to the provided callsign.
	// A sql.Result is returned if the query is successful. Otherwise an error is returned.
	UpdateSightingCallsign(sighting *Sighting, callsign string) (sql.Result, error)
	// UpdateSightingSquawk updates the Sighting.Squawk to the provided squawk.
	// A sql.Result is returned if the query is successful. Otherwise an error is returned.
	UpdateSightingSquawk(sighting *Sighting, squawk string) (sql.Result, error)

	// CreateNewSightingCallSign inserts a new SightingCallSign for a sighting, executing
	// the query on the provided tx. A sql.Result is returned if the query was successful.
	// Otherwise an error is returned.
	CreateNewSightingCallSign(sighting *Sighting, callsign string, observedAt time.Time) (sql.Result, error)
	// CreateNewSightingSquawk inserts a new SightingSquaqk for a sighting, executing
	// the query on the provided tx. A sql.Result is returned if the query was successful.
	// Otherwise an error is returned.
	CreateNewSightingSquawk(sighting *Sighting, squawk string, observedAt time.Time) (sql.Result, error)

	// CreateSightingLocation inserts a new SightingCallSign for a sighting. A sql.Result is
	// returned if the query was successful. Otherwise an error is returned.
	CreateSightingLocation(sightingID uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error)

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

	// GetLastSightingSquawk searches for the most recent SightingSquawk for this sighting,
	// otherwise an error is returned.
	GetLastSightingSquawk(sighting *Sighting) (*SightingSquawk, error)
	// GetLastSightingCallSign searches for the most recent SightingCallSign for this sighting,
	// otherwise an error is returned.
	GetLastSightingCallSign(sighting *Sighting) (*SightingCallSign, error)

	// CreateEmailJob inserts a new Email record, executing the query on the provided tx. A
	// sql.Result is returned if the query was successful, otherwise an error is returned.
	CreateEmailJob(createdAt time.Time, content []byte) (sql.Result, error)
	// GetPendingEmailJobs searches for non-failed emails with a retryTime less than or equal to
	// currentTime. A list of Email records is returned if successful, otherwise an error is
	// returned.
	GetPendingEmailJobs(currentTime time.Time) ([]Email, error)
	// DeleteCompletedEmail deletes the specified job, executing the query on the provided tx.
	// A sql.Result is returned if the query was successful, otherwise an error is returned.
	DeleteCompletedEmail(job Email) (sql.Result, error)
	// MarkEmailFailed sets job's status to failed, executing the query on the provided tx.
	// A sql.Result is returned if the query was successful, otherwise an error is returned.
	MarkEmailFailed(job *Email) (sql.Result, error)
	// RetryEmailAfter updates the job records retryAfter to the provided retryAfter value.
	// A sql.Result is returned if the query was successful, otherwise an error is returned.
	RetryEmailAfter(job *Email, retryAfter time.Time) (sql.Result, error)
}

// DatabaseImpl - Implements Database.
type DatabaseImpl struct {
	db *sqlx.DB
	*QueriesImpl
}

// NewDatabase creates a DatabaseImpl with the provided db and dialect.
func NewDatabase(db *sqlx.DB, dialect goqu.DialectWrapper) *DatabaseImpl {
	i := &DatabaseImpl{db: db}
	i.QueriesImpl = &QueriesImpl{db, SQLBuilder{dialect}}
	return i
}

// Transaction - see Database.Transaction
func (d *DatabaseImpl) Transaction(f func(tx Queries) error) error {
	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}
	err = f(&QueriesImpl{tx, d.b})
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			panic(rollbackErr)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			panic(rollbackErr)
		}
		return err
	}

	return nil
}
