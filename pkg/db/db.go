package db

import (
	"database/sql"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"time"
)

const (
	EmailPending = 1
	EmailFailed  = 0
	KmlPlainTextContentType = 0
)

type (
	CollectionSite struct {
		Id         uint64     `db:"id"`
		Identifier string     `db:"identifier"`
		Label      *string    `db:"label"`
		CreatedAt  time.Time  `db:"created_at"`
		UpdatedAt  time.Time  `db:"updated_at"`
		DeletedAt  *time.Time `db:"deleted_at"`
	}

	CollectionSession struct {
		Id                    uint64     `db:"id"`
		Identifier            string     `db:"identifier"`
		CollectionSiteId      uint64     `db:"collection_site_id"`
		ClosedAt              *time.Time `db:"closed_at"`
		CreatedAt             time.Time  `db:"created_at"`
		UpdatedAt             time.Time  `db:"updated_at"`
		DeletedAt             *time.Time `db:"deleted_at"`
		WithSquawks           bool       `db:"with_squawks"`
		WithTransmissionTypes bool       `db:"with_transmission_types"`
		WithCallSigns         bool       `db:"with_callsigns"`
	}

	Aircraft struct {
		Id        uint64    `db:"id"`
		Icao      string    `db:"icao"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	Sighting struct {
		Id                  uint64  `db:"id"`
		CollectionSiteId    uint64  `db:"collection_site_id"`
		CollectionSessionId uint64  `db:"collection_session_id"`
		AircraftId          uint64  `db:"aircraft_id"`
		CallSign            *string `db:"callsign"`

		CreatedAt time.Time  `db:"created_at"`
		UpdatedAt time.Time  `db:"updated_at"`
		ClosedAt  *time.Time `db:"closed_at"`

		TransmissionTypes uint8   `db:"transmission_types"`
		Squawk            *string `db:"squawk"`
	}

	SightingCallSign struct {
		Id         uint64    `db:"id"`
		SightingId uint64    `db:"sighting_id"`
		CallSign   string    `db:"callsign"`
		ObservedAt time.Time `db:"observed_at"`
	}

	SightingKml struct {
		Id          uint64 `db:"id"`
		SightingId  uint64 `db:"sighting_id"`
		ContentType int32  `db:"content_type"`
		Kml         string `db:"kml"`
	}

	SightingLocation struct {
		Id         uint64    `db:"id"`
		SightingId uint64    `db:"sighting_id"`
		TimeStamp  time.Time `db:"timestamp"`
		Altitude   int64     `db:"altitude"`
		Latitude   float64   `db:"latitude"`
		Longitude  float64   `db:"longitude"`
	}

	SightingSquawk struct {
		Id         uint64    `db:"id"`
		SightingId uint64    `db:"sighting_id"`
		Squawk     string    `db:"squawk"`
		ObservedAt time.Time `db:"observed_at"`
	}

	Email struct {
		Id         uint64     `db:"id"`
		Status     int32      `db:"status"`
		Retries    int32      `db:"retries"`
		RetryAfter *time.Time `db:"retry_after"`
		CreatedAt  time.Time  `db:"created_at"`
		UpdatedAt  time.Time  `db:"updated_at"`
		Job        []byte
	}

	EmailAttachment struct {
		ContentType string `json:"content_type"`
		FileName    string `json:"filename"`
		Contents    string `json:"contents"`
	}

	EmailJob struct {
		To          string            `json:"to"`
		Subject     string            `json:"subject"`
		Body        string            `json:"body"`
		Attachments []EmailAttachment `json:"attachments"`
	}
)

func CheckRowsUpdated(res sql.Result, expectAffected int64) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	} else if affected != expectAffected {
		return errors.Errorf("expected %d rows affected, got %d", expectAffected, affected)
	}
	return nil
}

type Database interface {
	Transaction(f func(tx *sql.Tx) error) error
	NewCollectionSite(siteName string, now time.Time) (sql.Result, error)
	LoadCollectionSite(siteName string) (*CollectionSite, error)
	NewCollectionSession(site *CollectionSite, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool) (sql.Result, error)
	LoadSessionByIdentifier(site *CollectionSite, identifier string) (*CollectionSession, error)
	CloseSession(session *CollectionSession) (sql.Result, error)
	LoadAircraftByIcao(icao string) (*Aircraft, error)
	LoadAircraftById(id int64) (*Aircraft, error)
	CreateAircraft(icao string) (sql.Result, error)
	CreateSighting(session *CollectionSession, ac *Aircraft) (sql.Result, error)
	ReopenSighting(sighting *Sighting) (sql.Result, error)
	LoadLastSighting(session *CollectionSession, ac *Aircraft) (*Sighting, error)
	UpdateSightingCallsignTx(tx *sql.Tx, sighting *Sighting, callsign string) (sql.Result, error)
	UpdateSightingSquawkTx(tx *sql.Tx, sighting *Sighting, squawk string) (sql.Result, error)
	CloseSightingBatch(sightings []*Sighting) error
	LoadSightingById(sightingId int64) (*Sighting, error)
	CreateNewSightingCallSignTx(tx *sql.Tx, sighting *Sighting, callsign string, observedAt time.Time) (sql.Result, error)
	InsertSightingLocation(sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error)
	GetLocationHistory(sighting *Sighting, lastId int64, batchSize int64) (*sqlx.Rows, error)
	GetLocationHistoryWalkBatch(sighting *Sighting, batchSize int64, f func([]SightingLocation)) error
	GetFullLocationHistory(sighting *Sighting, batchSize int64) ([]SightingLocation, error)
	CreateNewSightingSquawkTx(tx *sql.Tx, sighting *Sighting, squawk string, observedAt time.Time) (sql.Result, error)
	LoadSightingKml(sighting *Sighting) (*SightingKml, error)
	UpdateSightingKml(sightingKml *SightingKml, kmlStr string) (sql.Result, error)
	CreateSightingKml(sighting *Sighting, kmlData string) (sql.Result, error)
	CreateEmailJobTx(tx *sql.Tx, createdAt time.Time, content []byte) (sql.Result, error)
	GetPendingEmailJobs(now time.Time) ([]Email, error)
	DeleteCompletedEmail(tx *sql.Tx, job Email) (sql.Result, error)
	MarkEmailFailedTx(tx *sql.Tx, job Email) (sql.Result, error)
	RetryEmailAfter(tx *sql.Tx, job Email, retryAfter time.Time) (sql.Result, error)
}
type DatabaseImpl struct {
	db *sqlx.DB
	dialect goqu.DialectWrapper
}
func NewDatabase(db *sqlx.DB, dialect goqu.DialectWrapper) *DatabaseImpl {
	return &DatabaseImpl{db: db, dialect: dialect}
}
func (d *DatabaseImpl) Transaction(f func (tx *sql.Tx) error) error {
	return NewTxExecer(d.db.DB, f).Exec()
}
func (d *DatabaseImpl) NewCollectionSite(siteName string, now time.Time) (sql.Result, error) {
	q := d.dialect.
		Insert("collection_site").
		Prepared(true).
		Cols("identifier", "created_at", "updated_at").
		Vals(goqu.Vals{siteName, now, now})
	s, p, err := q.ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

func (d *DatabaseImpl) LoadCollectionSite(siteName string) (*CollectionSite, error) {
	q := d.dialect.
		From("collection_site").
		Prepared(true).
		Where(goqu.C("identifier").Eq(siteName))
	s, p, err := q.ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	site := CollectionSite{}
	err = row.StructScan(&site)
	if err != nil {
		return nil, err
	}
	return &site, err
}
func (d *DatabaseImpl) NewCollectionSession(site *CollectionSite, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Insert("collection_session").
		Prepared(true).
		Cols("collection_site_id", "identifier", "with_squawks",	"with_transmission_types",
			"with_callsigns", "created_at", "updated_at", "closed_at").
		Vals(goqu.Vals{site.Id, identifier, withSquawks, withTxTypes, withCallSigns, now, now, nil}).
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
func (d *DatabaseImpl) LoadSessionByIdentifier(site *CollectionSite, identifier string) (*CollectionSession, error) {
	s, p, err := d.dialect.
		From("collection_session").
		Prepared(true).
		Where(goqu.Ex{
			"collection_site_id": site.Id,
			"identifier": identifier,
		}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	row := d.db.QueryRowx(s, p...)
	session := &CollectionSession{}
	err = row.StructScan(session)
	if err != nil {
		return nil, err
	}
	return session, nil
}
func (d *DatabaseImpl) CloseSession(session *CollectionSession) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Update("collection_session").
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
		// todo: why here? why not after this block?
		session.ClosedAt = &now
		return nil, err
	}
	return res, nil
}
func (d *DatabaseImpl) LoadAircraftByIcao(icao string) (*Aircraft, error) {
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
func (d *DatabaseImpl) LoadAircraftById(id int64) (*Aircraft, error) {
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

func (d *DatabaseImpl) CreateSighting(session *CollectionSession, ac *Aircraft) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Insert("sighting").
		Prepared(true).
		Cols("collection_site_id", "collection_session_id", "aircraft_id", "created_at", "updated_at").
		Vals(goqu.Vals{session.CollectionSiteId, session.Id, ac.Id, &now, &now}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}
func (d *DatabaseImpl) ReopenSighting(sighting *Sighting) (sql.Result, error) {
	s, p, err := d.dialect.
		Update("collection_session").
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
func (d *DatabaseImpl) LoadLastSighting(session *CollectionSession, ac *Aircraft) (*Sighting, error) {
	s, p, err := d.dialect.
		From("sighting").
		Prepared(true).
		Where(goqu.Ex{
			"collection_session_id": session.Id,
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
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return sighting, nil
}
func (d *DatabaseImpl) UpdateSightingCallsignTx(tx *sql.Tx, sighting *Sighting, callsign string) (sql.Result, error) {
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
	return tx.Exec(s, p...)
}
func (d *DatabaseImpl) UpdateSightingSquawkTx(tx *sql.Tx, sighting *Sighting, squawk string) (sql.Result, error) {
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
	return tx.Exec(s, p...)
}
func (d *DatabaseImpl) CloseSightingBatch(sightings []*Sighting) error {
	if len(sightings) == 0 {
		return nil
	}
	n := len(sightings)
	closedAt := time.Now()
	err := d.Transaction(func(tx *sql.Tx) error {
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
func (d *DatabaseImpl) LoadSightingById(sightingId int64) (*Sighting, error) {
	s, p, err := d.dialect.
		From("sighting").
		Prepared(true).
		Where(goqu.C("id").Eq(sightingId)).
		Limit(1).
		ToSQL()
	if err != nil {
		return nil, err
	}
	// index: aircraft_id, collection_session_id
	row := d.db.QueryRowx(s, p...)
	sighting := &Sighting{}
	err = row.StructScan(sighting)
	if err != nil {
		return nil, err
	}
	return sighting, nil
}

func (d *DatabaseImpl) CreateNewSightingCallSignTx(tx *sql.Tx, sighting *Sighting, callsign string, observedAt time.Time) (sql.Result, error) {
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

func (d *DatabaseImpl) InsertSightingLocation(sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert("sighting_location").
		Prepared(true).
		Cols("sighting_id", "timestamp", "altitude", "latitude", "longitude").
		Vals(goqu.Vals{sightingId, t, altitude, lat, long}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	// index: aircraft_id, collection_session_id
	return d.db.Exec(s, p...)
}

func (d *DatabaseImpl) GetLocationHistory(sighting *Sighting, lastId int64, batchSize int64) (*sqlx.Rows, error) {
	s, p, err := d.dialect.
		From("sighting_location").
		Prepared(true).
		Where(goqu.Ex{
			"sighting_id": sighting.Id,
			"id": lastId,
		}).
		Limit(uint(batchSize)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := d.db.Queryx(s, p...)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}
func (d *DatabaseImpl) GetLocationHistoryWalkBatch(sighting *Sighting, batchSize int64, f func([]SightingLocation)) error {
	lastId := int64(-1)
	batch := make([]SightingLocation, 0, batchSize)
	more := true
	for more {
		more = false
		// res - needs closing
		res, err := d.GetLocationHistory(sighting, lastId, batchSize)
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
		res.Close()

		more = len(batch) > 0
		if more {
			f(batch)
			lastId = int64(batch[len(batch)-1].Id)
			batch = make([]SightingLocation, 0, batchSize)
		}
	}
	return nil
}
func (d *DatabaseImpl) GetFullLocationHistory(sighting *Sighting, batchSize int64) ([]SightingLocation, error) {
	var h []SightingLocation
	err := d.GetLocationHistoryWalkBatch(sighting, batchSize, func(location []SightingLocation) {
		h = append(h, location...)
	})
	if err != nil {
		return nil, err
	}
	return h, nil
}
func (d *DatabaseImpl) CreateNewSightingSquawkTx(tx *sql.Tx, sighting *Sighting, squawk string, observedAt time.Time) (sql.Result, error) {
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
func (d *DatabaseImpl) LoadSightingKml( sighting *Sighting) (*SightingKml, error) {
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
func (d *DatabaseImpl) UpdateSightingKml(sightingKml *SightingKml, kmlStr string) (sql.Result, error) {
	isSameKml := sightingKml.Kml == kmlStr
	s, p, err := d.dialect.
		Update("sighting_kml").
		Prepared(true).
		Set(goqu.Ex{
			"kml": kmlStr,
		}).
		Where(goqu.C("id").In(sightingKml.Id)).
		ToSQL()
	if err != nil {
		return nil, err
	}
	res, err := d.db.Exec(s, p...)
	if err != nil {
		return nil, err
	} else if err = CheckRowsUpdated(res, 1); err != nil {
		if isSameKml {
			panic("rows wrong, and isSameKml!")
		}
		return nil, err
	}
	sightingKml.Kml = kmlStr
	return res, nil
}
func (d *DatabaseImpl) CreateSightingKml(sighting *Sighting, kmlData string) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert("sighting_kml").
		Prepared(true).
		Cols("sighting_id", "content_type", "kml").
		Vals(goqu.Vals{sighting.Id, KmlPlainTextContentType, kmlData}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return d.db.Exec(s, p...)
}

func (d *DatabaseImpl) CreateEmailJobTx(tx *sql.Tx, createdAt time.Time, content []byte) (sql.Result, error) {
	s, p, err := d.dialect.
		Insert("email_v2").
		Prepared(true).
		Cols("status", "retries", "created_at", "updated_at", "retry_after", "job").
		Vals(goqu.Vals{EmailPending, 0, createdAt, createdAt, nil, content}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	return tx.Exec(s, p...)
}

func (d *DatabaseImpl) GetPendingEmailJobs(now time.Time) ([]Email, error) {
	s, p, err := d.dialect.
		From("email_v2").
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

func (d *DatabaseImpl) DeleteCompletedEmail(tx *sql.Tx, job Email) (sql.Result, error) {
	return tx.Exec("DELETE FROM email_v2 WHERE id = ?", job.Id)
}
func (d *DatabaseImpl) MarkEmailFailedTx(tx *sql.Tx, job Email) (sql.Result, error) {
	return tx.Exec("UPDATE email_v2 SET retry_after = 0, status = ? WHERE id = ?", EmailFailed, job.Id)
}
func (d *DatabaseImpl) RetryEmailAfter(tx *sql.Tx, job Email, retryAfter time.Time) (sql.Result, error) {
	return tx.Exec("UPDATE email_v2 SET retry_after = ?, retries = ? WHERE id = ?", retryAfter, job.Retries+1, job.Id)
}
