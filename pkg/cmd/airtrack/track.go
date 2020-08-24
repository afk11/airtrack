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
	dump1090 "github.com/afk11/airtrack/pkg/dump1090/acmap"
	"github.com/afk11/airtrack/pkg/fs"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/afk11/airtrack/pkg/geo/cup"
	"github.com/afk11/airtrack/pkg/geo/openaip"
	"github.com/afk11/airtrack/pkg/iso3166"
	"github.com/afk11/airtrack/pkg/mailer"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/afk11/airtrack/pkg/tar1090"
	"github.com/afk11/airtrack/pkg/tracker"
	smtp "github.com/afk11/mail"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

type TrackCmd struct {
	Verbosity  string `help:"Log level panic, fatal, error, warn, info, debug, trace)" default:"warn"`
	CPUProfile string `help:"Write CPU profile to file"`
}

func (c *TrackCmd) Run(ctx *Context) error {
	level, err := log.ParseLevel(c.Verbosity)
	if err != nil {
		return errors.Wrapf(err, "invalid log verbosity level")
	}
	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

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

	dbUrl, err := cfg.Database.DataSource(loc)
	if cfg.Database.Driver != config.DatabaseDriverMySQL && cfg.Database.Driver != config.DatabaseDriverSqlite3 {
		return errors.New("postgresql not yet supported")
	}

	if err != nil {
		return err
	}
	dbConn, err := sqlx.Connect(cfg.Database.Driver, dbUrl)
	if err != nil {
		return err
	}
	dialect := goqu.Dialect(cfg.Database.Driver)
	database := db.NewDatabase(dbConn, dialect)

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

			dialer := smtp.NewDialer(
				settings.Host, settings.Port, settings.Username, settings.Password)
			if settings.TLS {
				dialer.SSL = true
			}
			if settings.NoStartTLS {
				dialer.StartTLSPolicy = smtp.NoStartTLS
			} else if settings.MandatoryStartTLS {
				dialer.StartTLSPolicy = smtp.MandatoryStartTLS
			} else {
				dialer.StartTLSPolicy = smtp.OpportunisticStartTLS
			}
			dialer.Timeout = time.Second * 30
			m := mailer.NewMailer(database, settings.Sender, dialer, aesgcm)
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

	var airportFiles int
	var airportsFound int
	if len(cfg.Airports.OpenAIPDirectories) > 0 {
		files, err := fs.ScanDirectoriesForFiles("aip", cfg.Airports.OpenAIPDirectories)
		if err != nil {
			log.Fatalf("error scanning openaip directories: %s", err.Error())
		}
		for _, file := range files {
			openaipFile, err := openaip.ParseFile(file)
			if err != nil {
				return errors.Wrapf(err, "error reading openaip file: %s", file)
			}
			acRecords, err := openaip.ExtractOpenAIPRecords(openaipFile)
			if err != nil {
				return errors.Wrapf(err, "converting openaip record: %s", file)
			}
			err = nearestAirports.Register(acRecords)
			if err != nil {
				return errors.Wrapf(err, "registering airports from openaip file: %s", file)
			}
			log.Debugf("found %d airports in openaip file %s", len(acRecords), file)
			airportsFound += len(acRecords)
		}
		airportFiles += len(files)
	}
	if len(cfg.Airports.CupDirectories) > 0 {
		files, err := fs.ScanDirectoriesForFiles("cup", cfg.Airports.CupDirectories)
		if err != nil {
			log.Fatalf("error scanning cup directories: %s", err.Error())
		}
		for _, file := range files {
			cupRecords, err := cup.ParseFile(file)
			if err != nil {
				return errors.Wrapf(err, "error reading openaip file: %s", file)
			}
			acRecords, err := openaip.ExtractCupRecords(cupRecords)
			if err != nil {
				return errors.Wrapf(err, "extracting cup records: %s", file)
			}
			err = nearestAirports.Register(acRecords)
			if err != nil {
				return errors.Wrapf(err, "registering airports from cup file: %s", file)
			}
			log.Debugf("found %d airports in openaip file %s", len(acRecords), file)
			airportsFound += len(acRecords)
		}
		airportFiles += len(files)
	}
	if airportFiles > 0 {
		log.Infof("found %d airports in %d files", airportsFound, airportFiles)
	}

	opt.AirportGeocoder = nearestAirports

	countryCodesData, err := asset.Asset("assets/iso3166_country_codes.txt")
	if err != nil {
		panic(err)
	}
	countryCodeRows, err := iso3166.ParseColumnFormat(bytes.NewBuffer(countryCodesData))
	if err != nil {
		panic(err)
	}
	countryCodeStore, err := iso3166.New(countryCodeRows)
	if err != nil {
		panic(err)
	}

	icaoAllocationsData, err := asset.Asset("assets/icao_country_aircraft_allocation.txt")
	if err != nil {
		panic(err)
	}
	icaoAllocations, err := ccode.LoadCountryAllocations(bytes.NewBuffer(icaoAllocationsData), countryCodeStore)
	if err != nil {
		return err
	}

	opt.CountryCodes = countryCodeStore
	opt.Allocations = icaoAllocations

	msgs := make(chan *pb.Message)
	adsbxEndpoint := tracker.DefaultAdsbxEndpoint
	var adsbxApiKey string
	if cfg.AdsbxConfig.ApiUrl != "" {
		adsbxEndpoint = cfg.AdsbxConfig.ApiUrl
	}
	if cfg.AdsbxConfig.ApiKey != "" {
		adsbxApiKey = cfg.AdsbxConfig.ApiKey
	}
	p := tracker.NewAdsbxProducer(msgs, adsbxEndpoint, adsbxApiKey)

	t, err := tracker.New(database, opt)
	if err != nil {
		return err
	}

	if cfg.MapSettings != nil && cfg.MapSettings.Disabled == false {
		historyFiles := tracker.DefaultHistoryFileCount
		if cfg.MapSettings.HistoryCount != 0 {
			historyFiles = cfg.MapSettings.HistoryCount
		}
		m, err := tracker.NewAircraftMap(cfg.MapSettings)
		if err != nil {
			return err
		}
		mapsToUse := tracker.DefaultMapServices
		if cfg.MapSettings.Services != nil {
			mapsToUse = cfg.MapSettings.Services
		}
		for _, mapService := range mapsToUse {
			switch mapService {
			case tracker.Dump1090MapService:
				err = m.RegisterMapService(dump1090.NewDump1090Map(m))
			case tracker.Tar1090MapService:
				err = m.RegisterMapService(tar1090.NewTar1090Map(m, historyFiles))
			default:
				return errors.New("unsupported map service: " + mapService)
			}
			if err != nil {
				return errors.Wrapf(err, "registering map service (%s)", mapService)
			}
		}
		err = t.RegisterProjectStatusListener(tracker.NewMapProjectStatusListener(m))
		if err != nil {
			return err
		}
		err = t.RegisterProjectAircraftUpdateListener(tracker.NewMapProjectAircraftUpdateListener(m))
		if err != nil {
			return err
		}
		go m.Serve()
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

	if c.CPUProfile != "" {
		f, err := os.Create(c.CPUProfile)
		if err != nil {
			return err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
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
