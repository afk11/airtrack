package db

import (
	"bytes"
	"compress/gzip"
	"github.com/pkg/errors"
	"io/ioutil"
	"time"
)

// Values for Email status. Completed Email jobs are deleted.
const (
	// EmailPending - status of a pending email
	EmailPending = 1
	// EmailFailed - status of a failed email
	EmailFailed = 0
)

// Values for SightingKml content type
const (
	// KmlPlainTextContentType - content type used when
	// sighting_kml kml record is encoded in plain text KML
	KmlPlainTextContentType = 0
	// KmlGzipContentType - content type used when
	// sighting_kml kml record is gzipped KML
	KmlGzipContentType = 1
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
