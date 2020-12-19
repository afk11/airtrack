package tracker

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/afk11/airtrack/pkg/aircraft/ccode"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/email"
	"github.com/afk11/airtrack/pkg/geo"
	"github.com/afk11/airtrack/pkg/iso3166"
	"github.com/afk11/airtrack/pkg/kml"
	"github.com/afk11/airtrack/pkg/mailer"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/afk11/airtrack/pkg/readsb"
	"github.com/afk11/airtrack/pkg/readsb/aircraftdb"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
	"unicode"
)

const (
	locationFetchBatchSize int64 = 500
	// DefaultNearestAirportMaxAltitude - default max altitude (in ft)
	// for nearest airport
	DefaultNearestAirportMaxAltitude int64 = 1400
	// DefaultNearestAirportMaxDistance - default distance in meters
	// for nearest airport max distance
	DefaultNearestAirportMaxDistance float64 = 3000
	// DefaultOnGroundUpdateThreshold - default value for config option.
	// how many consecutive messages to receive before accepting a new on_ground status
	DefaultOnGroundUpdateThreshold int64 = 6
	// DefaultGeoHashLength - length of geohashes to use when bucketing
	// airlines by region
	DefaultGeoHashLength uint = 4

	// Dump1090MapService - name of the dump1090 map service
	Dump1090MapService = "dump1090"
	// Tar1090MapService - name of the tar1090 map service
	Tar1090MapService = "tar1090"
)

var (
	// DefaultMapServices contains the default list of map services to enable
	DefaultMapServices = []string{Dump1090MapService, Tar1090MapService}
)

type (
	// Options wraps options + interfaces for the Tracker type.
	Options struct {
		Filter                    string
		Workers                   int
		SightingTimeout           time.Duration
		NearestAirportMaxDistance float64
		NearestAirportMaxAltitude int64

		OnGroundUpdateThreshold int64

		// LocationUpdateInterval is the system-wide default location update interval.
		// If a project has no LocationUpdateInterval configured, this value will be used.
		// By default, this is zero, so all location updates are accepted. It can be
		// configured with the Sightings.LocationUpdateInterval configuration option.
		LocationUpdateInterval time.Duration

		AirportGeocoder *geo.NearestAirportGeocoder
		Mailer          mailer.MailSender

		CountryCodes *iso3166.Store
		Allocations  ccode.CountryAllocationSearcher
		AircraftDb   *aircraftdb.Db
	}
	// GeocodeLocation contains the result of a geocode search.
	// If ok is false, the search was unsuccessful and the other fields are empty.
	// If ok is true, the lat,long & address fields will be set.
	GeocodeLocation struct {
		ok      bool
		lat     float64
		long    float64
		address string
	}
	// callsignLog records a new callsign for a sighting and the time it was observed.
	callsignLog struct {
		callsign string
		time     time.Time
		// sighting is only set in the database processing routine, as it's
		// not necessarily available if the sighting is new
		sighting *db.Sighting
	}
	// squawkLog records a new squawk for a sighting and the time it was observed.
	squawkLog struct {
		squawk string
		time   time.Time
		// sighting is only set in the database processing routine, as it's
		// not necessarily available if the sighting is new
		sighting *db.Sighting
	}
	// locationLog records a new alt/lat/lon position for a sighting and the time it was observed.
	locationLog struct {
		alt  int64
		lat  float64
		lon  float64
		time time.Time
		// sighting is only set in the database processing routine, as it's
		// not necessarily available if the sighting is new
		sighting *db.Sighting
	}
	// ProjectObservation contains information about a sighting from the point of
	// view of a particular project.
	ProjectObservation struct {
		project *Project

		mem          *Sighting
		sighting     *db.Sighting
		firstSeen    time.Time
		lastSeen     time.Time
		lastLocation time.Time
		haveCallsign bool
		callsign     string
		haveSquawk   bool
		squawk       string

		origin      *GeocodeLocation
		destination *GeocodeLocation

		dirty        bool
		csLogs       []callsignLog
		squawkLogs   []squawkLog
		locationLogs []locationLog
		haveAltBaro  bool
		altitudeBaro int64

		haveAltGeom  bool
		altitudeGeom int64

		haveGS bool
		gs     float64

		haveTrack bool
		track     float64

		tags SightingTags

		haveLocation  bool
		latitude      float64
		longitude     float64
		locationCount int64
		mu            sync.RWMutex
	}
	// FlightTime contains calculated information about the flight time
	FlightTime struct {
		StartTime        time.Time
		StartTimeFmt     string
		EndTime          time.Time
		EndTimeFmt       string
		SightingDuration time.Duration
	}
	// Sighting represents an aircraft we are receiving messages about
	Sighting struct {
		// State represents everything we know about the aircraft. It
		// is one of the structs filters operate on.
		State pb.State
		// Tags contains some meta information about the sighting.
		Tags             SightingTags
		firstSeen        time.Time
		lastSeen         time.Time
		searchedCountry  bool
		searchedOperator bool
		searchedInfo     bool

		a                 *db.Aircraft
		observedBy        map[uint64]*ProjectObservation
		onGroundCandidate bool
		onGroundCounter   int64

		mu sync.RWMutex
	}
	// Tracker - this type is responsible for processing received messages
	// and tracking aircraft
	Tracker struct {
		database                 db.Database
		opt                      Options
		projects                 []*Project
		projectMu                sync.RWMutex
		sightingMu               sync.Mutex
		sighting                 map[string]*Sighting
		projectStatusListeners   []ProjectStatusListener
		projectAcUpdateListeners []ProjectAircraftUpdateListener
		consumerCanceller        context.CancelFunc
		lostAcCanceller          context.CancelFunc
		dbFlushCanceller         context.CancelFunc
		consumerWG               sync.WaitGroup
		mailTemplates            *email.MailTemplates
	}
	// lostSighting contains information needed to process an aircraft
	// that has gone out of view
	lostSighting struct {
		s       *Sighting
		project *Project
		session *db.Session
	}
	// SightingTags contains some meta information about the flight.
	SightingTags struct {
		// IsInTakeoff - this is set to true if we observe the aircraft
		// transitioning from on_ground=true to on_ground=false.
		IsInTakeoff bool
	}
)

// NewProjectObservation initializes a ProjectObservation structure for this sighting & project pair
func NewProjectObservation(p *Project, s *Sighting, msgTime time.Time) *ProjectObservation {
	return &ProjectObservation{
		mem:       s,
		project:   p,
		firstSeen: msgTime,
		lastSeen:  msgTime,
	}
}

// GetFlightTime creates a FlightTime structure containing calculated
// time information for the flight
func (o *ProjectObservation) GetFlightTime() FlightTime {
	return FlightTime{
		StartTime:        o.firstSeen,
		StartTimeFmt:     o.firstSeen.Format(time.RFC822),
		EndTime:          o.lastSeen,
		EndTimeFmt:       o.lastSeen.Format(time.RFC822),
		SightingDuration: o.lastSeen.Sub(o.firstSeen),
	}
}

// HaveCallSign - returns true if the ProjectObservation has a current callsign set
func (o *ProjectObservation) HaveCallSign() bool {
	return o.haveCallsign
}

// SetCallSign updates the current callsign for the sighting, and if track
// is true, creates a callsign log to be written to the database.
func (o *ProjectObservation) SetCallSign(callsign string, track bool, msgTime time.Time) error {
	if track {
		if o.HaveCallSign() {
			log.Infof("[session %d] %s: updated callsign %s -> %s", o.project.Session.ID, o.mem.State.Icao, o.CallSign(), callsign)
		} else {
			log.Infof("[session %d] %s: found callsign %s", o.project.Session.ID, o.mem.State.Icao, callsign)
		}
		o.dirty = true
		o.csLogs = append(o.csLogs, callsignLog{callsign, msgTime, nil})
	}
	o.callsign = callsign
	if !o.haveCallsign {
		o.haveCallsign = true
	}
	return nil
}

// CallSign returns the current callsign, or an empty string if unknown
func (o *ProjectObservation) CallSign() string {
	return o.callsign
}

// HaveSquawk - returns true if the ProjectObservation has a current squawk set
func (o *ProjectObservation) HaveSquawk() bool {
	return o.haveSquawk
}

// SetSquawk updates the current squawk for the sighting, and if track
// is true, creates a squawk log to be written to the database.
func (o *ProjectObservation) SetSquawk(squawk string, track bool, msgTime time.Time) error {
	if track {
		if o.HaveSquawk() {
			log.Infof("[session %d] %s: updated squawk %s -> %s", o.project.Session.ID, o.mem.State.Icao, o.Squawk(), squawk)
		} else {
			log.Infof("[session %d] %s: found squawk %s", o.project.Session.ID, o.mem.State.Icao, squawk)
		}
		o.dirty = true
		o.squawkLogs = append(o.squawkLogs, squawkLog{squawk, msgTime, nil})
	}
	o.squawk = squawk
	if !o.haveSquawk {
		o.haveSquawk = true
	}
	return nil
}

// Squawk returns the current squawk, or an empty string if unknown
func (o *ProjectObservation) Squawk() string {
	return o.squawk
}

// HaveAltitudeBarometric returns true if the current barometric altitude is known
func (o *ProjectObservation) HaveAltitudeBarometric() bool {
	return o.haveAltBaro
}

// AltitudeBarometric returns the current barometric altitude
func (o *ProjectObservation) AltitudeBarometric() int64 {
	return o.altitudeBaro
}

// SetAltitudeBarometric updates the current barmetric altitude
func (o *ProjectObservation) SetAltitudeBarometric(alt int64) error {
	o.altitudeBaro = alt
	if !o.haveAltBaro {
		o.haveAltBaro = true
	}
	return nil
}

// HaveAltitudeGeometric returns true if the current barometric altitude is known
func (o *ProjectObservation) HaveAltitudeGeometric() bool {
	return o.haveAltGeom
}

// AltitudeGeometric returns the current barometric altitude
func (o *ProjectObservation) AltitudeGeometric() int64 {
	return o.altitudeGeom
}

// SetAltitudeGeometric updates the current barmetric altitude
func (o *ProjectObservation) SetAltitudeGeometric(alt int64) error {
	o.altitudeGeom = alt
	if !o.haveAltGeom {
		o.haveAltGeom = true
	}
	return nil
}

// HaveLocation returns true if the current location is known
func (o *ProjectObservation) HaveLocation() bool {
	return o.haveLocation
}

// Location returns the current position
func (o *ProjectObservation) Location() (float64, float64) {
	return o.latitude, o.longitude
}

// SetLocation updates the current location for the sighting, and if track
// is true, creates a location log to be written to the database.
func (o *ProjectObservation) SetLocation(lat, lon float64, track bool, msgTime time.Time) error {
	// if locationUpdateInterval is set, only proceed if msgTime - last location time >= locationUpdateInterval
	if o.project.LocationUpdateInterval != 0 && msgTime.Sub(o.lastLocation) < o.project.LocationUpdateInterval {
		return nil
	}

	if track && o.haveAltBaro {
		o.dirty = true
		o.locationLogs = append(o.locationLogs, locationLog{
			o.AltitudeBarometric(), lat, lon, msgTime, nil,
		})
		log.Infof("[session %d] %s: new position: altitude %dft, position (%f, %f) #pos=%d",
			o.project.Session.ID, o.mem.State.Icao, o.AltitudeBarometric(), lat, lon, o.locationCount)
		o.locationCount++
	}
	o.lastLocation = msgTime
	o.latitude = lat
	o.longitude = lon
	if !o.haveLocation {
		o.haveLocation = true
	}
	return nil
}

// NewSighting initializes a new sighting for aircraft with this ICAO
func NewSighting(icao string, now time.Time) *Sighting {
	return &Sighting{
		State: pb.State{
			Icao: icao,
		},
		firstSeen:  now,
		observedBy: make(map[uint64]*ProjectObservation, 0),
	}
}

// New initializes a new Tracker, or an error if one occurs
func New(database db.Database, opt Options) (*Tracker, error) {
	if opt.SightingTimeout < time.Second*30 {
		return nil, errors.New("invalid sighting timeout - must be at least 30 seconds")
	} else if opt.OnGroundUpdateThreshold < 1 {
		return nil, errors.New("invalid onground confirmation threshold - must be at least 1")
	}

	tpls, err := email.LoadMailTemplates(email.GetTemplates()...)
	if err != nil {
		return nil, errors.Wrapf(err, "loading email templates")
	}

	return &Tracker{
		sighting:                 make(map[string]*Sighting),
		database:                 database,
		opt:                      opt,
		projectStatusListeners:   make([]ProjectStatusListener, 0),
		projectAcUpdateListeners: make([]ProjectAircraftUpdateListener, 0),
		mailTemplates:            tpls,
	}, nil
}

// RegisterProjectStatusListener - accepts a new ProjectStatusListener to
// use for project status updates
func (t *Tracker) RegisterProjectStatusListener(l ProjectStatusListener) error {
	t.projectMu.Lock()
	defer t.projectMu.Unlock()
	t.projectStatusListeners = append(t.projectStatusListeners, l)
	return nil
}

// RegisterProjectAircraftUpdateListener - accepts a new ProjectAircraftUpdateListener to
// use for aircraft updates
func (t *Tracker) RegisterProjectAircraftUpdateListener(l ProjectAircraftUpdateListener) error {
	t.projectMu.Lock()
	defer t.projectMu.Unlock()
	t.projectAcUpdateListeners = append(t.projectAcUpdateListeners, l)
	return nil
}

// Start takes the messages channel and launches consumer goroutines. It
// also starts the lost aircraft + database update goroutines.
func (t *Tracker) Start(msgs chan *pb.Message) {
	consumerCtx, consumerCanceller := context.WithCancel(context.Background())
	t.consumerCanceller = consumerCanceller
	t.consumerWG.Add(t.opt.Workers)
	log.Infof("starting %d message handlers", t.opt.Workers)
	for i := 0; i < t.opt.Workers; i++ {
		go t.startConsumer(consumerCtx, msgs)
	}
	lostAcCtx, lostAcCanceller := context.WithCancel(context.Background())
	t.lostAcCanceller = lostAcCanceller
	go t.checkForLostAircraft(lostAcCtx)

	dbFlushCtx, dbFlushCanceller := context.WithCancel(context.Background())
	t.dbFlushCanceller = dbFlushCanceller
	go t.startDatabaseTask(dbFlushCtx)
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// Stop begins the shutdown routine by signalling stop to goroutines,
// flushes state to disk, and closes open sightings cleanly.
func (t *Tracker) Stop() error {
	log.Info("shutting down tracker")
	log.Debug("await consumers to finish")
	// Wait for consumers to finish their work
	t.consumerWG.Wait()
	log.Debug("cancel lost aircraft handler")
	t.lostAcCanceller()
	log.Debug("cancel background database updates")
	t.dbFlushCanceller()
	log.Debug("flush database updates")
	err := t.processDatabaseUpdates()
	if err != nil {
		return errors.Wrapf(err, "flushing database updates")
	}

	// Take lock ourselves to cleanup pSightings and delete map
	t.sightingMu.Lock()
	defer t.sightingMu.Unlock()

	log.Infof("closing with %d aircraft in view", len(t.sighting))

	// Close pSightings
	pSightings := make([]*db.Sighting, 0, len(t.sighting))
	var pAircraft int64
	for _, sighting := range t.sighting {
		if len(sighting.observedBy) > 0 {
			pAircraft++
		}
		for _, observation := range sighting.observedBy {
			err := t.handleLostAircraft(observation.project, sighting)
			if err != nil {
				log.Warnf("error encountered processing lost aircraft: %s", err.Error())
			}
			if observation.sighting != nil {
				pSightings = append(pSightings, observation.sighting)
			}
			observation.project.obsMu.Lock()
			delete(observation.project.Observations, sighting.State.Icao)
			observation.project.obsMu.Unlock()
		}
		sighting.observedBy = nil
	}

	numListeners := len(t.projectStatusListeners)
	for i := 0; i < numListeners; i++ {
		for pi := range t.projects {
			t.projectStatusListeners[i].Deactivated(t.projects[pi])
		}
	}

	log.Infof("closed with %d aircraft being monitored", pAircraft)

	t.sighting = make(map[string]*Sighting)
	now := time.Now()
	// Split this into batches, full list can cause too many variables sqlite error
	limit := 100
	for i := 0; i < len(pSightings); i += limit {
		err := t.database.CloseSightingBatch(pSightings[i:min(i+limit, len(pSightings))], now)
		if err != nil {
			return errors.Wrapf(err, "closing batch of sightings")
		}
	}

	// Close session
	t.projectMu.RLock()
	defer t.projectMu.RUnlock()
	for _, p := range t.projects {
		res, err := t.database.CloseSession(p.Session, now)
		if err != nil {
			return errors.Wrapf(err, "closing session")
		} else if err = db.CheckRowsUpdated(res, 1); err != nil {
			return errors.Wrap(err, "should have updated 1 session")
		}
	}

	return nil
}

// AddProject accepts a new project and adds it to the tracker.
// Sends Activated event to ProjectStatusListeners
func (t *Tracker) AddProject(p *Project) error {
	t.projectMu.Lock()
	defer t.projectMu.Unlock()

	for _, other := range t.projects {
		if other.Name == p.Name {
			return errors.Errorf("duplicated project name %s", p.Name)
		}
	}

	if p.IsFeatureEnabled(GeocodeEndpoints) && t.opt.AirportGeocoder == nil {
		return errors.Errorf("geocoder must be available for %s feature to work", GeocodeEndpoints)
	}

	project, err := t.database.GetProject(p.Name)
	if err == sql.ErrNoRows {
		_, err := t.database.CreateProject(p.Name, time.Now())
		if err != nil {
			return errors.Wrap(err, "create new project")
		}
		project, err = t.database.GetProject(p.Name)
		if err != nil {
			return errors.Wrap(err, "load new project")
		}
	} else if err != nil {
		return errors.Wrap(err, "query project")
	}

	sessID, err := uuid.NewUUID()
	if err != nil {
		return errors.Wrap(err, "failed to generate session id")
	}
	_, err = t.database.CreateSession(project, sessID.String(),
		p.IsFeatureEnabled(TrackSquawks), p.IsFeatureEnabled(TrackTxTypes), p.IsFeatureEnabled(TrackCallSigns))
	if err != nil {
		return errors.Wrap(err, "create session record")
	}
	session, err := t.database.GetSessionByIdentifier(project, sessID.String())
	if err != nil {
		return errors.Wrap(err, "load session record")
	}
	p.Project = project
	p.Session = session
	// If project hasn't configured a custom LocationUpdateInterval,
	// ensure we use the system wide one.
	if !p.HasLocationUpdateInterval {
		p.LocationUpdateInterval = t.opt.LocationUpdateInterval
	}
	t.projects = append(t.projects, p)

	numListeners := len(t.projectStatusListeners)
	for i := 0; i < numListeners; i++ {
		t.projectStatusListeners[i].Activated(p)
	}
	return nil
}

// startDatabaseTask is a goroutine that periodically writes state
// to disk, and stops if the stop signal is received from ctx.
func (t *Tracker) startDatabaseTask(ctx context.Context) {
	waitTime := time.Second * 5
	for {
		select {
		case <-time.After(waitTime):
			err := t.processDatabaseUpdates()
			if err != nil {
				panic(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// processDatabaseUpdates is called periodically to persist
// recently received data to disk
func (t *Tracker) processDatabaseUpdates() error {
	t.projectMu.RLock()
	defer t.projectMu.RUnlock()
	begin := time.Now()
	updatedSightings := 0
	var csUpdates []callsignLog
	var squawkUpdates []squawkLog
	var locationUpdates []locationLog
	for _, proj := range t.projects {
		proj.obsMu.RLock()
		for _, o := range proj.Observations {
			createdSighting, csLogs, squawkLogs, locationLogs, err := t.updateSightingAndReturnLogs(o)
			if err != nil {
				proj.obsMu.RUnlock()
				return err
			}
			if createdSighting {
				updatedSightings++
			}
			csUpdates = append(csUpdates, csLogs...)
			squawkUpdates = append(squawkUpdates, squawkLogs...)
			locationUpdates = append(locationUpdates, locationLogs...)
		}
		proj.obsMu.RUnlock()
	}

	err := t.writeUpdates(csUpdates, squawkUpdates, locationUpdates)
	if err != nil {
		return errors.Wrapf(err, "write updates")
	}
	timeTaken := time.Since(begin)
	numCsUpdates := len(csUpdates)
	numSquawkUpdates := len(squawkUpdates)
	numLocationUpdates := len(locationUpdates)
	log.Debugf("flushing updates (took %s)  sightings:%d  callsigns:%d  squawks:%d  locations:%d",
		timeTaken, updatedSightings, numCsUpdates, numSquawkUpdates, numLocationUpdates)

	return nil
}
func (t *Tracker) updateSightingAndReturnLogs(o *ProjectObservation) (bool, []callsignLog, []squawkLog, []locationLog, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.sighting != nil && !o.dirty {
		// not interesting, move on
		return false, nil, nil, nil, nil
	}

	hasNoSighting := o.sighting == nil
	hasCsLogs := len(o.csLogs) > 0
	hasSquawkLogs := len(o.squawkLogs) > 0
	hasLocations := len(o.locationLogs) > 0
	var csUpdates []callsignLog
	var squawkUpdates []squawkLog
	var locationUpdates []locationLog
	if hasNoSighting || hasCsLogs || hasSquawkLogs {
		// Updates regarding the sighting record (also to gather build up inserts for batching)
		err := t.database.Transaction(func(tx db.Queries) error {
			var err error
			if hasNoSighting {
				o.sighting, _, err = initProjectSighting(tx, o.project, o.mem.a, o.firstSeen)
				if err != nil {
					// Cleanup reserved sighting
					return errors.Wrapf(err, "failed to init project sighting")
				}
			}
			if hasCsLogs {
				res, err := tx.UpdateSightingCallsign(o.sighting, o.csLogs[len(o.csLogs)-1].callsign)
				if err != nil {
					return errors.Wrap(err, "updating sighting callsign")
				} else if err = db.CheckRowsUpdated(res, 1); err != nil {
					return errors.Wrapf(err, "expected 1 updated row")
				}

				for i := 0; i < len(o.csLogs); i++ {
					(&o.csLogs[i]).sighting = o.sighting
				}
				csUpdates = o.csLogs
				o.csLogs = nil
			}
			if hasSquawkLogs {
				_, err := tx.UpdateSightingSquawk(o.sighting, o.squawkLogs[len(o.squawkLogs)-1].squawk)
				// todo: add this back in once we have 'sighting restore' added.
				// reopened sightings trigger update though row count will be zero
				if err != nil {
					return errors.Wrap(err, "updating sighting squawk")
				}

				for i := 0; i < len(o.squawkLogs); i++ {
					(&o.squawkLogs[i]).sighting = o.sighting
				}
				squawkUpdates = o.squawkLogs
				o.squawkLogs = nil
			}

			return nil
		})
		if err != nil {
			return false, nil, nil, nil, err
		}
	}

	// Locations is processed separately - if we only have locations, we avoid
	// the above transaction
	if hasLocations {
		for i := 0; i < len(o.locationLogs); i++ {
			(&o.locationLogs[i]).sighting = o.sighting
		}
		locationUpdates = o.locationLogs
		o.locationLogs = nil
	}

	o.dirty = false

	return hasNoSighting, csUpdates, squawkUpdates, locationUpdates, nil
}
func (t *Tracker) writeUpdates(csUpdates []callsignLog, squawkUpdates []squawkLog, locationUpdates []locationLog) error {
	csBatch := 100
	numCsUpdates := len(csUpdates)
	numSquawkUpdates := len(squawkUpdates)
	numLocationUpdates := len(locationUpdates)

	// Insert new callsigns, squawks, and location logs in batches
	for i := 0; i < numCsUpdates; i += csBatch {
		err := t.database.Transaction(func(tx db.Queries) error {
			var err error
			last := min(numCsUpdates, i+csBatch)
			for j := i; j < last; j++ {
				_, err = tx.CreateNewSightingCallSign(csUpdates[j].sighting, csUpdates[j].callsign, csUpdates[j].time)
				if err != nil {
					return errors.Wrap(err, "creating callsign record")
				}
			}
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "in transaction")
		}
	}
	for i := 0; i < numSquawkUpdates; i += csBatch {
		err := t.database.Transaction(func(tx db.Queries) error {
			var err error
			last := min(numSquawkUpdates, i+csBatch)
			for j := i; j < last; j++ {
				_, err = tx.CreateNewSightingSquawk(squawkUpdates[j].sighting, squawkUpdates[j].squawk, squawkUpdates[j].time)
				if err != nil {
					return errors.Wrap(err, "creating squawk record")
				}
			}
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "in transaction")
		}
	}
	for i := 0; i < numLocationUpdates; i += csBatch {
		err := t.database.Transaction(func(tx db.Queries) error {
			last := min(numLocationUpdates, i+csBatch)
			for j := i; j < last; j++ {
				_, err := tx.CreateSightingLocation(locationUpdates[j].sighting.ID, locationUpdates[j].time,
					locationUpdates[j].alt, locationUpdates[j].lat, locationUpdates[j].lon)
				if err != nil {
					return errors.Wrapf(err, "failed to insert sighting location")
				}
			}
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "in transaction")
		}
	}
	return nil
}

// checkForLostAircraft is a goroutine that periodically calls doLostAircraftCheck
// and stops once the stop signal is received.
func (t *Tracker) checkForLostAircraft(ctx context.Context) {
	waitTime := time.Second * 5
	for {
		select {
		case <-time.After(waitTime):
			err := t.doLostAircraftCheck()
			if err != nil {
				panic(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// doLostAircraftCheck is called by the checkForLostAircraft goroutine.
// It searches for t.sightings which
// - have a s.lastSeen > our timeout
// - have a project whose observation.lastSeen > our timeout
// If the sighting isn't lost, it's unlocked immediately
// If a sighting is lost and:
//   - no projects are interested -> it's deleted from the map in this section.
//   - at least one project is interested -> it remains locked and is included in results
func (t *Tracker) doLostAircraftCheck() error {
	t.sightingMu.Lock()
	defer t.sightingMu.Unlock()

	lostSightings := make([]lostSighting, 0)
	lostDbSightings := make([]*db.Sighting, 0)
	for _, sighting := range t.sighting {
		sighting.mu.Lock()
		lostForAll := time.Since(sighting.lastSeen) > t.opt.SightingTimeout
		if len(sighting.observedBy) > 0 {
			lostForProject := false
			for _, observation := range sighting.observedBy {
				if lostForAll || time.Since(observation.lastSeen) > t.opt.SightingTimeout {
					lostSightings = append(lostSightings, lostSighting{
						project: observation.project,
						s:       sighting,
						session: observation.project.Session,
					})
					if observation.sighting != nil {
						lostDbSightings = append(lostDbSightings, observation.sighting)
					}
					lostForProject = true
				}
			}
			if !lostForProject && !lostForAll {
				sighting.mu.Unlock()
			}
			// Sightings to be deleted are still locked
		} else {
			// Since the aircraft expired and there were no projects interested
			// delete it here, because it won't be cleaned up otherwise
			if lostForAll {
				delete(t.sighting, sighting.State.Icao)
			}
			sighting.mu.Unlock()
		}
	}

	// If there are no relevant sightings to process, exit early
	if len(lostSightings) == 0 {
		return nil
	}

	hadError := true
	defer func() {
		// Loop over lost sightings, and delete Observation from Sighting,
		// deleting Sighting if there are no more interested projects
		// * Doesn't clean up map if there's an error, in case we want to debug
		// * Build up unique list of sightings. Can't call unlock on a lostSighting
		//   because _several_ aircraft may be cleaned up.
		uniqueSightings := make(map[string]*Sighting)
		for _, lost := range lostSightings {
			if !hadError {
				project := lost.s.observedBy[lost.session.ID].project
				project.obsMu.Lock()
				delete(project.Observations, lost.s.State.Icao)
				delete(lost.s.observedBy, lost.session.ID)
				if len(lost.s.observedBy) == 0 {
					delete(t.sighting, lost.s.State.Icao)
				}
				project.obsMu.Unlock()
			}
			uniqueSightings[lost.s.State.Icao] = lost.s
		}

		// Finally, unlock every sighting.
		for _, s := range uniqueSightings {
			s.mu.Unlock()
		}

		aircraftCountVec.WithLabelValues().Set(float64(len(t.sighting)))
	}()

	// Do this in batches, ensure we don't get too many variables error from sqlite
	now := time.Now()
	limit := 100
	for i := 0; i < len(lostDbSightings); i += limit {
		err := t.database.CloseSightingBatch(lostDbSightings[i:min(i+limit, len(lostDbSightings))], now)
		if err != nil {
			return errors.Wrapf(err, "closing batch of sightings")
		}
	}

	for _, lost := range lostSightings {
		err := t.handleLostAircraft(lost.project, lost.s)
		if err != nil {
			return errors.Wrap(err, "processing lost aircraft")
		}
	}
	hadError = false
	return nil
}

// reverseGeocode attempts to determine the nearest airport for the provided
// latitude and longitude. The nearest airport is only accepted if we are
// within 'NearestAirportMaxDistance' in range
func (t *Tracker) reverseGeocode(lat float64, lon float64) (*GeocodeLocation, float64, error) {
	location := &GeocodeLocation{}
	addr, distance := t.opt.AirportGeocoder.ReverseGeocode(lat, lon)
	if addr == "" || distance > t.opt.NearestAirportMaxDistance {
		if addr != "" {
			log.Debugf("nearest airport %s is too away (%f is over limit %f)", addr, distance, t.opt.NearestAirportMaxDistance)
		}
		location.ok = false
	} else {
		location.ok = true
		location.lat = lat
		location.long = lon
		location.address = addr
	}
	return location, distance, nil
}

// needs to be called with t.sightingMu locked
func (t *Tracker) handleLostAircraft(project *Project, sighting *Sighting) error {
	observation, ok := sighting.observedBy[project.Session.ID]
	if !ok {
		panic(errors.New("failed to find project record in sighting"))
	}

	log.Infof("[session %d] %s: lost aircraft (firstSeen: %s, duration: %s)",
		project.Session.ID, sighting.State.Icao, observation.firstSeen.Format(time.RFC822), time.Since(observation.firstSeen))

	if project.IsFeatureEnabled(GeocodeEndpoints) && observation.HaveLocation() {
		if observation.AltitudeBarometric() > t.opt.NearestAirportMaxAltitude {
			// too high for an airport
			log.Debugf("[session %d] %s: too high to determine destination location",
				project.Session.ID, sighting.State.Icao)
			observation.destination = &GeocodeLocation{}
		} else {
			location, distance, err := t.reverseGeocode(observation.latitude, observation.longitude)
			if err != nil {
				return errors.Wrapf(err, "search destination location")
			}
			if location.ok {
				log.Debugf("[session %d] %s: Destination reverse geocode result: (%f, %f): %s %.1f km",
					project.Session.ID, sighting.State.Icao, observation.latitude, observation.longitude, location.address,
					distance/1000)
			} else {
				log.Debugf("[session %d] %s: Reverse geocode search for destination (%f, %f) yielded no results",
					project.Session.ID, sighting.State.Icao, observation.latitude, observation.longitude)
			}
			observation.destination = location
		}
	}

	// this ensures that all database updates are applied before doing map processing.
	// necessary for aircraft that go out of range, with a location in memory, but none
	// in the table yet. when handling session close, we have already processed updates, so
	// this shouldn't have any major cost
	_, csLogs, squawkLogs, locationLogs, err := t.updateSightingAndReturnLogs(observation)
	if err != nil {
		return errors.Wrapf(err, "updateSightingAndReturnLogs")
	}
	err = t.writeUpdates(csLogs, squawkLogs, locationLogs)
	if err != nil {
		return errors.Wrapf(err, "writeUpdates")
	}

	if project.IsFeatureEnabled(TrackKmlLocation) && observation.locationCount > 1 {
		err := t.processLostAircraftMap(sighting, observation)
		if err != nil {
			return errors.Wrapf(err, "processLostAircraftMap")
		}
	}

	numListeners := len(t.projectAcUpdateListeners)
	for i := 0; i < numListeners; i++ {
		t.projectAcUpdateListeners[i].LostAircraft(project, sighting)
	}
	return nil
}

// processLostAircraftMap performs KML processing when the sighting closes
func (t *Tracker) processLostAircraftMap(sighting *Sighting, observation *ProjectObservation) error {
	if observation.sighting == nil {
		return nil
	}

	project := observation.project
	flightTime := observation.GetFlightTime()

	plainTextKml, firstPos, lastPos, err := buildKml(t.database, project, sighting, observation, &flightTime)
	if err != nil {
		return err
	}

	var mapUpdated bool
	sightingKml, err := t.database.GetSightingKml(observation.sighting)
	if err == sql.ErrNoRows {
		log.Debugf("[session %d] creating KML for %s", project.Session.ID, sighting.State.Icao)
		// Can be created
		_, err = t.database.CreateSightingKmlContent(observation.sighting, plainTextKml)
		if err != nil {
			return errors.Wrap(err, "create sighting kml")
		}
	} else if err == nil {
		mapUpdated = true
		log.Debugf("[session %d] updating KML for %s", project.Session.ID, sighting.State.Icao)
		err = sightingKml.UpdateKml(plainTextKml)
		if err != nil {
			return errors.Wrapf(err, "updating kml")
		}
		_, err = t.database.UpdateSightingKml(sightingKml)
		if err != nil {
			return errors.Wrap(err, "update sighting kml")
		}
	}

	if project.IsEmailNotificationEnabled(MapProduced) {
		log.Debugf("[session %d] %s: sending %s notification", project.Session.ID, sighting.State.Icao, MapProduced)
		err = t.sendMapProducedEmail(project, sighting, observation, &flightTime, plainTextKml, mapUpdated, firstPos, lastPos)
		if err != nil {
			return err
		}
	}
	return nil
}
func buildKml(database db.Database, project *Project, sighting *Sighting, observation *ProjectObservation, flightTime *FlightTime) ([]byte, *db.SightingLocation, *db.SightingLocation, error) {
	var ac string
	var source = "Source"
	var destination = "Destination"
	if observation.HaveCallSign() {
		ac = observation.CallSign()
	} else {
		ac = sighting.State.Icao
	}
	if observation.origin != nil && observation.origin.ok {
		source += fmt.Sprintf(": near %s", observation.origin.address)
	}
	if observation.destination != nil && observation.destination.ok {
		destination += fmt.Sprintf(": near %s", observation.destination.address)
	}
	w := kml.NewWriter(kml.WriterOptions{
		RouteName:        fmt.Sprintf("%s flight", ac),
		RouteDescription: fmt.Sprintf("Departure: %s<br />Arrival: %s<br />Flight duration: %s<br />", flightTime.StartTimeFmt, flightTime.EndTimeFmt, flightTime.SightingDuration),

		SourceName:        source,
		SourceDescription: fmt.Sprintf("Departed at %s", flightTime.StartTimeFmt),

		DestinationName:        destination,
		DestinationDescription: fmt.Sprintf("Arrived at %s", flightTime.EndTimeFmt),
	})

	var numPoints int
	var firstPos, lastPos *db.SightingLocation
	err := database.WalkLocationHistoryBatch(observation.sighting, locationFetchBatchSize, func(location []db.SightingLocation) {
		if firstPos == nil {
			firstPos = &location[0]
		}
		lastPos = &location[len(location)-1]
		w.Write(location)
		numPoints += len(location)
	})
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "error walking location history")
	}

	log.Debugf("[session %d] location history for %s had %d points",
		project.Session.ID, sighting.State.Icao, numPoints)

	kmlStr, err := w.Final()
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "generating KML file")
	}
	return []byte(kmlStr), firstPos, lastPos, nil
}

// startConsumer is a goroutine that reads from the messages channel
// and updates our state for each aircraft. Then ProcessMessage is
// called with the sighting and each project.
func (t *Tracker) startConsumer(ctx context.Context, msgs chan *pb.Message) {
	defer t.consumerWG.Done()

	for msg := range msgs {
		inflightMsgVec.WithLabelValues().Inc()
		t.projectMu.RLock()
		now := time.Now()
		s := t.getSighting(msg.Icao, now)
		err := t.UpdateStateFromMessage(s, msg, now)
		if err != nil {
			s.mu.Unlock()
			panic(err)
		}

		for _, proj := range t.projects {
			err = t.ProcessMessage(proj, s, now, msg)
			if err != nil {
				s.mu.Unlock()
				panic(err)
			}
		}
		s.mu.Unlock()
		t.projectMu.RUnlock()

		t.sightingMu.Lock()
		aircraftCountVec.WithLabelValues().Set(float64(len(t.sighting)))
		t.sightingMu.Unlock()

		inflightMsgVec.WithLabelValues().Dec()
		msgsProcessed.Inc()
	}
}

// getSighting returns an existing Sighting if present,
// and creates a new one if missing. It locks sightingMu
// for this operation. The sighting will be returned
// Locked, so must be unlocked by the caller when finished.
func (t *Tracker) getSighting(icao string, msgTime time.Time) *Sighting {
	// init sighting in map
	t.sightingMu.Lock()
	defer t.sightingMu.Unlock()
	s, ok := t.sighting[icao]
	if !ok {
		s = NewSighting(icao, msgTime)
		s.mu.Lock()
		t.sighting[icao] = s
	} else {
		s.mu.Lock()
	}

	return s
}

// AirlineCodeFromCallsign attempts to extract an airline operator code from
// a callsign. The returned boolean indicates whether the result is valid. If
// true, the operator code will be returned
func AirlineCodeFromCallsign(callSign string) (string, bool) {
	csLen := len(callSign)
	if csLen < 3 {
		return "", false
	} else if !(unicode.IsLetter(rune(callSign[0])) && unicode.IsLetter(rune(callSign[1])) && unicode.IsLetter(rune(callSign[2]))) {
		return "", false
	} else if csLen > 3 && !unicode.IsNumber(rune(callSign[3])) {
		return "notnum", false
	}

	callSignPrefix := callSign[0:3]
	return callSignPrefix, true
}

// UpdateStateFromMessage takes msg and applies new or updated data to the sighting.
func (t *Tracker) UpdateStateFromMessage(s *Sighting, msg *pb.Message, now time.Time) error {
	s.lastSeen = now
	s.State.LastSignal = msg.Signal

	// Update Sighting state
	var err error
	if msg.AltitudeGeometric != "" {
		var alt int64
		alt, err = strconv.ParseInt(msg.AltitudeGeometric, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "parse geometric altitude")
		}
		s.State.HaveAltitudeGeometric = true
		s.State.AltitudeGeometric = alt
	}
	if msg.AltitudeBarometric != "" {
		var alt int64
		alt, err = strconv.ParseInt(msg.AltitudeBarometric, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "parse barometric altitudeBaro")
		}
		s.State.HaveAltitudeBarometric = true
		s.State.AltitudeBarometric = alt
	}
	if msg.Latitude != "" && msg.Longitude != "" {
		var lat, long float64
		lat, err = strconv.ParseFloat(msg.Latitude, 64)
		if err != nil {
			return errors.Wrapf(err, "parse msg latitude")
		}
		long, err = strconv.ParseFloat(msg.Longitude, 64)
		if err != nil {
			return errors.Wrapf(err, "parse msg longitude")
		}
		s.State.HaveLocation = true
		s.State.Latitude = lat
		s.State.Longitude = long
	}
	if msg.CallSign != "" && msg.CallSign != s.State.CallSign {
		s.State.HaveCallsign = true
		s.State.CallSign = msg.CallSign

		code, ok := AirlineCodeFromCallsign(msg.CallSign)
		if ok && (s.State.Operator == nil || s.State.OperatorCode != code) {
			s.State.OperatorCode = code
			if info, ok := t.opt.AircraftDb.GetOperator(code); ok {
				s.State.Operator = info
			}
		}
	}
	if msg.Squawk != "" && msg.Squawk != s.State.Squawk {
		s.State.HaveSquawk = true
		s.State.Squawk = msg.Squawk
	}
	if msg.HaveVerticalRateBarometric {
		s.State.HaveVerticalRateBarometric = true
		s.State.VerticalRateBarometric = msg.VerticalRateBarometric
		if msg.VerticalRateBarometric == 0 && s.Tags.IsInTakeoff && (s.State.HaveAltitudeBarometric && s.State.AltitudeBarometric > 200) {
			s.Tags.IsInTakeoff = false
			log.Tracef("ac finished takeoff %s (Alt: %d, VerticalRate: %d)",
				s.State.Icao, s.State.AltitudeBarometric, msg.VerticalRateBarometric)
		}
	}
	if msg.HaveVerticalRateGeometric {
		s.State.HaveVerticalRateGeometric = true
		s.State.VerticalRateGeometric = msg.VerticalRateBarometric
	}
	if msg.HaveFmsAltitude {
		s.State.HaveFmsAltitude = true
		s.State.FmsAltitude = msg.FmsAltitude
	}
	if msg.HaveNavHeading {
		s.State.HaveNavHeading = true
		s.State.NavHeading = msg.NavHeading
	}
	if msg.HaveNavQNH {
		s.State.HaveNavQNH = true
		s.State.NavQNH = msg.NavQNH
	}
	if msg.HaveTrueAirSpeed {
		s.State.HaveTrueAirSpeed = true
		s.State.TrueAirSpeed = msg.TrueAirSpeed
	}
	if msg.HaveIndicatedAirSpeed {
		s.State.HaveIndicatedAirSpeed = true
		s.State.IndicatedAirSpeed = msg.IndicatedAirSpeed
	}
	if msg.HaveMach {
		s.State.HaveMach = true
		s.State.Mach = msg.Mach
	}
	if msg.HaveRoll {
		s.State.HaveRoll = true
		s.State.Roll = msg.Roll
	}
	if msg.Category != "" {
		s.State.HaveCategory = true
		s.State.Category = msg.Category
	}
	if msg.NavModes != 0 {
		nm := readsb.NavModes(msg.NavModes)
		for _, navMode := range readsb.AllNavModes {
			if (nm & navMode) != 0 {
				s.State.NavModes |= uint32(navMode)
			}
		}
	}
	if msg.ADSBVersion != 0 {
		s.State.ADSBVersion = msg.ADSBVersion
	}
	if msg.HaveNACP {
		s.State.HaveNACP = true
		s.State.NACP = msg.NACP
	}
	if msg.HaveNACV {
		s.State.HaveNACV = true
		s.State.NACV = msg.NACV
	}
	if msg.HaveNICBaro {
		s.State.HaveNICBaro = true
		s.State.NICBaro = msg.NICBaro
	}
	if msg.HaveSIL {
		s.State.HaveSIL = true
		s.State.SIL = msg.SIL
		s.State.SILType = msg.SILType
	}
	if msg.Track != "" {
		track, err := strconv.ParseFloat(msg.Track, 64)
		if err != nil {
			return errors.Wrapf(err, "parse msg latitude")
		}
		s.State.Track = track
		s.State.HaveTrack = true
	}
	if msg.GroundSpeed != "" {
		gs, err := strconv.ParseFloat(msg.GroundSpeed, 64)
		if err != nil {
			return errors.Wrapf(err, "parse msg ground speed")
		}
		s.State.GroundSpeed = gs
		s.State.HaveGroundSpeed = true
	}
	if s.State.IsOnGround != msg.IsOnGround {
		if s.onGroundCandidate == msg.IsOnGround {
			s.onGroundCounter++
			if s.onGroundCounter > t.opt.OnGroundUpdateThreshold {
				log.Tracef("%s: updated IsOnGround: %t -> %t", s.State.Icao, s.State.IsOnGround, msg.IsOnGround)
				s.State.IsOnGround = msg.IsOnGround
				if !s.State.IsOnGround && s.State.VerticalRateBarometric > 0 {
					// todo: review best altitude for here
					log.Tracef("%s: IsInTakeoff (Alt: %d, VerticalRate: %d)", s.State.Icao, s.State.AltitudeBarometric, s.State.VerticalRateBarometric)
					s.Tags.IsInTakeoff = true
					// takeoff_begin
				}
			}
		} else {
			log.Tracef("%s: new candidate IsOnGround %t", s.State.Icao, msg.IsOnGround)
			s.onGroundCandidate = msg.IsOnGround
			s.onGroundCounter = 0
		}
	}

	if !s.searchedCountry && t.opt.Allocations != nil {
		s.searchedCountry = true
		code, err := t.opt.Allocations.DetermineCountryCode(s.State.Icao)
		if err != nil {
			return errors.Wrapf(err, "finding icao hex country code")
		} else if code == nil {
			log.Tracef("icao %s country unknown", s.State.Icao)
		} else {
			country, found := t.opt.CountryCodes.GetCountryCode(*code)
			if !found {
				panic("country code not found")
			}
			s.State.HaveCountry = true
			s.State.CountryCode = code.String()
			s.State.Country = country.Name()
			log.Tracef("icao %s country determined: %s %s", s.State.Icao, code.String(), country.Name())
		}
	}

	if !s.searchedInfo {
		ac, ok := t.opt.AircraftDb.GetAircraft(s.State.Icao)
		if ok {
			s.State.Info = ac
		}
		s.searchedInfo = true
	}

	return nil
}

// getObservation searches a s for an for an observation by this project.
// If an observation exists, it is returned. Otherwise, a new observation
// will be created and associated with the s and project. The returned
// ProjectObservation will have the write lock held.
func (t *Tracker) getObservation(project *Project, s *Sighting, msgTime time.Time) (*ProjectObservation, bool) {
	observation, ok := s.observedBy[project.Session.ID]
	var sightingOpened bool
	if !ok {
		observation = NewProjectObservation(project, s, msgTime)
		project.obsMu.Lock()
		s.observedBy[project.Session.ID] = observation
		project.Observations[s.State.Icao] = observation
		project.obsMu.Unlock()
		sightingOpened = true
		log.Infof("[session %d] %s: new sighting", project.Session.ID, s.State.Icao)
	}
	observation.mu.Lock()
	return observation, sightingOpened
}

// ProcessMessage is called to process the updated Sighting in the context
// of the provided project.
func (t *Tracker) ProcessMessage(project *Project, s *Sighting, now time.Time, msg *pb.Message) error {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000 // make microseconds
		msgDurations.Observe(us)
	}))
	defer timer.ObserveDuration()

	// Evaluate filter, see if we wish to continue
	if project.Program != nil {
		passed, err := checkIfPassesFilter(project.Program, msg, &s.State)
		if err != nil {
			return errors.Wrapf(err, "evaluating filter")
		}
		if !passed {
			msgsFiltered.Inc()
			return nil
		}
	}

	// Create if missing, and assign to sighting
	if s.a == nil {
		var err error
		s.a, err = t.loadAircraft(s.State.Icao, now)
		if err != nil {
			return errors.Wrapf(err, "loading aircraft by icao")
		}
	}

	// Find or initialize project observation, if not already done
	observation, sightingOpened := t.getObservation(project, s, now)
	defer observation.mu.Unlock()

	// Update Projects information in DB
	observation.lastSeen = now

	err := t.updateObservation(project, observation, s, sightingOpened, now)
	if err != nil {
		return err
	}

	numListeners := len(t.projectAcUpdateListeners)
	for i := 0; i < numListeners; i++ {
		if sightingOpened {
			t.projectAcUpdateListeners[i].NewAircraft(project, s)
		} else {
			t.projectAcUpdateListeners[i].UpdatedAircraft(project, s)
		}
	}

	return nil
}
func (t *Tracker) updateObservation(project *Project, observation *ProjectObservation, s *Sighting, sightingOpened bool, now time.Time) error {
	var updatedAltBaro, updatedAltGeom, updatedLocation, updatedGS, updatedTrack bool
	if s.State.HaveAltitudeBarometric {
		updatedAltBaro = !observation.HaveAltitudeBarometric() || s.State.AltitudeBarometric != observation.AltitudeBarometric()
		if updatedAltBaro {
			err := observation.SetAltitudeBarometric(s.State.AltitudeBarometric)
			if err != nil {
				return errors.Wrapf(err, "setting barometric altitude")
			}
		}
	}
	if s.State.HaveAltitudeGeometric {
		updatedAltGeom = !observation.HaveAltitudeGeometric() || s.State.AltitudeGeometric != observation.AltitudeGeometric()
		if updatedAltGeom {
			err := observation.SetAltitudeGeometric(s.State.AltitudeGeometric)
			if err != nil {
				return errors.Wrapf(err, "setting geometric altitude")
			}
		}
	}
	if s.State.HaveTrack {
		updatedTrack = !observation.haveTrack || s.State.Track != observation.track
		if updatedTrack {
			observation.track = s.State.Track
			observation.haveTrack = true
		}
	}
	if s.State.HaveGroundSpeed {
		updatedGS = !observation.haveGS || s.State.GroundSpeed != observation.gs
		if updatedGS {
			observation.gs = s.State.GroundSpeed
			observation.haveGS = true
		}
	}
	if s.State.HaveLocation {
		oldlat, oldlon := observation.Location()
		updatedLocation = !observation.HaveLocation() || s.State.Latitude != oldlat || s.State.Longitude != oldlon
		if updatedLocation {
			err := observation.SetLocation(s.State.Latitude, s.State.Longitude, project.IsFeatureEnabled(TrackKmlLocation), now)
			if err != nil {
				return errors.Wrapf(err, "setting location")
			}
		}
	}
	if s.State.HaveCallsign {
		updatedCallSign := !observation.HaveCallSign() || s.State.CallSign != observation.CallSign()
		if updatedCallSign {
			err := observation.SetCallSign(s.State.CallSign, project.IsFeatureEnabled(TrackCallSigns), now)
			if err != nil {
				return errors.Wrapf(err, "setting callsign")
			}
		}
	}
	if s.State.HaveSquawk {
		updatedSquawk := !observation.HaveSquawk() || (s.State.Squawk != observation.Squawk())
		if updatedSquawk {
			err := observation.SetSquawk(s.State.Squawk, project.IsFeatureEnabled(TrackSquawks), now)
			if err != nil {
				return errors.Wrapf(err, "setting squawk")
			}
		}
	}

	if s.Tags.IsInTakeoff != observation.tags.IsInTakeoff {
		observation.tags.IsInTakeoff = s.Tags.IsInTakeoff
		if project.IsFeatureEnabled(TrackTakeoff) {
			geocodeOK := observation.origin != nil && observation.origin.ok
			if observation.tags.IsInTakeoff {
				log.Infof("[session %d] %s: has begun takeoff",
					project.Session.ID, s.State.Icao)
				if geocodeOK && project.IsEmailNotificationEnabled(TakeoffFromAirport) {
					// takeoff start
					log.Debugf("[session %d] %s: sending %s notification", project.Session.ID, s.State.Icao, TakeoffFromAirport)
					err := t.sendTakeoffFromAirportEmail(project, s, observation)
					if err != nil {
						return err
					}
				} else if !geocodeOK && project.IsEmailNotificationEnabled(TakeoffUnknownAirport) {
					// didn't geocode origin airport
					log.Debugf("[session %d] %s: sending %s notification", project.Session.ID, s.State.Icao, TakeoffUnknownAirport)
					err := t.sendTakeoffUnknownAirportEmail(project, s, observation)
					if err != nil {
						return err
					}
				}
			} else {
				log.Infof("[session %d] %s: has finished takeoff",
					project.Session.ID, s.State.Icao)
				if project.IsEmailNotificationEnabled(TakeoffComplete) {
					// didn't geocode origin airport
					log.Debugf("[session %d] %s: sending %s notification", project.Session.ID, s.State.Icao, TakeoffComplete)
					var airport string
					if geocodeOK {
						airport = observation.origin.address
					}
					err := t.sendTakeoffCompleteEmail(project, s, observation, airport)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	if sightingOpened && project.IsEmailNotificationEnabled(SpottedInFlight) {
		log.Debugf("[session %d] %s: sending %s notification", project.Session.ID, s.State.Icao, SpottedInFlight)
		err := t.sendSpottedInFlightEmail(project, s, observation)
		if err != nil {
			return err
		}
	}

	if project.IsFeatureEnabled(GeocodeEndpoints) && observation.origin == nil && observation.HaveLocation() {
		if observation.altitudeBaro > t.opt.NearestAirportMaxAltitude {
			// too high for an airport
			log.Debugf("[session %d] %s: too high to determine origin location (%d over max %d)",
				project.Session.ID, s.State.Icao, observation.altitudeBaro, t.opt.NearestAirportMaxAltitude)
			observation.origin = &GeocodeLocation{}
		} else {
			lat, lon := observation.Location()
			location, distance, err := t.reverseGeocode(lat, lon)
			if err != nil {
				return errors.Wrap(err, "searching origin")
			}
			if location.ok {
				log.Debugf("[session %d] %s: Origin reverse geocode result: (%f, %f): %s %.1f km",
					project.Session.ID, s.State.Icao, lat, lon, location.address,
					distance/1000)
			} else {
				log.Debugf("[session %d] %s: Reverse geocode search for origin (%f, %f) yielded no results",
					project.Session.ID, s.State.Icao, lat, lon)
			}
			observation.origin = location
		}
	}
	return nil
}
func (t *Tracker) sendTakeoffFromAirportEmail(project *Project, s *Sighting, observation *ProjectObservation) error {
	msg, err := email.PrepareTakeoffFromAirport(t.mailTemplates, project.NotifyEmail, email.TakeoffParams{
		Project:      project.Name,
		Icao:         s.State.Icao,
		CallSign:     s.State.CallSign,
		AirportName:  observation.origin.address,
		StartTimeFmt: s.firstSeen.Format(time.RFC1123Z),
		StartLocation: email.Location{
			Latitude:  s.State.Latitude,
			Longitude: s.State.Longitude,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "preparing TakeoffFromAirport email")
	}
	err = t.opt.Mailer.Queue(*msg)
	if err != nil {
		return errors.Wrapf(err, "queueing TakeoffFromAirport email")
	}
	return nil
}
func (t *Tracker) sendTakeoffUnknownAirportEmail(project *Project, s *Sighting, observation *ProjectObservation) error {
	msg, err := email.PrepareTakeoffUnknownAirport(t.mailTemplates, project.NotifyEmail, email.TakeoffUnknownAirportParams{
		Project:      project.Name,
		Icao:         s.State.Icao,
		CallSign:     s.State.CallSign,
		StartTimeFmt: s.firstSeen.Format(time.RFC1123Z),
		StartLocation: email.Location{
			Latitude:  s.State.Latitude,
			Longitude: s.State.Longitude,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "preparing TakeoffUnknownAirport email")
	}
	err = t.opt.Mailer.Queue(*msg)
	if err != nil {
		return errors.Wrapf(err, "queueing TakeoffUnknownAirport email")
	}
	return nil
}
func (t *Tracker) sendTakeoffCompleteEmail(project *Project, s *Sighting, observation *ProjectObservation, airport string) error {
	msg, err := email.PrepareTakeoffComplete(t.mailTemplates, project.NotifyEmail, email.TakeoffCompleteParams{
		Project:      project.Name,
		Icao:         s.State.Icao,
		CallSign:     s.State.CallSign,
		StartTimeFmt: s.firstSeen.Format(time.RFC1123Z),
		AirportName:  airport,
		StartLocation: email.Location{
			Latitude:  s.State.Latitude,
			Longitude: s.State.Longitude,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "preparing TakeoffComplete email")
	}
	err = t.opt.Mailer.Queue(*msg)
	if err != nil {
		return errors.Wrapf(err, "queueing TakeoffComplete email")
	}
	return nil
}
func (t *Tracker) sendSpottedInFlightEmail(project *Project, s *Sighting, observation *ProjectObservation) error {
	msg, err := email.PrepareSpottedInFlightEmail(t.mailTemplates, project.NotifyEmail, email.SpottedInFlightParameters{
		Project:      project.Name,
		Icao:         s.State.Icao,
		CallSign:     s.State.CallSign,
		StartTime:    s.firstSeen,
		StartTimeFmt: s.firstSeen.Format(time.RFC1123Z),
	})
	if err != nil {
		return errors.Wrapf(err, "preparing SpottedInFlight email")
	}
	err = t.opt.Mailer.Queue(*msg)
	if err != nil {
		return errors.Wrapf(err, "queueing SpottedInFlight email")
	}
	return err
}
func (t *Tracker) sendMapProducedEmail(project *Project, s *Sighting, observation *ProjectObservation, ft *FlightTime, plainTextKml []byte, mapUpdated bool, firstPos, lastPos *db.SightingLocation) error {
	sp := email.MapProducedParameters{
		Project:      project.Name,
		Icao:         s.State.Icao,
		StartTimeFmt: ft.StartTimeFmt,
		EndTimeFmt:   ft.EndTimeFmt,
		DurationFmt:  ft.SightingDuration.String(),
		StartLocation: email.Location{
			Latitude:  firstPos.Latitude,
			Longitude: firstPos.Longitude,
			Altitude:  firstPos.Altitude,
		},
		EndLocation: email.Location{
			Latitude:  lastPos.Latitude,
			Longitude: lastPos.Longitude,
			Altitude:  lastPos.Altitude,
		},
		MapUpdated: mapUpdated,
	}
	if observation.HaveCallSign() {
		sp.CallSign = observation.CallSign()
	}
	msg, err := email.PrepareMapProducedEmail(t.mailTemplates, project.NotifyEmail, plainTextKml, sp)
	if err != nil {
		return errors.Wrapf(err, "creating MapProduced email")
	}
	err = t.opt.Mailer.Queue(*msg)
	if err != nil {
		return errors.Wrapf(err, "queueing MapProduced email for delivery")
	}
	return nil
}

// loadAircraft finds or creates an aircraft record for the provided icao.
func (t *Tracker) loadAircraft(icao string, seenTime time.Time) (*db.Aircraft, error) {
	// create sighting
	a, err := t.database.GetAircraftByIcao(icao)
	if err == sql.ErrNoRows {
		_, err := t.database.CreateAircraft(icao, seenTime)
		if err != nil {
			return nil, err
		}
		a, err = t.database.GetAircraftByIcao(icao)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return a, nil
}

// initProjectSighting creates or reopens a sighting for the aircraft in the provided project
func initProjectSighting(tx db.Queries, p *Project, ac *db.Aircraft, firstSeenTime time.Time) (*db.Sighting, bool, error) {
	s, err := tx.GetLastSighting(p.Session, ac)
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}

	if p.ReopenSightings && s != nil {
		if s.ClosedAt == nil {
			return nil, false, errors.Errorf("last session for %s (id=%d) is still open - possibly running multiple instances of this software", ac.Icao, s.ID)
		}
		timeSinceClosed := time.Since(*s.ClosedAt)
		// reactivate session if it's within our interval
		// todo: maybe other checks here, like, finished_on_ground or something to avoid rapid stops, so
		// we break up legs of the journey
		if timeSinceClosed < p.ReopenSightingsInterval {
			res, err := tx.ReopenSighting(s)
			if err != nil {
				return nil, false, errors.Wrap(err, "reopening sighting")
			} else if err = db.CheckRowsUpdated(res, 1); err != nil {
				return nil, false, err
			}

			log.Infof("[session %d] %s: reopened sighting after %s", p.Session.ID, ac.Icao, timeSinceClosed)
			return s, true, nil
		}
	}

	// A new sighting is needed
	_, err = tx.CreateSighting(p.Session, ac, firstSeenTime)
	if err != nil {
		return nil, false, errors.Wrapf(err, "creating sighting record failed")
	}
	s, err = tx.GetLastSighting(p.Session, ac)
	if err != nil {
		return nil, false, errors.Wrapf(err, "load new sighting failed")
	}

	return s, false, nil
}

// checkIfPassesFilter evaluates the CEL program and passes it's inputs.
// The returned boolean result is only valid if no error is returned.
func checkIfPassesFilter(prg cel.Program, msg *pb.Message, state *pb.State) (bool, error) {
	filterTimer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000 // make microseconds
		filterDurations.Observe(us)
	}))
	defer filterTimer.ObserveDuration()

	out, _, err := prg.Eval(map[string]interface{}{
		"msg":                msg,
		"state":              state,
		"AdsbExchangeSource": pb.Source_AdsbExchange,
		"BeastSource":        pb.Source_BeastServer,
	})
	if err != nil {
		return false, err
	} else if out.Type() != types.BoolType {
		return false, errors.New("filter returned non-boolean result")
	}
	ret, ok := out.Value().(bool)
	if !ok {
		//if (out.Type() != types.BoolType) {
		panic(errors.New("not a bool"))
	}

	return ret, nil
}
