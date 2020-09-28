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
	Project struct {
		Id         uint64     `db:"id"`
		Identifier string     `db:"identifier"`
		Label      *string    `db:"label"`
		CreatedAt  time.Time  `db:"created_at"`
		UpdatedAt  time.Time  `db:"updated_at"`
		DeletedAt  *time.Time `db:"deleted_at"`
	}

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

	Aircraft struct {
		Id        uint64    `db:"id"`
		Icao      string    `db:"icao"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

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
		Kml         []byte `db:"kml"`
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
		Contents    []byte `json:"contents"`
	}

	EmailJob struct {
		To          string            `json:"to"`
		Subject     string            `json:"subject"`
		Body        string            `json:"body"`
		Attachments []EmailAttachment `json:"attachments"`
	}
)

func (k *SightingKml) UpdateKml(kml []byte) error {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(kml)
	if err != nil {
		return err
	}
	k.ContentType = KmlGzipContentType
	k.Kml = b.Bytes()
	return nil
}

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
	Transaction(f func(tx *sqlx.Tx) error) error
	NewProject(siteName string, now time.Time) (sql.Result, error)
	LoadProject(siteName string) (*Project, error)
	NewSession(site *Project, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool) (sql.Result, error)
	LoadSessionByIdentifier(site *Project, identifier string) (*Session, error)
	CloseSession(session *Session) (sql.Result, error)
	LoadAircraftByIcao(icao string) (*Aircraft, error)
	LoadAircraftById(id int64) (*Aircraft, error)
	CreateAircraft(icao string) (sql.Result, error)
	CreateSighting(session *Session, ac *Aircraft) (sql.Result, error)
	CreateSightingTx(tx *sqlx.Tx, session *Session, ac *Aircraft) (sql.Result, error)
	ReopenSighting(sighting *Sighting) (sql.Result, error)
	ReopenSightingTx(tx *sqlx.Tx, sighting *Sighting) (sql.Result, error)
	LoadLastSighting(session *Session, ac *Aircraft) (*Sighting, error)
	LoadLastSightingTx(tx *sqlx.Tx, session *Session, ac *Aircraft) (*Sighting, error)
	UpdateSightingCallsignTx(tx *sqlx.Tx, sighting *Sighting, callsign string) (sql.Result, error)
	UpdateSightingSquawkTx(tx *sqlx.Tx, sighting *Sighting, squawk string) (sql.Result, error)
	CloseSightingBatch(sightings []*Sighting) error
	LoadSightingById(sightingId int64) (*Sighting, error)
	LoadSightingByIdTx(tx *sqlx.Tx, sightingId int64) (*Sighting, error)
	CreateNewSightingCallSignTx(tx *sqlx.Tx, sighting *Sighting, callsign string, observedAt time.Time) (sql.Result, error)
	InsertSightingLocation(sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error)
	InsertSightingLocationTx(tx *sqlx.Tx, sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error)
	GetLocationHistory(sighting *Sighting, lastId int64, batchSize int64) (*sqlx.Rows, error)
	GetLocationHistoryWalkBatch(sighting *Sighting, batchSize int64, f func([]SightingLocation)) error
	GetFullLocationHistory(sighting *Sighting, batchSize int64) ([]SightingLocation, error)
	CreateNewSightingSquawkTx(tx *sqlx.Tx, sighting *Sighting, squawk string, observedAt time.Time) (sql.Result, error)
	LoadSightingKml(sighting *Sighting) (*SightingKml, error)
	UpdateSightingKml(sightingKml *SightingKml) (sql.Result, error)
	CreateSightingKml(sighting *Sighting, kmlData []byte) (sql.Result, error)
	CreateEmailJobTx(tx *sqlx.Tx, createdAt time.Time, content []byte) (sql.Result, error)
	GetPendingEmailJobs(now time.Time) ([]Email, error)
	DeleteCompletedEmail(tx *sqlx.Tx, job Email) (sql.Result, error)
	MarkEmailFailedTx(tx *sqlx.Tx, job Email) (sql.Result, error)
	RetryEmailAfter(tx *sqlx.Tx, job Email, retryAfter time.Time) (sql.Result, error)
}
type DatabaseImpl struct {
	db      *sqlx.DB
	dialect goqu.DialectWrapper
}

func NewDatabase(db *sqlx.DB, dialect goqu.DialectWrapper) *DatabaseImpl {
	return &DatabaseImpl{db: db, dialect: dialect}
}
func (d *DatabaseImpl) Transaction(f func(tx *sqlx.Tx) error) error {
	return NewTxExecer(d.db, f).Exec()
}
func (d *DatabaseImpl) NewProject(siteName string, now time.Time) (sql.Result, error) {
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

func (d *DatabaseImpl) LoadProject(siteName string) (*Project, error) {
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
func (d *DatabaseImpl) NewSession(site *Project, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool) (sql.Result, error) {
	now := time.Now()
	s, p, err := d.dialect.
		Insert("session").
		Prepared(true).
		Cols("project_id", "identifier", "with_squawks", "with_transmission_types",
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
func (d *DatabaseImpl) LoadSessionByIdentifier(site *Project, identifier string) (*Session, error) {
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
func (d *DatabaseImpl) LoadLastSighting(session *Session, ac *Aircraft) (*Sighting, error) {
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
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return sighting, nil
}

func (d *DatabaseImpl) LoadLastSightingTx(tx *sqlx.Tx, session *Session, ac *Aircraft) (*Sighting, error) {
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
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return sighting, nil
}
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
	return tx.Exec(s, p...)
}
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
	return tx.Exec(s, p...)
}
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
	// index: aircraft_id, session_id
	row := d.db.QueryRowx(s, p...)
	sighting := &Sighting{}
	err = row.StructScan(sighting)
	if err != nil {
		return nil, err
	}
	return sighting, nil
}
func (d *DatabaseImpl) LoadSightingByIdTx(tx *sqlx.Tx, sightingId int64) (*Sighting, error) {
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
	// index: aircraft_id, session_id
	return d.db.Exec(s, p...)
}

func (d *DatabaseImpl) InsertSightingLocationTx(tx *sqlx.Tx, sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error) {
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
func (d *DatabaseImpl) GetLocationHistory(sighting *Sighting, lastId int64, batchSize int64) (*sqlx.Rows, error) {
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
	for {
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

		_ = res.Close()
		if len(batch) == 0 {
			return nil
		}

		f(batch)
		lastId = int64(batch[len(batch)-1].Id)
		batch = make([]SightingLocation, 0, batchSize)
	}
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
func (d *DatabaseImpl) LoadSightingKml(sighting *Sighting) (*SightingKml, error) {
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
func (d *DatabaseImpl) CreateSightingKml(sighting *Sighting, kmlData []byte) (sql.Result, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(kmlData)
	if err != nil {
		return nil, err
	}
	err = w.Flush()
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

func (d *DatabaseImpl) DeleteCompletedEmail(tx *sqlx.Tx, job Email) (sql.Result, error) {
	return tx.Exec("DELETE FROM email WHERE id = ?", job.Id)
}
func (d *DatabaseImpl) MarkEmailFailedTx(tx *sqlx.Tx, job Email) (sql.Result, error) {
	return tx.Exec("UPDATE email SET retry_after = 0, status = ? WHERE id = ?", EmailFailed, job.Id)
}
func (d *DatabaseImpl) RetryEmailAfter(tx *sqlx.Tx, job Email, retryAfter time.Time) (sql.Result, error) {
	return tx.Exec("UPDATE email SET retry_after = ?, retries = ? WHERE id = ?", retryAfter, job.Retries+1, job.Id)
}
