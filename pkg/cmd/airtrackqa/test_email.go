package airtrackqa

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/email"
	"github.com/afk11/airtrack/pkg/kml"
	"github.com/afk11/airtrack/pkg/mailer"
	"github.com/afk11/airtrack/pkg/tracker"
	"github.com/pkg/errors"
	"gopkg.in/mail.v2"
	"time"
)

type TestEmail struct {
	To       string `help:"send test email to"`
	Type     string `help:"notification type"`
	Sighting int64  `help:"sighting ID"`
}

func (e *TestEmail) Run(ctx *Context) error {
	if e.To == "" {
		return errors.New("missing to email address")
	} else if e.Type == "" {
		return errors.New("missing to email notification type")
	} else if e.Sighting == 0 {
		return errors.New("missing sighting id")
	}
	cfg := ctx.Config
	key, err := base64.StdEncoding.DecodeString(cfg.Encryption.Key)
	if err != nil {
		return errors.Wrap(err, "decoding encryption.key base64")
	} else if len(key) != 32 {
		return errors.New("encryption.key should be base64 encoding of 32 random bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(cfg.TimeZone)
	if err != nil {
		return errors.Wrapf(err, "invalid timezone %s", ctx.Config.TimeZone)
	}

	dbConn, err := db.NewConn(cfg.Database.Driver, cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database, loc)
	if err != nil {
		return err
	}

	settings := cfg.EmailSettings.Smtp
	dialer := mail.NewDialer(
		settings.Host, settings.Port, settings.Username, settings.Password)
	if settings.MandatoryStartTLS {
		dialer.StartTLSPolicy = mail.MandatoryStartTLS
	}
	m := mailer.NewMailer(dbConn, settings.Sender, dialer, aesgcm)
	m.Start()
	defer m.Stop()
	tpls, err := email.LoadMailTemplates(email.GetTemplates()...)
	if err != nil {
		return err
	}

	sighting, err := db.LoadSightingById(dbConn, e.Sighting)
	if err != nil {
		return err
	}
	ac, err := db.LoadAircraftById(dbConn, int64(sighting.AircraftId))
	if err != nil {
		return err
	}

	notifyType, err := tracker.EmailNotificationFromString(e.Type)
	if err != nil {
		return err
	}

	var job *db.EmailJob
	switch notifyType {
	case tracker.MapProduced:
		w := kml.NewWriter(kml.WriterOptions{
			RouteName:        "Route",
			RouteDescription: "Route description..",

			SourceName:        "Source",
			SourceDescription: "Source description..",

			DestinationName:        "Destination",
			DestinationDescription: "Destination description..",
		})

		var numPoints int
		var firstLocation, lastLocation *db.SightingLocation
		err := db.GetLocationHistoryWalkBatch(dbConn, sighting, tracker.LocationFetchBatchSize, func(location []db.SightingLocation) {
			w.Write(location)
			if firstLocation == nil {
				firstLocation = &location[0]
			}
			lastLocation = &location[len(location)-1]
			numPoints += len(location)
		})
		if err != nil {
			return err
		}
		kmlStr, err := w.Final()
		if err != nil {
			return err
		}
		firstSeen := sighting.CreatedAt
		lastSeen := time.Now()
		if sighting.ClosedAt != nil {
			lastSeen = *sighting.ClosedAt
		}
		job, err = email.PrepareMapProducedEmail(tpls, e.To, kmlStr, email.MapProducedParameters{
			Project:      "TESTEMAIL",
			Icao:         ac.Icao,
			CallSign:     *sighting.CallSign,
			StartTime:    firstSeen,
			EndTime:      lastSeen,
			DurationFmt:  lastSeen.Sub(firstSeen).String(),
			StartTimeFmt: firstSeen.Format(time.RFC1123Z),
			EndTimeFmt:   lastSeen.Format(time.RFC1123Z),
			StartLocation: email.Location{
				Latitude:  firstLocation.Latitude,
				Longitude: firstLocation.Longitude,
				Altitude:  firstLocation.Altitude,
			},
			EndLocation: email.Location{
				Latitude:  lastLocation.Latitude,
				Longitude: lastLocation.Longitude,
				Altitude:  lastLocation.Altitude,
			},
			MapUpdated: true,
		})
		if err != nil {
			return err
		}
	case tracker.SpottedInFlight:
		history, err := db.GetFullLocationHistory(dbConn, sighting, tracker.LocationFetchBatchSize)
		if err != nil {
			return err
		}
		if len(history) < 1 {
			return errors.New("no history for flight")
		}
		firstLocation := history[0]
		firstSeen := sighting.CreatedAt
		job, err = email.PrepareSpottedInFlightEmail(tpls, e.To, email.SpottedInFlightParameters{
			Project:      "TESTEMAIL",
			Icao:         ac.Icao,
			CallSign:     *sighting.CallSign,
			StartTime:    firstSeen,
			StartTimeFmt: firstSeen.Format(time.RFC1123Z),
			StartLocation: email.Location{
				Latitude:  firstLocation.Latitude,
				Longitude: firstLocation.Longitude,
				Altitude:  firstLocation.Altitude,
			},
		})
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown email type")
	}

	if err != nil {
		return err
	}
	err = m.Queue(*job)
	if err != nil {
		return err
	}
	return nil
}
