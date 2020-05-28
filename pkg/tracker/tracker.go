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
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
)

const (
	LocationFetchBatchSize           int64   = 500
	DefaultNearestAirportMaxAltitude int64   = 1400
	DefaultNearestAirportMaxDistance float64 = 3000
	DefaultOnGroundUpdateThreshold   int64   = 6
	DefaultGeoHashLength             uint    = 4
)

type (
	Options struct {
		Filter                    string
		Workers                   int
		SightingTimeout           time.Duration
		NearestAirportMaxDistance float64
		NearestAirportMaxAltitude int64

		OnGroundUpdateThreshold int64

		AirportGeocoder *geo.NearestAirportGeocoder
		Mailer          mailer.MailSender

		CountryCodes *iso3166.Store
		Allocations  ccode.CountryAllocationSearcher
	}
	GeocodeLocation struct {
		ok      bool
		lat     float64
		long    float64
		address string
	}
	ProjectObservation struct {
		project   *Project
		sighting  *db.Sighting
		firstSeen time.Time
		lastSeen  time.Time

		origin      *GeocodeLocation
		destination *GeocodeLocation

		haveAlt  bool
		altitude int64

		tags SightingTags

		haveLocation  bool
		latitude      float64
		longitude     float64
		locationCount int64
	}
	Sighting struct {
		State           pb.State
		Tags            SightingTags
		firstSeen       time.Time
		lastSeen        time.Time
		searchedCountry bool

		a          *db.Aircraft
		observedBy map[uint64]*ProjectObservation

		onGroundCandidate bool
		onGroundCounter   int64

		mu sync.RWMutex
	}
	Tracker struct {
		dbConn            *sqlx.DB
		opt               Options
		projects          []*Project
		projectMu         sync.RWMutex
		sightingMu        sync.Mutex
		sighting          map[string]*Sighting
		consumerCanceller context.CancelFunc
		lostAcCanceller   context.CancelFunc
		consumerWG        sync.WaitGroup
		mailTemplates     *email.MailTemplates
	}
	lostSighting struct {
		s       *Sighting
		project *Project
		session *db.CollectionSession
	}
)

type SightingTags struct {
	IsInTakeoff bool
}

func NewProjectObservation(p *Project, s *db.Sighting, msgTime time.Time) *ProjectObservation {
	return &ProjectObservation{
		project:   p,
		sighting:  s,
		firstSeen: msgTime,
		lastSeen:  msgTime,
	}
}

func NewSightingWithAircraft(a *db.Aircraft) *Sighting {
	return &Sighting{
		State: pb.State{
			Icao: a.Icao,
		},
		a:          a,
		firstSeen:  time.Now(),
		observedBy: make(map[uint64]*ProjectObservation, 0),
	}
}
func NewSighting(icao string) *Sighting {
	return &Sighting{
		State: pb.State{
			Icao: icao,
		},
		firstSeen:  time.Now(),
		observedBy: make(map[uint64]*ProjectObservation, 0),
	}
}

func New(dbConn *sqlx.DB, opt Options) (*Tracker, error) {
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
		sighting:      make(map[string]*Sighting),
		dbConn:        dbConn,
		opt:           opt,
		mailTemplates: tpls,
	}, nil
}

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
}

func (t *Tracker) Stop() error {
	log.Info("shutting down tracker")
	log.Debug("await consumers to finish")
	// Wait for consumers to finish their work
	t.consumerWG.Wait()
	log.Debug("cancel lost aircraft handler")
	t.lostAcCanceller()

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
			for _, observation := range sighting.observedBy {
				pSightings = append(pSightings, observation.sighting)
				err := t.handleLostAircraft(observation.project, sighting)
				if err != nil {
					log.Warnf("error encountered processing lost aircraft: %s", err.Error())
				}
			}
		}
	}

	log.Infof("closed with %d aircraft being monitored", pAircraft)

	t.sighting = make(map[string]*Sighting)
	err := db.CloseSightingBatch(t.dbConn, pSightings)
	if err != nil {
		return err
	}

	// Close session
	t.projectMu.RLock()
	defer t.projectMu.RUnlock()
	for _, p := range t.projects {
		log.Debugf("close session %d", p.Session.Id)
		res, err := db.CloseSession(t.dbConn, p.Session)
		if err != nil {
			return err
		} else if err = db.CheckRowsUpdated(res, 1); err != nil {
			return errors.Wrap(err, "should have updated 1 session")
		}
	}

	return nil
}

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

	site, err := db.LoadCollectionSite(t.dbConn, p.Name)
	if err == sql.ErrNoRows {
		_, err := db.NewCollectionSite(t.dbConn, p.Name, time.Now())
		if err != nil {
			return errors.Wrap(err, "create new project")
		}
		site, err = db.LoadCollectionSite(t.dbConn, p.Name)
		if err != nil {
			return errors.Wrap(err, "load new project")
		}
	} else if err != nil {
		return errors.Wrap(err, "query collection site")
	}
	sessId, err := uuid.NewUUID()
	if err != nil {
		return errors.Wrap(err, "failed to generate session id")
	}
	_, err = db.NewCollectionSession(t.dbConn, site, sessId.String(),
		p.IsFeatureEnabled(TrackSquawks), p.IsFeatureEnabled(TrackTxTypes), p.IsFeatureEnabled(TrackCallSigns))
	if err != nil {
		return errors.Wrap(err, "create session record")
	}
	session, err := db.LoadSessionByIdentifier(t.dbConn, site, sessId.String())
	if err != nil {
		return errors.Wrap(err, "load session record")
	}
	p.Site = site
	p.Session = session
	t.projects = append(t.projects, p)
	return nil
}

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

func (t *Tracker) doLostAircraftCheck() error {
	t.sightingMu.Lock()
	defer t.sightingMu.Unlock()

	// This section searches for t.sightings which
	// - have a s.lastSeen > our timeout
	// - have a project whose observation.lastSeen > our timeout
	// If the sighting isn't lost, it's unlocked immediately
	// If a sighting is lost and:
	//   - no projects are interested -> it's deleted from the map in this section.
	//   - at least one project is interested -> it remains locked and is included in results
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
					lostDbSightings = append(lostDbSightings, observation.sighting)
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
				delete(lost.s.observedBy, lost.session.Id)
				if len(lost.s.observedBy) == 0 {
					delete(t.sighting, lost.s.a.Icao)
				}
			}
			uniqueSightings[lost.s.State.Icao] = lost.s
		}

		// Finally, unlock every sighting.
		for _, s := range uniqueSightings {
			s.mu.Unlock()
		}

		aircraftCountVec.WithLabelValues().Set(float64(len(t.sighting)))
	}()

	err := db.CloseSightingBatch(t.dbConn, lostDbSightings)
	if err != nil {
		return err
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

func (t *Tracker) reverseGeocode(lat float64, lon float64) (*GeocodeLocation, float64, error) {
	location := &GeocodeLocation{}
	addr, distance, err := t.opt.AirportGeocoder.ReverseGeocode(lat, lon)
	if err != nil {
		return nil, 0.0, errors.Wrap(err, "reverse geocoding location")
	} else if addr == "" || distance > t.opt.NearestAirportMaxDistance {
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
	observation, ok := sighting.observedBy[project.Session.Id]
	if !ok {
		panic(errors.New("failed to find project record in sighting!"))
	}

	log.Infof("[session %d] %s: lost aircraft (firstSeen: %s, duration: %s)",
		project.Session.Id, sighting.a.Icao, observation.firstSeen.Format(time.RFC822), time.Since(observation.firstSeen))

	if project.IsFeatureEnabled(GeocodeEndpoints) && observation.haveLocation {
		if observation.altitude > t.opt.NearestAirportMaxAltitude {
			// too high for an airport
			log.Debugf("[session %d] %s: too high to determine destination location",
				project.Session.Id, sighting.a.Icao)
			observation.destination = &GeocodeLocation{}
		} else {
			location, distance, err := t.reverseGeocode(observation.latitude, observation.longitude)
			if err != nil {
				return errors.Wrapf(err, "search destination location")
			}
			if location.ok {
				log.Debugf("[session %d] %s: Destination reverse geocode result: (%f, %f): %s %.1f km",
					project.Session.Id, sighting.a.Icao, observation.latitude, observation.longitude, location.address,
					distance/1000)
			} else {
				log.Debugf("[session %d] %s: Reverse geocode search for destination (%f, %f) yielded no results",
					project.Session.Id, sighting.a.Icao, observation.latitude, observation.longitude)
			}
			observation.destination = location
		}
	}

	if project.IsFeatureEnabled(TrackKmlLocation) && observation.locationCount > 1 {
		var ac string
		var source = "Source"
		var destination = "Destination"
		startTimeFmt := observation.firstSeen.Format(time.RFC822)
		endTimeFmt := observation.lastSeen.Format(time.RFC822)
		sightingDuration := observation.lastSeen.Sub(observation.firstSeen)
		if observation.sighting.CallSign != nil {
			ac = *observation.sighting.CallSign
		} else {
			ac = sighting.State.Icao
		}
		if observation.origin.ok {
			source += fmt.Sprintf(": near %s", observation.origin.address)
		}
		if observation.destination.ok {
			destination += fmt.Sprintf(": near %s", observation.destination.address)
		}
		w := kml.NewWriter(kml.WriterOptions{
			RouteName:        fmt.Sprintf("%s flight", ac),
			RouteDescription: fmt.Sprintf("Departure: %s<br />Arrival: %s<br />Flight duration: %s<br />", startTimeFmt, endTimeFmt, sightingDuration),

			SourceName:        source,
			SourceDescription: fmt.Sprintf("Departed at %s", startTimeFmt),

			DestinationName:        destination,
			DestinationDescription: fmt.Sprintf("Arrived at %s", endTimeFmt),
		})

		var numPoints int
		var firstPos, lastPos *db.SightingLocation
		err := db.GetLocationHistoryWalkBatch(t.dbConn, observation.sighting, LocationFetchBatchSize, func(location []db.SightingLocation) {
			if firstPos == nil {
				firstPos = &location[0]
			}
			lastPos = &location[len(location)-1]
			w.Write(location)
			numPoints += len(location)
		})
		if err != nil {
			return err
		}
		kmlStr, err := w.Final()
		if err != nil {
			return err
		}
		log.Debugf("[session %d] location history for %s had %d points",
			project.Session.Id, sighting.a.Icao, numPoints)

		var mapUpdated bool
		sightingKml, err := db.LoadSightingKml(t.dbConn, observation.sighting)
		if err == sql.ErrNoRows {
			log.Debugf("[session %d] creating KML for %s", project.Session.Id, sighting.a.Icao)
			// Can be created
			_, err = db.CreateSightingKml(t.dbConn, observation.sighting, kmlStr)
			if err != nil {
				err = errors.Wrap(err, "create sighting kml")
			}
		} else if err == nil {
			mapUpdated = true
			log.Debugf("[session %d] updating KML for %s", project.Session.Id, sighting.a.Icao)
			_, err = db.UpdateSightingKml(t.dbConn, sightingKml, kmlStr)
			if err != nil {
				err = errors.Wrap(err, "update sighting kml")
			}
		}

		if err != nil {
			return err
		}

		sp := email.MapProducedParameters{
			Project:      project.Name,
			Icao:         sighting.a.Icao,
			StartTimeFmt: startTimeFmt,
			EndTimeFmt:   endTimeFmt,
			DurationFmt:  sightingDuration.String(),
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
		if observation.sighting.CallSign != nil {
			sp.CallSign = *observation.sighting.CallSign
		}

		if project.IsEmailNotificationEnabled(MapProduced) {
			log.Debugf("[session %d] %s: sending %s notification", project.Session.Id, sighting.a.Icao, MapProduced)
			msg, err := email.PrepareMapProducedEmail(t.mailTemplates, project.NotifyEmail, kmlStr, sp)
			if err != nil {
				return err
			}
			err = t.opt.Mailer.Queue(*msg)
			if err != nil {
				return nil
			}
		}
	}

	return nil
}

func (t *Tracker) startConsumer(ctx context.Context, msgs chan *pb.Message) {
	defer t.consumerWG.Done()

	for msg := range msgs {
		inflightMsgVec.WithLabelValues().Inc()
		t.projectMu.RLock()
		for _, proj := range t.projects {
			err := t.ProcessMessage(proj, msg)
			if err != nil {
				t.projectMu.RUnlock()
				panic(err)
			}
		}
		t.projectMu.RUnlock()
		aircraftCountVec.WithLabelValues().Set(float64(len(t.sighting)))
		inflightMsgVec.WithLabelValues().Dec()
		msgsProcessed.Inc()
	}
}

// getSighting returns an existing Sighting if present,
// and creates a new one if missing. It locks sightingMu
// for this operation. The sighting will be returned
// Locked, so must be unlocked by the caller when finished.
func (t *Tracker) getSighting(icao string) *Sighting {
	// init sighting in map
	t.sightingMu.Lock()
	defer t.sightingMu.Unlock()
	s, ok := t.sighting[icao]
	if !ok {
		s = NewSighting(icao)
		t.sighting[icao] = s
	}

	s.mu.Lock()
	return s
}

func (t *Tracker) ProcessMessage(project *Project, msg *pb.Message) error {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000 // make microseconds
		msgDurations.Observe(us)
	}))
	defer timer.ObserveDuration()

	s := t.getSighting(msg.Icao)
	defer s.mu.Unlock()

	now := time.Now()
	s.lastSeen = now

	// Update Sighting state
	var err error
	if msg.Altitude != "" {
		var alt int64
		alt, err = strconv.ParseInt(msg.Altitude, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "parse altitude")
		}
		s.State.HaveAltitude = true
		s.State.Altitude = alt
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
	}
	if msg.Squawk != "" && msg.Squawk != s.State.Squawk {
		s.State.HaveSquawk = true
		s.State.Squawk = msg.Squawk
	}
	if msg.VerticalRate != "" {
		var vr int64
		vr, err = strconv.ParseInt(msg.VerticalRate, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "parse vertical rate")
		}
		s.State.HaveVerticalRate = true
		s.State.VerticalRate = vr
		if vr == 0 && s.Tags.IsInTakeoff && (s.State.HaveAltitude && s.State.Altitude > 200) {
			s.Tags.IsInTakeoff = false
			log.Tracef("ac finished takeoff %s (Alt: %d, VerticalRate: %d)",
				s.State.Icao, s.State.Altitude, vr)
		}
	}
	if s.State.IsOnGround != msg.IsOnGround {
		if s.onGroundCandidate == msg.IsOnGround {
			s.onGroundCounter++
			if s.onGroundCounter > t.opt.OnGroundUpdateThreshold {
				log.Tracef("%s: updated IsOnGround: %t -> %t", s.State.Icao, s.State.IsOnGround, msg.IsOnGround)
				s.State.IsOnGround = msg.IsOnGround
				if !s.State.IsOnGround && s.State.VerticalRate > 0 {
					log.Tracef("%s: IsInTakeoff (Alt: %d, VerticalRate: %d)", s.State.Icao, s.State.Altitude, s.State.VerticalRate)
					s.Tags.IsInTakeoff = true
				}
			} else {
				log.Tracef("%s: IsOnGround confirmation %t %d", s.State.Icao, s.onGroundCandidate, s.onGroundCounter)
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
		s.a, err = t.loadAircraft(s.State.Icao)
		if err != nil {
			return err
		}
	}

	// initialize project sighting, if not already done
	observation, ok := s.observedBy[project.Session.Id]
	var firstSighting bool
	if !ok {
		sighting, isReopened, err := t.initProjectSighting(project, s.a)
		if err != nil {
			// Cleanup reserved sighting
			return errors.Wrapf(err, "failed to init project sighting")
		}
		observation = NewProjectObservation(project, sighting, now)
		s.observedBy[project.Session.Id] = observation
		firstSighting = !isReopened
	}

	// Update Projects information in DB
	observation.lastSeen = now

	var updatedAlt, updatedLocation, updatedCallSign, updatedSquawk bool
	if s.State.HaveAltitude {
		updatedAlt = !observation.haveAlt || (s.State.Altitude != observation.altitude)
		if updatedAlt {
			observation.altitude = s.State.Altitude
			observation.haveAlt = true
		}
	}
	if s.State.HaveLocation {
		updatedLocation = !observation.haveLocation || (s.State.Latitude != observation.latitude || s.State.Longitude != observation.longitude)
		if updatedLocation {
			observation.latitude = s.State.Latitude
			observation.longitude = s.State.Longitude
			observation.haveLocation = true
		}
	}
	if s.State.HaveCallsign {
		updatedCallSign = observation.sighting.CallSign == nil || (s.State.CallSign != *observation.sighting.CallSign)
	}
	if s.State.HaveSquawk {
		updatedSquawk = observation.sighting.Squawk == nil || (s.State.Squawk != *observation.sighting.Squawk)
	}

	if s.Tags.IsInTakeoff != observation.tags.IsInTakeoff {
		observation.tags.IsInTakeoff = s.Tags.IsInTakeoff
		if project.IsFeatureEnabled(TrackTakeoff) {
			if observation.tags.IsInTakeoff {
				if project.IsEmailNotificationEnabled(TakeoffStart) {

				}
				log.Infof("[session %d] %s: has begun takeoff",
					project.Session.Id, s.State.Icao)
			} else {
				if project.IsEmailNotificationEnabled(TakeoffComplete) {

				}
				log.Infof("[session %d] %s: has finished takeoff",
					project.Session.Id, s.State.Icao)
			}
		}
	}

	if firstSighting && project.IsEmailNotificationEnabled(SpottedInFlight) {
		log.Debugf("[session %d] %s: sending %s notification", project.Session.Id, s.State.Icao, SpottedInFlight)
		msg, err := email.PrepareSpottedInFlightEmail(t.mailTemplates, project.NotifyEmail, email.SpottedInFlightParameters{
			Project:      project.Name,
			Icao:         s.State.Icao,
			CallSign:     s.State.CallSign,
			StartTime:    s.firstSeen,
			StartTimeFmt: s.firstSeen.Format(time.RFC1123Z),
		})
		if err != nil {
			return err
		}
		err = t.opt.Mailer.Queue(*msg)
		if err != nil {
			return nil
		}
	}

	if s.State.HaveAltitude && s.State.HaveLocation && project.IsFeatureEnabled(TrackKmlLocation) {
		// We have alt + location, and there's been an update - save the location
		if (observation.haveAlt && observation.haveLocation) && (updatedAlt || updatedLocation) {
			_, err = db.InsertSightingLocation(t.dbConn, observation.sighting.Id, now,
				observation.altitude, observation.latitude, observation.longitude)
			if err != nil {
				return err
			}
			observation.locationCount++
			log.Infof("[session %d] %s: new position: altitude %dft, position (%f, %f)",
				project.Session.Id, msg.Icao, observation.altitude, observation.latitude, observation.longitude)
		}
	}

	if updatedCallSign && project.IsFeatureEnabled(TrackCallSigns) {
		txExecer := db.NewTxExecer(t.dbConn, func(tx *sql.Tx) error {
			res, err := db.UpdateSightingCallsignTx(tx, observation.sighting, s.State.CallSign)
			if err != nil {
				return errors.Wrap(err, "updating sighting callsign")
			} else if err = db.CheckRowsUpdated(res, 1); err != nil {
				return err
			}
			_, err = db.CreateNewSightingCallSignTx(tx, observation.sighting, s.State.CallSign, now)
			if err != nil {
				return errors.Wrap(err, "creating callsign record")
			}
			return nil
		})
		err := txExecer.Exec()
		if err != nil {
			return errors.Wrap(err, "saving callsign")
		}
		if observation.sighting.CallSign == nil {
			log.Infof("[session %d] %s: found callsign %s", project.Session.Id, s.State.Icao, s.State.CallSign)
		} else {
			log.Infof("[session %d] %s: updated callsign %s -> %s", project.Session.Id, s.State.Icao, *observation.sighting.CallSign, s.State.CallSign)
		}
		observation.sighting.CallSign = &msg.CallSign
	}

	if updatedSquawk && project.IsFeatureEnabled(TrackSquawks) {
		if observation.sighting.Squawk == nil {
			log.Infof("[session %d] %s: found squawk %s", project.Session.Id, s.State.Icao, s.State.Squawk)
		} else {
			log.Infof("[session %d] %s: updated squawk %s -> %s", project.Session.Id, s.State.Icao, *observation.sighting.Squawk, s.State.Squawk)
		}

		// db
		txExecer := db.NewTxExecer(t.dbConn, func(tx *sql.Tx) error {
			res, err := db.UpdateSightingSquawkTx(tx, observation.sighting, s.State.Squawk)
			// todo: add this back in once we have 'sighting restore' added.
			// reopened sightings trigger update though row count will be zero
			if err != nil {
				return errors.Wrap(err, "updating sighting squawk")
			}
			//else if err = db.CheckRowsUpdated(res, 1); err != nil {
			//	return err
			//}
			affected, err := res.RowsAffected()
			if err != nil {
				return err
			}
			// work around - don't repeatedly save squawks, til we deal with 'sighting restore'
			if affected > 0 {
				_, err = db.CreateNewSightingSquawkTx(tx, observation.sighting, s.State.Squawk, now)
				if err != nil {
					return errors.Wrap(err, "creating squawk record")
				}
			}
			return nil
		})
		err := txExecer.Exec()
		if err != nil {
			return errors.Wrap(err, "saving squawk")
		}
		observation.sighting.Squawk = &s.State.Squawk
		// end of db
	}

	if project.IsFeatureEnabled(GeocodeEndpoints) && observation.origin == nil && observation.haveLocation {
		if observation.altitude > t.opt.NearestAirportMaxAltitude {
			// too high for an airport
			log.Debugf("[session %d] %s: too high to determine origin location (%d over max %d)",
				project.Session.Id, s.State.Icao, observation.altitude, t.opt.NearestAirportMaxAltitude)
			observation.origin = &GeocodeLocation{}
		} else {
			location, distance, err := t.reverseGeocode(observation.latitude, observation.longitude)
			if err != nil {
				return errors.Wrap(err, "searching origin")
			}
			if location.ok {
				log.Debugf("[session %d] %s: Origin reverse geocode result: (%f, %f): %s %.1f km",
					project.Session.Id, s.State.Icao, observation.latitude, observation.longitude, location.address,
					distance/1000)
			} else {
				log.Debugf("[session %d] %s: Reverse geocode search for origin (%f, %f) yielded no results",
					project.Session.Id, s.State.Icao, observation.latitude, observation.longitude)
			}
			observation.origin = location
		}
	}

	return nil
}

func (t *Tracker) loadAircraft(icao string) (*db.Aircraft, error) {
	// create sighting
	a, err := db.LoadAircraftByIcao(t.dbConn, icao)
	if err == sql.ErrNoRows {
		acRes, err := db.CreateAircraft(t.dbConn, icao)
		if err != nil {
			return nil, err
		}
		acId, err := acRes.LastInsertId()
		if err != nil {
			return nil, err
		}
		a, err = db.LoadAircraftById(t.dbConn, acId)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return a, nil
}

func (t *Tracker) initProjectSighting(p *Project, ac *db.Aircraft) (*db.Sighting, bool, error) {
	s, err := db.LoadLastSighting(t.dbConn, p.Session, ac)
	if err != nil {
		return nil, false, err
	}

	if p.ReopenSightings && s != nil {
		if s.ClosedAt == nil {
			return nil, false, errors.Errorf("last session for %s (id=%d) is still open - possibly running multiple instances of this software", ac.Icao, s.Id)
		}
		timeSinceClosed := time.Since(*s.ClosedAt)
		// reactivate session if it's within our interval
		// todo: maybe other checks here, like, finished_on_ground or something to avoid rapid stops, so
		// we break up legs of the journey
		if timeSinceClosed < p.ReopenSightingsInterval {
			res, err := db.ReopenSighting(t.dbConn, s)
			if err != nil {
				return nil, false, errors.Wrap(err, "reopening sighting")
			} else if err = db.CheckRowsUpdated(res, 1); err != nil {
				return nil, false, err
			}

			log.Infof("[session %d] %s: reopened sighting after %s", p.Session.Id, ac.Icao, timeSinceClosed)
			return s, true, nil
		}
	}

	// A new sighting is needed
	res, err := db.CreateSighting(t.dbConn, p.Session, ac)
	if err != nil {
		return nil, false, errors.Wrapf(err, "creating sighting record failed")
	}
	sightingId, err := res.LastInsertId()
	if err != nil {
		return nil, false, errors.Wrapf(err, "fetching sighting id failed")
	}
	s, err = db.LoadSightingById(t.dbConn, sightingId)
	if err != nil {
		return nil, false, errors.Wrapf(err, "load sighting by id failed")
	}

	if s != nil && s.ClosedAt != nil {
		log.Infof("[session %d] %s: new sighting (last seen: %s)", p.Session.Id, ac.Icao, *s.ClosedAt)
	} else {
		log.Infof("[session %d] %s: new sighting", p.Session.Id, ac.Icao)
	}

	return s, false, nil
}

func checkIfPassesFilter(prg cel.Program, msg *pb.Message, state *pb.State) (bool, error) {
	filterTimer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000 // make microseconds
		filterDurations.Observe(us)
	}))
	defer filterTimer.ObserveDuration()

	out, _, err := prg.Eval(map[string]interface{}{
		"msg":   msg,
		"state": state,
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
