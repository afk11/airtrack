package db

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/url"
	"strings"
	"time"
)

const (
	KmlPlainTextContentType = 0
)

type EmailAttachment struct {
	ContentType string `json:"content_type"`
	FileName    string `json:"filename"`
	Contents    string `json:"contents"`
}
type EmailJob struct {
	To          string            `json:"to"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	Attachments []EmailAttachment `json:"attachments"`
}
type EmailStatus int

const (
	EmailPending = 1
	EmailFailed  = 0
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
)

func NewConn(driver string, user string, pass string, host string, port int, db string, location *time.Location) (*sqlx.DB, error) {
	return sqlx.Connect(driver, fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=%s",
		user, pass, host, port, db, url.PathEscape(location.String())))
}
func NewMultiStmtConn(driver string, user string, pass string, host string, port int, db string, location *time.Location) (*sqlx.DB, error) {
	return sqlx.Connect(driver, fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true&loc=%s",
		user, pass, host, port, db, url.PathEscape(location.String())))
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

func NewCollectionSite(db *sqlx.DB, siteName string, now time.Time) (sql.Result, error) {
	return db.Exec("INSERT INTO collection_site (identifier, created_at, updated_at) VALUES (?, ?, ?)", siteName, now, now)
}

func LoadCollectionSite(db *sqlx.DB, siteName string) (*CollectionSite, error) {
	row := db.QueryRowx("SELECT * FROM collection_site WHERE identifier = ?", siteName)
	site := &CollectionSite{}
	err := row.StructScan(site)
	if err != nil {
		return nil, err
	}
	return site, err
}

func NewCollectionSession(db *sqlx.DB, site *CollectionSite, identifier string, withSquawks bool, withTxTypes bool, withCallSigns bool) (sql.Result, error) {
	now := time.Now()
	res, err := db.Exec(
		`INSERT INTO collection_session (collection_site_id, identifier, with_squawks, 
		with_transmission_types, with_callsigns, created_at, updated_at, closed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		site.Id, identifier, withSquawks, withTxTypes, withCallSigns, now, now, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}
func LoadSessionByIdentifier(db *sqlx.DB, site *CollectionSite, identifier string) (*CollectionSession, error) {
	row := db.QueryRowx("SELECT * FROM collection_session WHERE collection_site_id = ? and identifier = ?", site.Id, identifier)
	session := &CollectionSession{}
	err := row.StructScan(session)
	if err != nil {
		return nil, err
	}
	return session, nil
}
func CloseSession(db *sqlx.DB, session *CollectionSession) (sql.Result, error) {
	now := time.Now()
	res, err := db.Exec(
		`UPDATE collection_session SET closed_at = ? WHERE id = ?`,
		now, session.Id)
	if err != nil {
		session.ClosedAt = &now
		return nil, err
	}
	return res, nil
}

func LoadAircraftByIcao(db *sqlx.DB, icao string) (*Aircraft, error) {
	row := db.QueryRowx("SELECT * FROM aircraft WHERE icao = ?", icao)
	aircraft := &Aircraft{}
	err := row.StructScan(aircraft)
	if err != nil {
		return nil, err
	}
	return aircraft, nil
}
func LoadAircraftById(db *sqlx.DB, id int64) (*Aircraft, error) {
	row := db.QueryRowx("SELECT * FROM aircraft WHERE id = ?", id)
	aircraft := &Aircraft{}
	err := row.StructScan(aircraft)
	if err != nil {
		return nil, err
	}
	return aircraft, nil
}
func CreateAircraft(db *sqlx.DB, icao string) (sql.Result, error) {
	now := time.Now()
	return db.Exec("INSERT INTO aircraft (icao, created_at, updated_at) VALUES (?, ?, ?)", icao, now, now)
}

func CreateSighting(db *sqlx.DB, session *CollectionSession, ac *Aircraft) (sql.Result, error) {
	now := time.Now()
	return db.Exec("INSERT INTO sighting (collection_site_id, collection_session_id, aircraft_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		session.CollectionSiteId, session.Id, ac.Id, &now, &now)
}
func ReopenSighting(db *sqlx.DB, sighting *Sighting) (sql.Result, error) {
	res, err := db.Exec(
		`UPDATE sighting SET closed_at = NULL WHERE id = ?`,
		sighting.Id)
	if err != nil {
		return nil, err
	}
	sighting.ClosedAt = nil
	return res, nil
}
func LoadLastSighting(db *sqlx.DB, session *CollectionSession, ac *Aircraft) (*Sighting, error) {
	row := db.QueryRowx(`SELECT * FROM sighting 
	WHERE collection_session_id = ? AND aircraft_id = ?
	ORDER BY id DESC 
	LIMIT 1`, session.Id, ac.Id)
	sighting := &Sighting{}
	err := row.StructScan(sighting)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return sighting, nil
}
func CloseSighting(db *sqlx.DB, sighting *Sighting) (sql.Result, error) {
	now := time.Now()
	res, err := db.Exec(
		`UPDATE sighting SET closed_at = ? WHERE id = ?`,
		&now, sighting.Id)
	if err != nil {
		return nil, err
	}
	sighting.ClosedAt = &now
	return res, nil
}
func UpdateSightingCallsignTx(tx *sql.Tx, sighting *Sighting, callsign string) (sql.Result, error) {
	res, err := tx.Exec("UPDATE sighting SET callsign = ? WHERE id = ?", callsign, sighting.Id)
	if err != nil {
		return nil, err
	}
	return res, nil
}
func UpdateSightingSquawkTx(tx *sql.Tx, sighting *Sighting, squawk string) (sql.Result, error) {
	return tx.Exec("UPDATE sighting SET squawk = ? WHERE id = ?", squawk, sighting.Id)
}
func CloseSightingBatch(db *sqlx.DB, sightings []*Sighting) error {
	if len(sightings) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	n := len(sightings)
	closedAt := time.Now()
	argPlaceholders := strings.Repeat("?,", n-1) + "?"
	args := make([]interface{}, 0, 1+n)
	args = append(args, closedAt)
	for i := 0; i < n; i++ {
		args = append(args, sightings[i].Id)
	}
	res, err := tx.Exec(
		`UPDATE sighting SET closed_at = ? WHERE id in (`+argPlaceholders+`)`,
		args...)
	if err != nil {
		return err
	}
	err = CheckRowsUpdated(res, int64(n))
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		sightings[i].ClosedAt = &closedAt
	}
	return nil
}
func LoadSightingById(db *sqlx.DB, sightingId int64) (*Sighting, error) {
	// index: aircraft_id, collection_session_id
	row := db.QueryRowx("SELECT * FROM sighting WHERE id = ?", sightingId)
	sighting := &Sighting{}
	err := row.StructScan(sighting)
	if err != nil {
		return nil, err
	}
	return sighting, nil
}

func CreateNewSightingCallSignTx(db *sql.Tx, sighting *Sighting, callsign string, observedAt time.Time) (sql.Result, error) {
	return db.Exec(`INSERT INTO sighting_callsign (sighting_id, callsign, observed_at)
				VALUES (?,?,?)`, sighting.Id, callsign, observedAt)
}

func InsertSightingLocation(db *sqlx.DB, sightingId uint64, t time.Time, altitude int64, lat float64, long float64) (sql.Result, error) {
	// index: aircraft_id, collection_session_id
	return db.Exec(`INSERT INTO sighting_location (sighting_id, timestamp, altitude, latitude, longitude)
		VALUES (?,?,?,?,?)`, sightingId, t, altitude, lat, long)
}

func GetLocationHistory(db *sqlx.DB, sighting *Sighting, lastId int64, batchSize int64) (*sqlx.Rows, error) {
	res, err := db.Queryx("SELECT * FROM sighting_location WHERE sighting_id = ? AND id > ? ORDER BY id ASC LIMIT ?", sighting.Id, lastId, batchSize)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}
func GetLocationHistoryWalkBatch(db *sqlx.DB, sighting *Sighting, batchSize int64, f func([]SightingLocation)) error {
	lastId := int64(-1)
	batch := make([]SightingLocation, 0, batchSize)
	more := true
	for more {
		more = false
		// res - needs closing
		res, err := GetLocationHistory(db, sighting, lastId, batchSize)
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
func GetFullLocationHistory(db *sqlx.DB, sighting *Sighting, batchSize int64) ([]SightingLocation, error) {
	var h []SightingLocation
	err := GetLocationHistoryWalkBatch(db, sighting, batchSize, func(location []SightingLocation) {
		h = append(h, location...)
	})
	if err != nil {
		return nil, err
	}
	return h, nil
}
func CreateNewSightingSquawkTx(db *sql.Tx, sighting *Sighting, squawk string, observedAt time.Time) (sql.Result, error) {
	return db.Exec(`INSERT INTO sighting_squawk (sighting_id, squawk, observed_at)
				VALUES (?,?,?)`, sighting.Id, squawk, observedAt)
}
func LoadSightingKml(db *sqlx.DB, sighting *Sighting) (*SightingKml, error) {
	row := db.QueryRowx("SELECT * FROM sighting_kml WHERE sighting_id = ?", sighting.Id)
	sightingKml := &SightingKml{}
	err := row.StructScan(sightingKml)
	if err != nil {
		return nil, err
	}
	return sightingKml, nil
}
func UpdateSightingKml(db *sqlx.DB, sightingKml *SightingKml, kmlStr string) (sql.Result, error) {
	isSameKml := sightingKml.Kml == kmlStr
	if isSameKml {
		log.Warnf("sighting %d sighting_kml %d - tried to update kml with same content",
			sightingKml.SightingId, sightingKml.Id)
	}
	res, err := db.Exec("UPDATE sighting_kml SET kml = ? WHERE id = ?", kmlStr, sightingKml.Id)
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
func CreateSightingKml(db *sqlx.DB, sighting *Sighting, kmlData string) (sql.Result, error) {
	return db.Exec("INSERT INTO sighting_kml (sighting_id, content_type, kml) VALUES (?, ?, ?)", sighting.Id, KmlPlainTextContentType, kmlData)
}

func CreateEmailJobTx(tx *sql.Tx, createdAt time.Time, content []byte) (sql.Result, error) {
	return tx.Exec("INSERT INTO email_v2 (status, retries, created_at, updated_at, retry_after, job) VALUES (?, ?, ?, ?, ?, ?)",
		EmailPending, 0, createdAt, createdAt, nil, content)
}

func GetPendingEmailJobs(db *sqlx.DB, now time.Time) ([]Email, error) {
	var jobs []Email
	rows, err := db.Queryx(`SELECT * FROM email_v2 where status = ? and (retry_after is NULL OR retry_after <= ?) ORDER BY id DESC`, EmailPending, now)
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

func DeleteCompletedEmail(tx *sql.Tx, job Email) (sql.Result, error) {
	return tx.Exec("DELETE FROM email_v2 WHERE id = ?", job.Id)
}
func MarkEmailFailedTx(tx *sql.Tx, job Email) (sql.Result, error) {
	return tx.Exec("UPDATE email_v2 SET retry_after = 0, status = ? WHERE id = ?", EmailFailed, job.Id)
}
func RetryEmailAfter(tx *sql.Tx, job Email, retryAfter time.Time) (sql.Result, error) {
	return tx.Exec("UPDATE email_v2 SET retry_after = ?, retries = ? WHERE id = ?", retryAfter, job.Retries+1, job.Id)
}
