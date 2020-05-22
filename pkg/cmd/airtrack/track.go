package airtrack

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"github.com/afk11/airtrack/pkg/aircraft/ccode"
	asset "github.com/afk11/airtrack/pkg/assets"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/fs"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/afk11/airtrack/pkg/geo/openaip"
	"github.com/afk11/airtrack/pkg/iso3166"
	"github.com/afk11/airtrack/pkg/mailer"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/afk11/airtrack/pkg/tracker"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mail.v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type TrackCmd struct {
	Verbosity string `help:"Log level panic, fatal, error, warn, info, debug, trace)" default:"warn"`
}

func (c *TrackCmd) Run(ctx *Context) error {
	level, err := log.ParseLevel(c.Verbosity)
	if err != nil {
		return errors.Wrapf(err, "invalid log verbosity level")
	}
	log.SetLevel(level)

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	var gracefulReload = make(chan os.Signal)
	signal.Notify(gracefulReload, syscall.SIGHUP)

	cfg := ctx.Config
	loc, err := time.LoadLocation(ctx.Config.TimeZone)
	if err != nil {
		return errors.Wrapf(err, "Invalid timezone `%s`", ctx.Config.TimeZone)
	}
	dbConn, err := db.NewConn(cfg.Database.Driver, cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database, loc)
	if err != nil {
		return err
	}

	opt := tracker.Options{
		Workers:                   64,
		SightingTimeout:           time.Second * 60,
		OnGroundUpdateThreshold:   tracker.DefaultOnGroundUpdateThreshold,
		NearestAirportMaxDistance: tracker.DefaultNearestAirportMaxDistance,
		NearestAirportMaxAltitude: tracker.DefaultNearestAirportMaxAltitude,
	}
	if cfg.Sighting.Timeout != nil {
		opt.SightingTimeout = time.Second * time.Duration(*cfg.Sighting.Timeout)
	}
	if cfg.Sighting.OnGroundUpdateThreshold != nil {
		opt.OnGroundUpdateThreshold = *cfg.Sighting.OnGroundUpdateThreshold
	}

	if cfg.Encryption.Key == "" {
		return errors.New("encryption.key not set or empty")
		//return errors.Wrap(err, "")
	}

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

	if cfg.EmailSettings != nil {
		switch cfg.EmailSettings.Driver {
		case config.MailDriverSmtp:
			settings := cfg.EmailSettings.Smtp
			if settings == nil {
				return errors.New("email.driver is smtp but missing email.smtp configuration")
			} else if settings.Sender == "" {
				return errors.New("email.sender not set")
			}

			dialer := mail.NewDialer(
				settings.Host, settings.Port, settings.Username, settings.Password)
			if settings.MandatoryStartTLS {
				dialer.StartTLSPolicy = mail.MandatoryStartTLS
			}
			m := mailer.NewMailer(dbConn, settings.Sender, dialer, aesgcm)
			m.Start()
			opt.Mailer = m
			defer func() {
				m.Stop()
				log.Info("stopped mailer")
			}()
		default:
			return errors.New("unknown email driver")
		}
		log.Infof("using %s mailer", cfg.EmailSettings.Driver)
	} else {
		log.Info("no mailer configured")
	}

	nearestAirports := geo.NewNearestAirportGeocoder(tracker.DefaultGeoHashLength)

	if len(cfg.Airports.Directories) > 0 {
		files, err := fs.ScanDirectoriesForFiles("aip", cfg.Airports.Directories)
		if err != nil {
			log.Fatalf("error scanning airport location directories: %s", err.Error())
		}
		var totalAirports int
		for _, file := range files {
			openaipFile, err := openaip.ParseFile(file)
			if err != nil {
				return errors.Wrapf(err, "error reading openaip file: %s", file)
			}
			acRecords, err := openaip.ExtractAirports(openaipFile)
			if err != nil {
				return errors.Wrapf(err, "converting openaip record: %s", file)
			}
			err = nearestAirports.Register(acRecords)
			if err != nil {
				return errors.Wrapf(err, "error reading openaip file: %s", file)
			}
			log.Debugf("found %d openaipFile in file %s", len(acRecords), file)
			totalAirports += len(acRecords)
		}
		log.Infof("found %d airports in %d files", totalAirports, len(files))
	}
	opt.AirportGeocoder = nearestAirports

	countryCodesData, err := asset.Asset("assets/iso3166_country_codes.txt")
	if err != nil {
		panic(err)
	}

	countryCodeStore, err := iso3166.ParseColumnFormat(bytes.NewBuffer(countryCodesData))
	if err != nil {
		panic(err)
	}

	icaoAllocationsData, err := asset.Asset("assets/icao_country_aircraft_allocation.txt")
	if err != nil {
		panic(err)
	}

	icaoAllocations, err := ccode.LoadCountryAllocations(bytes.NewBuffer(icaoAllocationsData), countryCodeStore)
	if err != nil {
		panic(err)
	}

	opt.CountryCodes = countryCodeStore
	opt.Allocations = icaoAllocations

	msgs := make(chan *pb.Message)
	p := tracker.NewAdsbxProducer(msgs)

	t, err := tracker.New(dbConn, opt)
	if err != nil {
		return err
	}
	var ignored int32
	for _, proj := range cfg.Projects {
		if proj.Disabled {
			ignored++
			continue
		}
		p, err := tracker.InitProject(proj)
		if err != nil {
			return errors.Wrap(err, "failed to init project")
		}
		err = t.AddProject(p)
		if err != nil {
			return errors.Wrap(err, "failed to add project to tracker")
		}
		log.Debugf("init project %s", p.Name)
	}
	if ignored > 0 {
		log.Debugf("skipping %d disabled projects", ignored)
	}

	t.Start(msgs)

	if cfg.Metrics != nil && cfg.Metrics.Enabled {
		var prometheusPort = 9206
		if cfg.Metrics.Port != 0 {
			prometheusPort = cfg.Metrics.Port
		}
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			err = http.ListenAndServe(fmt.Sprintf(":%d", prometheusPort), nil)
			if err != nil {
				panic(err)
			}
		}()
	}

	p.Start()

	select {
	case <-gracefulStop:
		p.Stop()
		err = t.Stop()
		if err != nil {
			return err
		}
		return nil
	}
}
