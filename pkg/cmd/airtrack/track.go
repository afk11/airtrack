package airtrack

import (
	"bytes"
	"context"
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
	"github.com/afk11/airtrack/pkg/readsb/aircraft_db"
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
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

// TrackCmd - aircraft tracking task
type TrackCmd struct {
	// Config - aircraft configuration file path
	Config string `help:"Configuration file path"`
	// Projects - List of project configuration files
	Projects []string `help:"Projects configuration file (may be repeated, and in addition to main configuration file)"`
	// Verbosity - log level to use
	Verbosity string `help:"Log level panic, fatal, error, warn, info, debug, trace)" default:"warn"`
	// CPUProfile - will start CPU profile and write result to this file path
	CPUProfile string `help:"Write CPU profile to file"`
	// HeapProfile - will run heap profiler every 10 seconds and write results to
	// files with this prefix suffixed by a counter.
	HeapProfile string `help:"Write heap profile to file"`
}

// Run - the command line entry point for TrackCmd
func (c *TrackCmd) Run() error {
	var stopSignal = make(chan os.Signal)
	var reloadSignal = make(chan os.Signal)
	signal.Notify(stopSignal, syscall.SIGTERM)
	signal.Notify(stopSignal, syscall.SIGINT)
	signal.Notify(reloadSignal, syscall.SIGHUP)

	for {
		loader := Loader{}
		err := loader.Load(c)
		if err != nil {
			return errors.Wrapf(err, "during initialization")
		}
		err = loader.Start()
		if err != nil {
			return errors.Wrapf(err, "during startup")
		}
		select {
		case sig := <-stopSignal:
			log.Infof("stop signal received - %s", sig.String())
			return loader.Stop()
		case <-reloadSignal:
			log.Infof("reload signal received")
			err = loader.Stop()
			if err != nil {
				return errors.Wrapf(err, "error stopping during reload")
			}
		}
	}
}

// Loader takes care of initializing dependencies for airtrack
// and processing the configuration. It is not intended to be reused
// for multiple runs.
type Loader struct {
	cfg                  *config.Config
	location             *time.Location
	dbConn               *sqlx.DB
	options              *tracker.Options
	mailSender           *mailer.Mailer
	producers            []tracker.Producer
	mapServer            *tracker.AircraftMap
	metricsServer        *http.Server
	t                    *tracker.Tracker
	usingBeast           bool
	readsbInitialized    bool
	cpuProfileFile       *os.File
	heapProfileFileName  string
	heapProfileCanceller func()
	msgs                 chan *pb.Message

	icaoFilterExpirationCanceller func()
}

func (l *Loader) loadCleanup() error {
	if l.dbConn != nil {
		err := l.dbConn.Close()
		if err != nil {
			log.Warnf("failed to close db connection: %s", err)
		}
		l.dbConn = nil
	}
	if l.options != nil {
		l.options = nil
	}
	if l.cpuProfileFile != nil {
		err := l.cpuProfileFile.Close()
		if err != nil {
			log.Warnf("failed to close cpu profile file: %s", err)
		}
	}
	return nil
}

// ExtractOpenAipFile takes an .aip files data and registers the airports
func ExtractOpenAipFile(nearestAirports *geo.NearestAirportGeocoder, d []byte) (int, error) {
	openaipFile, err := openaip.Parse(d)
	if err != nil {
		return 0, errors.Wrapf(err, "parsing .aip file")
	}
	acRecords, err := openaip.ExtractOpenAIPRecords(openaipFile)
	if err != nil {
		return 0, errors.Wrapf(err, "extracting .aip file")
	}
	err = nearestAirports.Register(acRecords)
	if err != nil {
		return 0, errors.Wrapf(err, "registering airports")
	}
	return len(acRecords), nil
}

// Load loads and processes the configuration sets everything up
func (l *Loader) Load(c *TrackCmd) error {
	var err error
	defer func() {
		if err == nil {
			return
		}
		err = l.loadCleanup()
		if err != nil {
			panic(err)
		}
	}()

	l.cfg, err = config.ReadConfigs(c.Config, c.Projects)
	if err != nil {
		return err
	}

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

	l.location, err = l.cfg.GetTimeLocation()
	if err != nil {
		return errors.Wrapf(err, "loading timezone")
	}

	dbUrl, err := l.cfg.Database.DataSource(l.location)
	if err != nil {
		return errors.Wrapf(err, "creating database connection parameters")
	}
	l.dbConn, err = sqlx.Connect(l.cfg.Database.Driver, dbUrl)
	if err != nil {
		return errors.Wrapf(err, "creating database connection")
	}
	if l.cfg.Database.Driver == config.DatabaseDriverSqlite3 {
		l.dbConn.SetMaxOpenConns(1)
	}
	dialect := goqu.Dialect(l.cfg.Database.Driver)
	database := db.NewDatabase(l.dbConn, dialect)

	opt := tracker.Options{
		Workers:                   64,
		SightingTimeout:           time.Second * 60,
		OnGroundUpdateThreshold:   tracker.DefaultOnGroundUpdateThreshold,
		NearestAirportMaxDistance: tracker.DefaultNearestAirportMaxDistance,
		NearestAirportMaxAltitude: tracker.DefaultNearestAirportMaxAltitude,
	}
	if l.cfg.Sighting.Timeout != nil {
		opt.SightingTimeout = time.Second * time.Duration(*l.cfg.Sighting.Timeout)
	}

	if l.cfg.EmailSettings != nil {
		switch l.cfg.EmailSettings.Driver {
		case config.MailDriverSmtp:
			settings := l.cfg.EmailSettings.Smtp
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
			l.mailSender = mailer.NewMailer(database, settings.Sender, dialer)
			opt.Mailer = l.mailSender
		default:
			return errors.New("unknown email driver")
		}
		log.Infof("using %s mailer", l.cfg.EmailSettings.Driver)
	} else {
		log.Info("no mailer configured")
	}

	nearestAirports := geo.NewNearestAirportGeocoder(tracker.DefaultGeoHashLength)

	var airportFiles int
	var airportsFound int
	useBuiltinAirports := l.cfg.Airports == nil || l.cfg.Airports.DisableBuiltInAirports == false
	if useBuiltinAirports {
		airportAssetFiles := airports.AssetNames()
		airportFiles += len(airportAssetFiles)
		for i := range airportAssetFiles {
			d, err := airports.Asset(airportAssetFiles[i])
			if err != nil {
				return errors.Wrapf(err, "loading built-in airport file: %s", airportAssetFiles[i])
			}
			numAcRecords, err := ExtractOpenAipFile(nearestAirports, d)
			if err != nil {
				return errors.Wrapf(err, "processing built-in airport file: %s", airportAssetFiles[i])
			}
			log.Debugf("found %d airports in built-in airport file %s", numAcRecords, airportAssetFiles[i])
			airportsFound += numAcRecords
		}
	}

	if l.cfg.Airports != nil && len(l.cfg.Airports.OpenAIPDirectories) > 0 {
		files, err := fs.ScanDirectoriesForFiles("aip", l.cfg.Airports.OpenAIPDirectories)
		if err != nil {
			log.Fatalf("error scanning openaip directories: %s", err.Error())
		}
		airportFiles += len(files)
		for i := range files {
			d, err := ioutil.ReadFile(files[i])
			if err != nil {
				return errors.Wrapf(err, "loading openaip airport file: %s", files[i])
			}
			numAcRecords, err := ExtractOpenAipFile(nearestAirports, d)
			if err != nil {
				return errors.Wrapf(err, "processing openaip airport file: %s", files[i])
			}
			log.Debugf("found %d airports in openaip file %s", numAcRecords, files[i])
			airportsFound += numAcRecords
		}
	}
	if l.cfg.Airports != nil && len(l.cfg.Airports.CupDirectories) > 0 {
		files, err := fs.ScanDirectoriesForFiles("cup", l.cfg.Airports.CupDirectories)
		if err != nil {
			log.Fatalf("error scanning cup directories: %s", err.Error())
		}
		airportFiles += len(files)
		for i := range files {
			cupRecords, err := cup.ParseFile(files[i])
			if err != nil {
				return errors.Wrapf(err, "error reading openaip file: %s", files[i])
			}
			acRecords, err := cup.ExtractCupRecords(cupRecords)
			if err != nil {
				return errors.Wrapf(err, "extracting cup records: %s", files[i])
			}
			err = nearestAirports.Register(acRecords)
			if err != nil {
				return errors.Wrapf(err, "registering airports from cup file: %s", files[i])
			}
			log.Debugf("found %d airports in openaip file %s", len(acRecords), files[i])
			airportsFound += len(acRecords)
		}
	}
	if airportFiles > 0 {
		log.Infof("found %d airports in %d files", airportsFound, airportFiles)
	}

	opt.AirportGeocoder = nearestAirports

	countryCodesData, err := asset.Asset("assets/iso3166_country_codes.txt")
	if err != nil {
		return errors.Wrapf(err, "loading country codes file")
	}
	countryCodeRows, err := iso3166.ParseColumnFormat(bytes.NewBuffer(countryCodesData))
	if err != nil {
		return errors.Wrapf(err, "parsing country codes file")
	}
	countryCodeStore, err := iso3166.New(countryCodeRows)
	if err != nil {
		return errors.Wrapf(err, "creating country info store")
	}

	icaoAllocationsData, err := asset.Asset("assets/icao_country_aircraft_allocation.txt")
	if err != nil {
		return errors.Wrapf(err, "loading aircraft countries file")
	}
	icaoAllocations, err := ccode.LoadCountryAllocations(bytes.NewBuffer(icaoAllocationsData), countryCodeStore)
	if err != nil {
		return errors.Wrapf(err, "creating aircraft countries store")
	}

	opt.CountryCodes = countryCodeStore
	opt.Allocations = icaoAllocations

	if c.CPUProfile != "" {
		l.cpuProfileFile, err = os.Create(c.CPUProfile)
		if err != nil {
			return errors.Wrapf(err, "creating cpu profile file: %s", c.CPUProfile)
		}
	}

	if c.HeapProfile != "" {
		l.heapProfileFileName = c.HeapProfile
	}

	if l.cfg.Metrics != nil && l.cfg.Metrics.Enabled {
		var prometheusPort = 9206
		var prometheusIface = "0.0.0.0"
		if l.cfg.Metrics.Port != 0 {
			prometheusPort = l.cfg.Metrics.Port
		}
		if l.cfg.Metrics.Interface != "" {
			prometheusIface = l.cfg.Metrics.Interface
		}
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		l.metricsServer = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", prometheusIface, prometheusPort),
			Handler:      mux,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		}
	}

	l.msgs = make(chan *pb.Message)
	if l.cfg.AdsbxConfig != nil {
		var adsbxEndpoint = tracker.DefaultAdsbxEndpoint
		var adsbxApiKey string
		if l.cfg.AdsbxConfig.ApiUrl != "" {
			adsbxEndpoint = l.cfg.AdsbxConfig.ApiUrl
		}
		if l.cfg.AdsbxConfig.ApiKey != "" {
			adsbxApiKey = l.cfg.AdsbxConfig.ApiKey
		}
		p := tracker.NewAdsbxProducer(l.msgs, adsbxEndpoint, adsbxApiKey)
		v, ok := os.LookupEnv("AIRTRACK_ADSBX_PANIC_IF_STUCK")
		p.PanicIfStuck(!ok || (v == "true" || v == "1" || v == "y" || v == "Y"))
		l.producers = append(l.producers, p)
	}
	if len(l.cfg.Beast) > 0 {
		l.usingBeast = true
		for i, bcfg := range l.cfg.Beast {
			if bcfg.Name == "" {
				return errors.Errorf("beast server %d is missing name field", i)
			} else if bcfg.Host == "" {
				return errors.Errorf("beast server '%s' is missing host field", bcfg.Name)
			}
			l.producers = append(l.producers, tracker.NewBeastProducer(l.msgs, bcfg.Host, bcfg.Port, bcfg.Name))
		}
	}

	opt.AircraftDb = aircraft_db.New()
	err = aircraft_db.LoadAssets(opt.AircraftDb, aircraft_db.Asset)
	if err != nil {
		return errors.Wrapf(err, "load aircraft db assets")
	}
	l.t, err = tracker.New(database, opt)
	if err != nil {
		return errors.Wrapf(err, "initializing tracker")
	}

	if l.cfg.MapSettings != nil && l.cfg.MapSettings.Disabled == false {
		historyFiles := tracker.DefaultHistoryFileCount
		if l.cfg.MapSettings.HistoryCount != 0 {
			historyFiles = l.cfg.MapSettings.HistoryCount
		}
		l.mapServer, err = tracker.NewAircraftMap(l.cfg.MapSettings)
		if err != nil {
			return errors.Wrapf(err, "initializing map")
		}
		mapsToUse := tracker.DefaultMapServices
		if l.cfg.MapSettings.Services != nil {
			mapsToUse = l.cfg.MapSettings.Services
		}
		for _, mapService := range mapsToUse {
			switch mapService {
			case tracker.Dump1090MapService:
				err = l.mapServer.RegisterMapService(dump1090.NewDump1090Map(l.mapServer))
			case tracker.Tar1090MapService:
				err = l.mapServer.RegisterMapService(tar1090.NewTar1090Map(l.mapServer, historyFiles))
			default:
				return errors.New("unsupported map service: " + mapService)
			}
			if err != nil {
				return errors.Wrapf(err, "registering map service (%s)", mapService)
			}
		}
		err = l.t.RegisterProjectStatusListener(tracker.NewMapProjectStatusListener(l.mapServer))
		if err != nil {
			return errors.Wrapf(err, "registering map ProjectStatusListener")
		}
		err = l.t.RegisterProjectAircraftUpdateListener(tracker.NewMapProjectAircraftUpdateListener(l.mapServer))
		if err != nil {
			return errors.Wrapf(err, "registering map ProjectAircraftUpdateListener")
		}
	}

	var ignored int32
	for _, proj := range l.cfg.Projects {
		if proj.Disabled {
			ignored++
			continue
		}
		p, err := tracker.InitProject(proj)
		if err != nil {
			return errors.Wrap(err, "failed to init project")
		}
		err = l.t.AddProject(p)
		if err != nil {
			return errors.Wrap(err, "failed to add project to tracker")
		}
		log.Debugf("init project %s", p.Name)
	}
	if ignored > 0 {
		log.Debugf("skipping %d disabled projects", ignored)
	}
	return nil
}

// Start launches all the configured services
func (l *Loader) Start() error {
	if l.usingBeast {
		l.initBeast()
	}
	if l.mapServer != nil {
		l.mapServer.Serve()
	}
	l.t.Start(l.msgs)
	if l.mailSender != nil {
		l.mailSender.Start()
	}
	if l.cfg.Metrics != nil && l.cfg.Metrics.Enabled {
		go func() {
			err := l.metricsServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		}()
	}

	if l.cpuProfileFile != nil {
		err := pprof.StartCPUProfile(l.cpuProfileFile)
		if err != nil {
			return err
		}
	}
	if l.heapProfileFileName != "" {
		heapProfileCtx, heapProfileCanceller := context.WithCancel(context.Background())
		go l.periodicallyWriteHeapProfile(heapProfileCtx)
		l.heapProfileCanceller = heapProfileCanceller
	}

	for _, producer := range l.producers {
		log.Infof("starting %s producer..", producer.Name())
		producer.Start()
	}
	return nil
}

// initBeast triggers the readsb init and background routines
func (l *Loader) initBeast() {
	// readsb library housekeeping
	readsb.IcaoFilterInitOnce()
	readsb.ModeACInitOnce()
	readsb.ModesChecksumInitOnce(1)

	ctx, icaoExpirationCanceller := context.WithCancel(context.Background())
	l.icaoFilterExpirationCanceller = icaoExpirationCanceller
	l.readsbInitialized = true

	go l.readsbIcaoFilterExpiration(ctx)
}

// readsbIcaoFilterExpiration is a goroutine which perioidcally expires aircraft
// from the readsb icao filter
func (l *Loader) readsbIcaoFilterExpiration(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// todo: need some way to empty the icao filter for proper reloading..
			readsb.IcaoFilterExpire()
			return
		case <-time.After(time.Minute):
			readsb.IcaoFilterExpire()
		}
	}
}

// periodicallyWriteHeapProfile is a goroutine which writes a heap profile
// file every 10 seconds to l.heapProfileFileName suffixed by the heap profile counter
func (l *Loader) periodicallyWriteHeapProfile(ctx context.Context) {
	i := 0
	running := true
	for running {
		select {
		case <-ctx.Done():
			running = false
		case <-time.After(10 * time.Second):
		}
		f, err := os.Create(fmt.Sprintf("%s-%d", l.heapProfileFileName, i))
		if err != nil {
			panic(err)
		}
		err = pprof.WriteHeapProfile(f)
		if err != nil {
			panic(err)
		}
		i++
	}
}

// Stop ends aircraft tracking and brings down running services
func (l *Loader) Stop() error {
	for _, producer := range l.producers {
		log.Debugf("stopping %s producer..", producer.Name())
		producer.Stop()
	}
	close(l.msgs)
	log.Debugf("stopping tracker")
	err := l.t.Stop()
	if err != nil {
		return err
	}

	if l.mapServer != nil {
		log.Debugf("stopping map server")
		err = l.mapServer.Stop()
		if err != nil {
			return err
		}
	}
	if l.metricsServer != nil {
		log.Debugf("stopping metrics server")
		ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			cancel()
		}()
		err := l.metricsServer.Shutdown(ctxShutDown)
		if err != nil && err != http.ErrServerClosed {
			return errors.Wrapf(err, "in metrics server shutdown")
		}
	}
	if l.mailSender != nil {
		log.Debugf("stopping mailer")
		l.mailSender.Stop()
	}
	if l.usingBeast {
		log.Debugf("stopping readsb icao filter expiration routine")
		l.icaoFilterExpirationCanceller()
	}
	if l.cpuProfileFile != nil {
		log.Debugf("stopping CPU profiler")
		pprof.StopCPUProfile()
	}
	if l.heapProfileFileName != "" {
		log.Debugf("stopping heap profile routine")
		l.heapProfileCanceller()
	}
	log.Info("shutdown complete")
	return nil
}
