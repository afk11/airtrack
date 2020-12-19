package db

import (
	"github.com/doug-martin/goqu/v9"
	"time"
)

const (
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

// SQLBuilder is a helper class for producing a named SQL query for
// the dialect.
type SQLBuilder struct {
	dialect goqu.DialectWrapper
}

// CreateProject return the SQL and parameter list to create a project
func (b *SQLBuilder) CreateProject(name string, now time.Time) (string, []interface{}, error) {
	return b.dialect.
		Insert(projectTable).
		Prepared(true).
		Cols("identifier", "created_at", "updated_at").
		Vals(goqu.Vals{name, now, now}).
		ToSQL()
}

// GetProject return the SQL and parameter list to query a project by its name
func (b *SQLBuilder) GetProject(name string) (string, []interface{}, error) {
	return b.dialect.
		From(projectTable).
		Prepared(true).
		Where(goqu.C("identifier").Eq(name)).
		ToSQL()
}

// CreateSession return the SQL and parameter list to create a session for project,
// with the provided attributes.
func (b *SQLBuilder) CreateSession(project *Project, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool, now time.Time) (string, []interface{}, error) {
	return b.dialect.
		Insert(sessionTable).
		Prepared(true).
		Cols("project_id", "identifier", "with_squawks", "with_transmission_types",
			"with_callsigns", "created_at", "updated_at", "closed_at").
		Vals(goqu.Vals{project.ID, identifier, withSquawks, withTxTypes, withCallSigns, now, now, nil}).
		ToSQL()
}

// GetSessionByIdentifier return the SQL and parameter list to find a session for a
// project by its identifier.
func (b *SQLBuilder) GetSessionByIdentifier(project *Project, identifier string) (string, []interface{}, error) {
	return b.dialect.
		From(sessionTable).
		Prepared(true).
		Where(goqu.Ex{
			"project_id": project.ID,
			"identifier": identifier,
		}).
		ToSQL()
}

// CloseSession return the SQL and parameter list to close a session.
func (b *SQLBuilder) CloseSession(session *Session, closedAt time.Time) (string, []interface{}, error) {
	return b.dialect.
		Update(sessionTable).
		Prepared(true).
		Set(goqu.Ex{
			"closed_at": closedAt,
		}).
		Where(goqu.C("id").Eq(session.ID)).
		ToSQL()
}

// GetAircraftByIcao return the SQL and parameter list to query for an aircraft by its ICAO.
func (b *SQLBuilder) GetAircraftByIcao(icao string) (string, []interface{}, error) {
	return b.dialect.
		From(aircraftTable).
		Prepared(true).
		Where(goqu.C("icao").Eq(icao)).
		ToSQL()
}

// GetAircraftByID returns the SQL and parameter list to query for an aircraft by its ID.
func (b *SQLBuilder) GetAircraftByID(id uint64) (string, []interface{}, error) {
	return b.dialect.
		From(aircraftTable).
		Prepared(true).
		Where(goqu.C("id").Eq(id)).
		ToSQL()
}

// CreateAircraft returns the SQL and parameter list to create a new aircraft.
func (b *SQLBuilder) CreateAircraft(icao string, firstSeenTime time.Time) (string, []interface{}, error) {
	return b.dialect.
		Insert(aircraftTable).
		Prepared(true).
		Cols("icao", "created_at", "updated_at").
		Vals(goqu.Vals{icao, firstSeenTime, firstSeenTime}).
		ToSQL()
}

// CreateSighting returns the SQL and parameter list to create a sighting of ac in session.
func (b *SQLBuilder) CreateSighting(session *Session, ac *Aircraft, firstSeenTime time.Time) (string, []interface{}, error) {
	return b.dialect.
		Insert(sightingTable).
		Prepared(true).
		Cols("project_id", "session_id", "aircraft_id", "created_at", "updated_at").
		Vals(goqu.Vals{session.ProjectID, session.ID, ac.ID, &firstSeenTime, &firstSeenTime}).
		ToSQL()
}

// ReopenSighting returns the SQL and parameter list to reopen a closed sighting by its ID.
func (b *SQLBuilder) ReopenSighting(sighting *Sighting) (string, []interface{}, error) {
	return b.dialect.
		Update(sessionTable).
		Prepared(true).
		Set(goqu.Ex{
			"closed_at": nil,
		}).
		Where(goqu.C("id").Eq(sighting.ID)).
		ToSQL()
}

// GetLastSighting returns the SQL and parameter list to query for the most recent sighting of ac
// in session (if there was any)
func (b *SQLBuilder) GetLastSighting(session *Session, ac *Aircraft) (string, []interface{}, error) {
	return b.dialect.
		From(sightingTable).
		Prepared(true).
		Where(goqu.Ex{
			"session_id":  session.ID,
			"aircraft_id": ac.ID,
		}).
		Order(goqu.C("id").Desc()).
		Limit(1).
		ToSQL()
}

// UpdateSightingCallsign returns the SQL and parameter list to update the Sighting's callsign value
func (b *SQLBuilder) UpdateSightingCallsign(sighting *Sighting, callsign string) (string, []interface{}, error) {
	return b.dialect.
		Update(sightingTable).
		Prepared(true).
		Set(goqu.Ex{
			"callsign": callsign,
		}).
		Where(goqu.C("id").Eq(sighting.ID)).
		ToSQL()
}

// UpdateSightingSquawk returns the SQL and parameter list to update the Sighting's squawk value
func (b *SQLBuilder) UpdateSightingSquawk(sighting *Sighting, squawk string) (string, []interface{}, error) {
	return b.dialect.
		Update(sightingTable).
		Prepared(true).
		Set(goqu.Ex{
			"squawk": squawk,
		}).
		Where(goqu.C("id").Eq(sighting.ID)).
		ToSQL()
}

// CloseSightingsBatch returns the SQL and parameter list to close many sightings at once.
func (b *SQLBuilder) CloseSightingsBatch(ids []uint64, closedAt time.Time) (string, []interface{}, error) {
	return b.dialect.
		Update(sightingTable).
		Prepared(true).
		Set(goqu.Ex{
			"closed_at": closedAt,
		}).
		Where(goqu.C("id").In(ids)).
		ToSQL()
}

// GetSchemaMigration returns the SQL and parameter list to query for the migrations state
func (b *SQLBuilder) GetSchemaMigration() (string, []interface{}, error) {
	return b.dialect.
		From(schemaMigrationsTable).
		Prepared(true).
		ToSQL()
}

// GetSightingByID returns the SQL and parameter list to query for a sighting by its ID.
func (b *SQLBuilder) GetSightingByID(sightingID uint64) (string, []interface{}, error) {
	return b.dialect.
		From(sightingTable).
		Prepared(true).
		Where(goqu.C("id").Eq(sightingID)).
		Limit(1).
		ToSQL()
}

// CreateNewSightingCallSign returns the SQL and parameter list to create a new
// SightingCallSign record
func (b *SQLBuilder) CreateNewSightingCallSign(sighting *Sighting, callsign string, observedAt time.Time) (string, []interface{}, error) {
	return b.dialect.
		Insert(sightingCallsignTable).
		Prepared(true).
		Cols("sighting_id", "callsign", "observed_at").
		Vals(goqu.Vals{sighting.ID, callsign, observedAt}).
		ToSQL()
}

// GetLastSightingCallSign returns the SQL and parameter list to query the
// most recent SightingCallsign for sighting (if any)
func (b *SQLBuilder) GetLastSightingCallSign(sighting *Sighting) (string, []interface{}, error) {
	return b.dialect.
		From(sightingCallsignTable).
		Prepared(true).
		Where(goqu.C("sighting_id").Eq(sighting.ID)).
		Order(goqu.C("id").Desc()).
		Limit(1).
		ToSQL()
}

// CreateSightingLocation returns the SQL and parameter list to insert a
// new position for associated with sightingID
func (b *SQLBuilder) CreateSightingLocation(sightingID uint64, t time.Time, altitude int64, lat float64, long float64) (string, []interface{}, error) {
	return b.dialect.
		Insert(sightingLocationTable).
		Prepared(true).
		Cols("sighting_id", "timestamp", "altitude", "latitude", "longitude").
		Vals(goqu.Vals{sightingID, t, altitude, lat, long}).
		ToSQL()
}

// LoadLocationHistory returns the SQL and parameter list to query for a
// batch of SightingLocation records for a sighting. Records will start
// from at least lastID, and will contain at most batchSize results.
func (b *SQLBuilder) LoadLocationHistory(sighting *Sighting, lastID int64, batchSize int64) (string, []interface{}, error) {
	return b.dialect.
		From(sightingLocationTable).
		Prepared(true).
		Where(goqu.Ex{
			"sighting_id": sighting.ID,
			"id":          goqu.Op{"gt": lastID},
		}).
		Limit(uint(batchSize)).
		ToSQL()
}

// CreateNewSightingSquawk returns the SQL and parameter list to create a new
// SightingSquawk record
func (b *SQLBuilder) CreateNewSightingSquawk(sighting *Sighting, squawk string, observedAt time.Time) (string, []interface{}, error) {
	return b.dialect.
		Insert(sightingSquawkTable).
		Prepared(true).
		Cols("sighting_id", "squawk", "observed_at").
		Vals(goqu.Vals{sighting.ID, squawk, observedAt}).
		ToSQL()
}

// GetLastSightingSquawk returns the SQL and parameter list to query the
// most recent SightingSquawk for sighting (if any)
func (b *SQLBuilder) GetLastSightingSquawk(sighting *Sighting) (string, []interface{}, error) {
	return b.dialect.
		From(sightingSquawkTable).
		Prepared(true).
		Where(goqu.C("sighting_id").Eq(sighting.ID)).
		Order(goqu.C("id").Desc()).
		Limit(1).
		ToSQL()
}

// GetSightingKml returns the SQL and parameter list to load a SightingKML
// record for the provided sighting.
func (b *SQLBuilder) GetSightingKml(sighting *Sighting) (string, []interface{}, error) {
	return b.dialect.
		From(sightingKmlTable).
		Prepared(true).
		Where(goqu.Ex{
			"sighting_id": sighting.ID,
		}).
		ToSQL()
}

// UpdateSightingKml returns the SQL and parameter list to update the KML
// of the provided SightingKml record.
func (b *SQLBuilder) UpdateSightingKml(sightingKml *SightingKml) (string, []interface{}, error) {
	return b.dialect.
		Update(sightingKmlTable).
		Prepared(true).
		Set(goqu.Ex{
			"kml":          sightingKml.Kml,
			"content_type": sightingKml.ContentType,
		}).
		Where(goqu.C("id").Eq(sightingKml.ID)).
		ToSQL()
}

// CreateSightingKmlContent returns the SQL and parameter list to create a
// new SightingKml record associated with Sighting
func (b *SQLBuilder) CreateSightingKmlContent(sighting *Sighting, contentType int, kmlData []byte) (string, []interface{}, error) {
	return b.dialect.
		Insert(sightingKmlTable).
		Prepared(true).
		Cols("sighting_id", "content_type", "kml").
		Vals(goqu.Vals{sighting.ID, contentType, kmlData}).
		ToSQL()
}

// CreateEmailJob returns the SQL and parameter list to create a
// new Email with the provided content.
func (b *SQLBuilder) CreateEmailJob(createdAt time.Time, content []byte) (string, []interface{}, error) {
	return b.dialect.
		Insert(emailTable).
		Prepared(true).
		Cols("status", "retries", "created_at", "updated_at", "retry_after", "job").
		Vals(goqu.Vals{EmailPending, 0, createdAt, createdAt, nil, content}).
		ToSQL()
}

// GetPendingEmailJobs returns the SQL and parameter list to query for Email's
// withe the pending status due to be retried <= now.
func (b *SQLBuilder) GetPendingEmailJobs(now time.Time) (string, []interface{}, error) {
	return b.dialect.
		From(emailTable).
		Prepared(true).
		Where(goqu.C("status").Eq(EmailPending)).
		Where(goqu.Or(
			goqu.C("retry_after").Eq(nil),
			goqu.C("retry_after").Lte(now))).
		Order(goqu.C("id").Desc()).
		ToSQL()
}

// DeleteCompletedEmail returns the SQL and parameter list to delete an
// Email which has been completely processed
func (b *SQLBuilder) DeleteCompletedEmail(job Email) (string, []interface{}, error) {
	return b.dialect.
		Delete(emailTable).
		Prepared(true).
		Where(goqu.C("id").Eq(job.ID)).
		ToSQL()
}

// MarkEmailFailed returns the SQL and parameter list to update an
// Email to the FAILED status (preventing further attempts)
func (b *SQLBuilder) MarkEmailFailed(job *Email) (string, []interface{}, error) {
	return b.dialect.
		Update(emailTable).
		Prepared(true).
		Set(goqu.Ex{
			"status": EmailFailed,
		}).
		Where(goqu.C("id").Eq(job.ID)).
		ToSQL()
}

// RetryEmailAfter returns the SQL and parameter list to set a new retry
// time for an Email job, and increase the retry count.
func (b *SQLBuilder) RetryEmailAfter(job *Email, retryAfter time.Time) (string, []interface{}, error) {
	return b.dialect.
		Update(emailTable).
		Prepared(true).
		Set(goqu.Ex{
			"retry_after": retryAfter,
			"retries":     job.Retries + 1,
		}).
		Where(goqu.C("id").Eq(job.ID)).
		ToSQL()
}
