package airtrackqa

import (
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/email"
	"github.com/afk11/airtrack/pkg/kml"
	"github.com/afk11/airtrack/pkg/mailer"
	"github.com/afk11/airtrack/pkg/tracker"
	"github.com/afk11/mail"
	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"time"
)

// TestEmail - sends a test email for a certain sighting
type TestEmail struct {
	To       string `help:"send test email to"`
	Type     string `help:"notification type"`
	Sighting uint64 `help:"sighting ID"`
}

// Run - attempts to send email for a certain sighting
func (e *TestEmail) Run(ctx *Context) error {
	if e.To == "" {
		return errors.New("missing to email address")
	} else if e.Type == "" {
		return errors.New("missing to email notification type")
	} else if e.Sighting == 0 {
		return errors.New("missing sighting id")
	}
	cfg := ctx.Config

	loc, err := cfg.GetTimeLocation()
	if err != nil {
		return err
	}

	dbURL, err := cfg.Database.DataSource(loc)
	if err != nil {
		return err
	}
	dbConn, err := sqlx.Connect(cfg.Database.Driver, dbURL)
	if err != nil {
		return err
	}
	database := db.NewDatabase(dbConn, goqu.Dialect(cfg.Database.Driver))
	settings := cfg.EmailSettings.SMTP
	dialer := mail.NewDialer(
		settings.Host, settings.Port, settings.Username, settings.Password)
	if settings.MandatoryStartTLS {
		dialer.StartTLSPolicy = mail.MandatoryStartTLS
	}
	m := mailer.NewMailer(database, settings.Sender, dialer)
	m.Start()
	defer m.Stop()
	tpls, err := email.LoadMailTemplates(email.GetTemplates()...)
	if err != nil {
		return err
	}

	sighting, err := database.GetSightingByID(e.Sighting)
	if err != nil {
		return err
	}
	ac, err := database.GetAircraftByID(sighting.AircraftID)
	if err != nil {
		return err
	}

	notifyType, err := tracker.EmailNotificationFromString(e.Type)
	if err != nil {
		return err
	}

	var job *mailer.EmailJob
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
		err := database.WalkLocationHistoryBatch(sighting, 50, func(location []db.SightingLocation) {
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
		kmlBytes := []byte(kmlStr)
		firstSeen := sighting.CreatedAt
		lastSeen := time.Now()
		if sighting.ClosedAt != nil {
			lastSeen = *sighting.ClosedAt
		}
		job, err = email.PrepareMapProducedEmail(tpls, e.To, kmlBytes, email.MapProducedParameters{
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
		history, err := database.GetFullLocationHistory(sighting, 50)
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
