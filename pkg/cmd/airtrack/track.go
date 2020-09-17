package airtrack

import (
	"bytes"
	"fmt"
	"github.com/afk11/airtrack/pkg/aircraft/ccode"
	"github.com/afk11/airtrack/pkg/airports"
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
	"github.com/afk11/airtrack/pkg/readsb"
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
	loc, err := cfg.GetTimeLocation()
	if err != nil {
		return err
	}

	dbUrl, err := cfg.Database.DataSource(loc)
	if err != nil {
		return err
	}
	dbConn, err := sqlx.Connect(cfg.Database.Driver, dbUrl)
	if err != nil {
		return err
	}
	if cfg.Database.Driver == config.DatabaseDriverSqlite3 {
		dbConn.SetMaxOpenConns(1)
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

	var mailSender *mailer.Mailer
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
			mailSender = mailer.NewMailer(database, settings.Sender, dialer)
			opt.Mailer = mailSender
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
	useBuiltinAirports := cfg.Airports == nil || cfg.Airports.DisableBuiltInAirports == false
	if useBuiltinAirports {
		for _, file := range airports.AssetNames() {
			d, err := airports.Asset(file)
			if err != nil {
				return errors.Wrapf(err, "reading built-in airport file")
			}
			openaipFile, err := openaip.Parse(d)
			if err != nil {
				return errors.Wrapf(err, "error reading built-in airport file: %s", file)
			}
			acRecords, err := openaip.ExtractOpenAIPRecords(openaipFile)
			if err != nil {
				return errors.Wrapf(err, "converting aircraft record from built-in %s", file)
			}
			err = nearestAirports.Register(acRecords)
			if err != nil {
				return errors.Wrapf(err, "registering airports from openaip file: %s", file)
			}
			log.Debugf("found %d airports in built-in airport file %s", len(acRecords), file)
			airportsFound += len(acRecords)
		}
	}

	if cfg.Airports != nil && len(cfg.Airports.OpenAIPDirectories) > 0 {
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
	if cfg.Airports != nil && len(cfg.Airports.CupDirectories) > 0 {
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
	var producers []tracker.Producer
	if cfg.AdsbxConfig != nil {
		var adsbxEndpoint = tracker.DefaultAdsbxEndpoint
		var adsbxApiKey string
		if cfg.AdsbxConfig.ApiUrl != "" {
			adsbxEndpoint = cfg.AdsbxConfig.ApiUrl
		}
		if cfg.AdsbxConfig.ApiKey != "" {
			adsbxApiKey = cfg.AdsbxConfig.ApiKey
		}
		producers = append(producers, tracker.NewAdsbxProducer(msgs, adsbxEndpoint, adsbxApiKey))
	}
	if len(cfg.Beast) > 0 {
		// readsb library housekeeping
		readsb.IcaoFilterInit()
		readsb.ModeACInit()
		readsb.ModesChecksumInit(1)
		go func() {
			for {
				<-time.After(time.Minute)
				readsb.IcaoFilterExpire()
			}
		}()
		for _, bcfg := range cfg.Beast {
			producers = append(producers, tracker.NewBeastProducer(msgs, bcfg.Host, bcfg.Port))
		}
	}

	t, err := tracker.New(database, opt)
	if err != nil {
		return err
	}

	var mapServer *tracker.AircraftMap
	if cfg.MapSettings != nil && cfg.MapSettings.Disabled == false {
		historyFiles := tracker.DefaultHistoryFileCount
		if cfg.MapSettings.HistoryCount != 0 {
			historyFiles = cfg.MapSettings.HistoryCount
		}
		mapServer, err = tracker.NewAircraftMap(cfg.MapSettings)
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
				err = mapServer.RegisterMapService(dump1090.NewDump1090Map(mapServer))
			case tracker.Tar1090MapService:
				err = mapServer.RegisterMapService(tar1090.NewTar1090Map(mapServer, historyFiles))
			default:
				return errors.New("unsupported map service: " + mapService)
			}
			if err != nil {
				return errors.Wrapf(err, "registering map service (%s)", mapService)
			}
		}
		err = t.RegisterProjectStatusListener(tracker.NewMapProjectStatusListener(mapServer))
		if err != nil {
			return err
		}
		err = t.RegisterProjectAircraftUpdateListener(tracker.NewMapProjectAircraftUpdateListener(mapServer))
		if err != nil {
			return err
		}
		mapServer.Serve()
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
	if mailSender != nil {
		mailSender.Start()
	}
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

	for _, producer := range producers {
		log.Infof("starting %s producer..", producer.Name())
		producer.Start()
	}

	select {
	case <-gracefulStop:
		log.Infof("graceful stop received")
		for _, producer := range producers {
			log.Infof("stopping %s producer..", producer.Name())
			producer.Stop()
		}
		close(msgs)
		log.Info("stopping tracker")
		err = t.Stop()
		if err != nil {
			return err
		}

		if mapServer != nil {
			log.Info("stopping map server")
			err = mapServer.Stop()
			if err != nil {
				return err
			}
		}
		if mailSender != nil {
			log.Info("stopping mailer")
			mailSender.Stop()
		}
		return nil
	}
}
