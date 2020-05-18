package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/email"
	"github.com/afk11/airtrack/pkg/fs"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/afk11/airtrack/pkg/geo/openaip"
	"github.com/afk11/airtrack/pkg/kml"
	"github.com/afk11/airtrack/pkg/mailer"
	"github.com/afk11/airtrack/pkg/tracker"
	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mail.v2"
	"io/ioutil"
	"time"
)

type Context struct {
	Debug  bool
	Config *config.Config
}
type NearestAirport struct {
	Lat float64 `help:"latitude"`
	Lon float64 `help:"longitude"`
}

func (na *NearestAirport) Run(ctx *Context) error {
	fmt.Printf("%f %f\n", na.Lat, na.Lon)

	cfg := ctx.Config
	if len(cfg.Airports.Directories) == 0 {
		return errors.New("no airport directories configured")
	}

	nearestAirports := geo.NewNearestAirportGeocoder(tracker.DefaultGeoHashLength)
	files, err := fs.ScanDirectoriesForFiles("aip", cfg.Airports.Directories)
	for _, file := range files {
		airports, err := openaip.ReadAirportsFromFile(file)
		if err != nil {
			return errors.Wrapf(err, "error reading openaip file: %s", file)
		}
		err = nearestAirports.Register(airports)
		if err != nil {
			return errors.Wrapf(err, "error reading openaip file: %s", file)
		}
		log.Infof("found %d airports in file %s", len(airports), file)
	}

	place, distance, err := nearestAirports.ReverseGeocode(na.Lat, na.Lon)
	if err != nil {
		return err
	} else if place == "" {
		fmt.Printf("place not found")
	} else {
		fmt.Printf("place: %s\n", place)
		fmt.Printf("distance: %f\n", distance)
	}
	return nil
}

type OpenAipAirportStats struct {
	File string `help:"File path to openaip file"`
}

func (o *OpenAipAirportStats) Run(ctx *Context) error {
	fmt.Printf("%s\n", o.File)

	contents, err := ioutil.ReadFile(o.File)
	if err != nil {
		return errors.Wrapf(err, "reading openaip file")
	}
	f, err := openaip.Parse(contents)
	if err != nil {
		return errors.Wrapf(err, "parsing openaip file")
	}

	airportTypeToCount := make(map[string]int64)
	hasIcao := map[bool]int64{
		true:  0,
		false: 0,
	}
	for _, airport := range f.Waypoints.Airports {
		_, ok := airportTypeToCount[airport.Type]
		if !ok {
			airportTypeToCount[airport.Type] = 0
		}
		airportTypeToCount[airport.Type]++
		hasIcao[airport.Icao != ""]++
	}
	fmt.Println("TOTAL FOR EACH AIRPORT TYPE")
	for airportType, count := range airportTypeToCount {
		fmt.Printf("%s %d\n", airportType, count)
	}
	fmt.Println()

	fmt.Println("TOTAL WITH ICAO FIELD")
	fmt.Printf("%t %d\n", true, hasIcao[true])
	fmt.Printf("%t %d\n", false, hasIcao[false])
	fmt.Println()

	return nil
}

type EmptyKml struct {
}

func (e *EmptyKml) Run(ctx *Context) error {
	history := []db.SightingLocation{
		{
			Altitude:  100,
			Latitude:  100,
			Longitude: 100,
		},
	}
	w := kml.NewWriter(kml.WriterOptions{
		RouteName:        "Route",
		RouteDescription: "Route description..",

		SourceName:        "Source",
		SourceDescription: "Source description..",

		DestinationName:        "Destination",
		DestinationDescription: "Destination description..",
	})
	w.Add(history)
	_, _, data := w.Final()

	fmt.Printf("%s\n", data)
	return nil
}

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
		return errors.Wrapf(err, "Invalid timezone", ctx.Config.TimeZone)
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
		err := db.GetLocationHistoryWalkBatch(dbConn, sighting, tracker.LocationFetchBatchSize, func(location []db.SightingLocation) {
			w.Add(location)
			numPoints += len(location)
		})
		if err != nil {
			return err
		}
		firstLocation, lastLocation, kmlStr := w.Final()
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

var cli struct {
	Config           string              `help:"Configuration file path"`
	Email            TestEmail           `cmd help:"test email"`
	NearestAirport   NearestAirport      `cmd help:"find nearest airport"`
	AirportFileStats OpenAipAirportStats `cmd help:"print stats for openaip airport file"`
	EmptyKml         EmptyKml            `cmd help:"compare kml files"`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	cfg, err := config.ReadConfigFromFile(cli.Config)
	err = ctx.Run(&Context{Config: cfg})
	ctx.FatalIfErrorf(err)
}
