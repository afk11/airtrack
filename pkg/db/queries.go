package db

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"time"
)

// QueriesImpl - Implements Queries.
type QueriesImpl struct {
	db sqlx.Ext
	b  SQLBuilder
}

// CreateProject - see Queries.CreateProject
func (d *QueriesImpl) CreateProject(name string, now time.Time) (sql.Result, error) {
	s, p, err := d.b.CreateProject(name, now)
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// GetProject - see Queries.GetProject
func (d *QueriesImpl) GetProject(name string) (*Project, error) {
	s, p, err := d.b.GetProject(name)
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

// CreateSession - see Queries.CreateSession
func (d *QueriesImpl) CreateSession(project *Project, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.b.CreateSession(project, identifier, withSquawks, withTxTypes, withCallSigns, now)
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetSessionByIdentifier - see Queries.GetSessionByIdentifier
func (d *QueriesImpl) GetSessionByIdentifier(project *Project, identifier string) (*Session, error) {
	s, p, err := d.b.GetSessionByIdentifier(project, identifier)
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

// CloseSession - see Queries.CloseSession
func (d *QueriesImpl) CloseSession(session *Session, closedAt time.Time) (sql.Result, error) {
	s, p, err := d.b.CloseSession(session, closedAt)
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

// GetAircraftByIcao - see Queries.GetAircraftByIcao
func (d *QueriesImpl) GetAircraftByIcao(icao string) (*Aircraft, error) {
	s, p, err := d.b.GetAircraftByIcao(icao)
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

// GetAircraftByID - see Queries.GetAircraftByID
func (d *QueriesImpl) GetAircraftByID(id uint64) (*Aircraft, error) {
	s, p, err := d.b.GetAircraftByID(id)
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

// CreateAircraft - see Queries.CreateAircraft
func (d *QueriesImpl) CreateAircraft(icao string, firstSeenTime time.Time) (sql.Result, error) {
	s, p, err := d.b.CreateAircraft(icao, firstSeenTime)
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// CreateSighting - see Queries.CreateSighting
func (d *QueriesImpl) CreateSighting(session *Session, ac *Aircraft, firstSeenTime time.Time) (sql.Result, error) {
	s, p, err := d.b.CreateSighting(session, ac, firstSeenTime)
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// ReopenSighting - see Queries.ReopenSighting
func (d *QueriesImpl) ReopenSighting(sighting *Sighting) (sql.Result, error) {
	s, p, err := d.b.ReopenSighting(sighting)
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

// GetLastSighting - see Queries.GetLastSighting
func (d *QueriesImpl) GetLastSighting(session *Session, ac *Aircraft) (*Sighting, error) {
	s, p, err := d.b.GetLastSighting(session, ac)
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

// UpdateSightingCallsign - see Queries.UpdateSightingCallsign
func (d *QueriesImpl) UpdateSightingCallsign(sighting *Sighting, callsign string) (sql.Result, error) {
	s, p, err := d.b.UpdateSightingCallsign(sighting, callsign)
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	sighting.CallSign = &callsign
	return res, nil
}

// UpdateSightingSquawk - see Queries.UpdateSightingSquawk
func (d *QueriesImpl) UpdateSightingSquawk(sighting *Sighting, squawk string) (sql.Result, error) {
	s, p, err := d.b.UpdateSightingSquawk(sighting, squawk)
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	sighting.Squawk = &squawk
	return res, nil
}

// CloseSightingBatch - see Queries.CloseSightingBatch
func (d *QueriesImpl) CloseSightingBatch(sightings []*Sighting, closedAt time.Time) error {
	if len(sightings) == 0 {
		return nil
	}
	n := len(sightings)
	var ids []uint64
	for i := 0; i < n; i++ {
		ids = append(ids, sightings[i].ID)
	}

	s, p, err := d.b.CloseSightingsBatch(ids, closedAt)
	if err != nil {
		return err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return err
	}
	err = CheckRowsUpdated(res, int64(n))
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		sightings[i].ClosedAt = &closedAt
	}
	return nil
}

// GetSchemaMigration - see Queries.GetSchemaMigration
func (d *QueriesImpl) GetSchemaMigration() (*SchemaMigrations, error) {
	s, p, err := d.b.GetSchemaMigration()
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

// GetSightingByID - see Queries.GetSightingByID
func (d *QueriesImpl) GetSightingByID(sightingID uint64) (*Sighting, error) {
	s, p, err := d.b.GetSightingByID(sightingID)
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

// CreateNewSightingCallSign - see Queries.CreateNewSightingCallSign
func (d *QueriesImpl) CreateNewSightingCallSign(sighting *Sighting, callsign string, observedAt time.Time) (sql.Result, error) {
	s, p, err := d.b.CreateNewSightingCallSign(sighting, callsign, observedAt)
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// GetLastSightingCallSign - see Queries.GetLastSightingCallSign
func (d *QueriesImpl) GetLastSightingCallSign(sighting *Sighting) (*SightingCallSign, error) {
	s, p, err := d.b.GetLastSightingCallSign(sighting)
	if err != nil {
		return nil, err
	}

	callsign := SightingCallSign{}
	row := d.db.QueryRowx(s, p...)
	if err = row.StructScan(&callsign); err != nil {
		return nil, err
	}
	return &callsign, nil
}

// CreateSightingLocation - see Queries.CreateSightingLocation
func (d *QueriesImpl) CreateSightingLocation(sightingID uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error) {
	s, p, err := d.b.CreateSightingLocation(sightingID, t, altitude, lat, long)
	if err != nil {
		return nil, err
	}
	// index: aircraft_id, session_id
	return d.db.Exec(s, p...)
}

// LoadLocationHistory - see Queries.LoadLocationHistory
func (d *QueriesImpl) LoadLocationHistory(sighting *Sighting, lastID int64, batchSize int64) (*sqlx.Rows, error) {
	s, p, err := d.b.LoadLocationHistory(sighting, lastID, batchSize)
	if err != nil {
		return nil, err
	}
	res, err := d.db.Queryx(s, p...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// WalkLocationHistoryBatch - see Queries.WalkLocationHistoryBatch
func (d *QueriesImpl) WalkLocationHistoryBatch(sighting *Sighting, batchSize int64, f func([]SightingLocation)) error {
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

// GetFullLocationHistory - see Queries.GetFullLocationHistory
func (d *QueriesImpl) GetFullLocationHistory(sighting *Sighting, batchSize int64) ([]SightingLocation, error) {
	var h []SightingLocation
	err := d.WalkLocationHistoryBatch(sighting, batchSize, func(location []SightingLocation) {
		h = append(h, location...)
	})
	if err != nil {
		return nil, err
	}
	return h, nil
}

// CreateNewSightingSquawk - see Queries.CreateNewSightingSquawk
func (d *QueriesImpl) CreateNewSightingSquawk(sighting *Sighting, squawk string, observedAt time.Time) (sql.Result, error) {
	s, p, err := d.b.CreateNewSightingSquawk(sighting, squawk, observedAt)
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// GetLastSightingSquawk - see Queries.GetLastSightingSquawk
func (d *QueriesImpl) GetLastSightingSquawk(sighting *Sighting) (*SightingSquawk, error) {
	s, p, err := d.b.GetLastSightingSquawk(sighting)
	if err != nil {
		return nil, err
	}

	squawk := SightingSquawk{}
	row := d.db.QueryRowx(s, p...)
	if err = row.StructScan(&squawk); err != nil {
		return nil, err
	}
	return &squawk, nil
}

// GetSightingKml - see Queries.GetSightingKml
func (d *QueriesImpl) GetSightingKml(sighting *Sighting) (*SightingKml, error) {
	s, p, err := d.b.GetSightingKml(sighting)
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

// UpdateSightingKml - see Queries.UpdateSightingKml
func (d *QueriesImpl) UpdateSightingKml(sightingKml *SightingKml) (sql.Result, error) {
	s, p, err := d.b.UpdateSightingKml(sightingKml)
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

// CreateSightingKmlContent - see Queries.CreateSightingKmlContent
func (d *QueriesImpl) CreateSightingKmlContent(sighting *Sighting, kmlData []byte) (sql.Result, error) {
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

	s, p, err := d.b.CreateSightingKmlContent(sighting, KmlGzipContentType, b.Bytes())
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// CreateEmailJob - see Queries.CreateEmailJob
func (d *QueriesImpl) CreateEmailJob(createdAt time.Time, content []byte) (sql.Result, error) {
	s, p, err := d.b.CreateEmailJob(createdAt, content)
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// GetPendingEmailJobs - see Queries.GetPendingEmailJobs
// Does not return sql.ErrNoRows
func (d *QueriesImpl) GetPendingEmailJobs(now time.Time) ([]Email, error) {
	s, p, err := d.b.GetPendingEmailJobs(now)
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

// DeleteCompletedEmail - see Queries.DeleteCompletedEmail
func (d *QueriesImpl) DeleteCompletedEmail(job Email) (sql.Result, error) {
	s, p, err := d.b.DeleteCompletedEmail(job)
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

// MarkEmailFailed - see Queries.MarkEmailFailed
func (d *QueriesImpl) MarkEmailFailed(job *Email) (sql.Result, error) {
	s, p, err := d.b.MarkEmailFailed(job)
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	ts := time.Unix(0, 0)
	job.RetryAfter = &ts
	job.Status = EmailFailed
	return res, nil
}

// RetryEmailAfter - see Queries.RetryEmailAfter
func (d *QueriesImpl) RetryEmailAfter(job *Email, retryAfter time.Time) (sql.Result, error) {
	s, p, err := d.b.RetryEmailAfter(job, retryAfter)
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	}
	job.RetryAfter = &retryAfter
	job.Retries = job.Retries + 1
	return res, nil
}
