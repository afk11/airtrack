package tracker

import "C"
import (
	"context"
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultHistoryInterval - the default time between history
	// updates
	DefaultHistoryInterval = time.Second * 30
	// DefaultHistoryFileCount - the default number of history
	// files to keep
	DefaultHistoryFileCount = 60
)

// ErrUnknownProject is returned by MapAccess.GetProjectAircraft
var ErrUnknownProject = errors.New("unknown project")

type (
	// MapAccess provides access to the current map view
	MapAccess interface {
		// GetProjectAircraft loads the subset of aircraft in
		// view for this project, and the current message count
		// and []*JSONAircraft is passed to the provided closure.
		// The aircraft are locked for the lifetime of the closure
		// and must be copied to be safely used elsewhere.
		GetProjectAircraft(projectName string, f func(int64, []*JSONAircraft) error) error
	}

	// MapHistoryUpdateScheduler defines an interface allowing
	// AircraftMap trigger MapServices to save a new history file
	// Used by AircraftMap
	MapHistoryUpdateScheduler interface {
		// UpdateHistory triggers the MapService to save
		// a new history file
		UpdateHistory(projects []string) error
	}

	// MapService is a contract for map backends.
	MapService interface {
		// MapService returns the name of the backend
		MapService() string
		// RegisterRoutes allows the MapService to add
		// it's map related routes to the router
		RegisterRoutes(r *mux.Router) error
		// UpdateHistory is used to
		UpdateHistory(projNames []string) error
	}

	// MapProjectStatusListener implements the ProjectStatusListener
	// allowing tracker to notify us about new or closed projects
	MapProjectStatusListener struct {
		m *AircraftMap
	}
)

// NewMapProjectStatusListener creates a new *MapProjectStatusListener
func NewMapProjectStatusListener(m *AircraftMap) *MapProjectStatusListener {
	return &MapProjectStatusListener{m: m}
}

// Activated - see ProjectStatusListener.Activated. This function
// informs the map about a new project to track.
func (p *MapProjectStatusListener) Activated(project *Project) {
	p.m.registerProject(project)
}

// Deactivated - implements ProjectStatusListener.Activated. This
// function removes the data about this project from the map service.
func (p *MapProjectStatusListener) Deactivated(project *Project) {
	p.m.deregisterProject(project)
}

// MapProjectAircraftUpdateListener implements ProjectAircraftUpdateListener
// and dispatches notifications to the AircraftMap so they can be applied to the map.
type MapProjectAircraftUpdateListener struct {
	m *AircraftMap
}

// NewMapProjectAircraftUpdateListener returns a new MapProjectAircraftUpdateListener
func NewMapProjectAircraftUpdateListener(m *AircraftMap) *MapProjectAircraftUpdateListener {
	return &MapProjectAircraftUpdateListener{m}
}

// NewAircraft informs map about new aircraft. Implements
// ProjectAircraftUpdateListener.NewAircraft
func (l *MapProjectAircraftUpdateListener) NewAircraft(p *Project, s *Sighting) {
	l.m.projectNewAircraft(p, s)
}

// UpdatedAircraft informs map about updated aircraft. Implements
// ProjectAircraftUpdateListener.UpdatedAircraft
func (l *MapProjectAircraftUpdateListener) UpdatedAircraft(p *Project, s *Sighting) {
	l.m.projectUpdatedAircraft(p, s)
}

// LostAircraft informs map about lost aircraft. Implements
// ProjectAircraftUpdateListener.LostAircraft
func (l *MapProjectAircraftUpdateListener) LostAircraft(p *Project, s *Sighting) {
	l.m.projectLostAircraft(p, s)
}

// projectState stores the subset of the map this project
// is tracking
type projectState struct {
	sync.RWMutex
	name     string
	aircraft []string
}

// JSONAircraft is a dump1090 aircraft structure
type JSONAircraft struct {
	sync.RWMutex
	// referenceCount tracks how many projects currently refer
	// to this aircraft
	referenceCount int64
	// last time our position was updated
	lastPosTime time.Time
	// last time we received a message
	lastMsgTime time.Time
	// Hex: the 24-bit ICAO identifier of the aircraft, as 6 hex digits. The identifier may start with '~', this means that the address is a non-ICAO address (e.g. from TIS-B).
	Hex string `json:"hex"`
	// Type: type of underlying message, one of:
	// adsb_icao: messages from a Mode S or ADS-B transponder, using a 24-bit ICAO address
	// adsb_icao_nt: messages from an ADS-B equipped "non-transponder" emitter e.g. a ground vehicle, using a 24-bit ICAO address
	// adsr_icao: rebroadcast of ADS-B messages originally sent via another data link e.g. UAT, using a 24-bit ICAO address
	// tisb_icao: traffic information about a non-ADS-B target identified by a 24-bit ICAO address, e.g. a Mode S target tracked by secondary radar
	// adsb_other: messages from an ADS-B transponder using a non-ICAO address, e.g. anonymized address
	// adsr_other: rebroadcast of ADS-B messages originally sent via another data link e.g. UAT, using a non-ICAO address
	// tisb_other: traffic information about a non-ADS-B target using a non-ICAO address
	// tisb_trackfile: traffic information about a non-ADS-B target using a track/file identifier, typically from primary or Mode A/C radar
	Type string `json:"type,omitempty"`
	// Flight: callsign, the flight name or aircraft registration as 8 chars (2.2.8.2.6)
	Flight string `json:"flight,omitempty"`
	// BarometricAltitude: the aircraft barometric altitude in feet
	BarometricAltitude int64 `json:"alt_baro,omitempty"`
	// GeometricAltitude: geometric (GNSS / INS) altitude in feet referenced to the WGS84 ellipsoid
	GeometricAltitude int64 `json:"alt_geom,omitempty"`
	// GroundSpeed: ground speed in knots
	GroundSpeed float64 `json:"gs,omitempty"`
	// IndicatedAirSpeed: indicated air speed in knots
	IndicatedAirSpeed uint64 `json:"ias,omitempty"`
	// TrueAirSpeed: true air speed in knots
	TrueAirSpeed uint64 `json:"tas,omitempty"`
	// Mach: Mach number
	Mach float64 `json:"mach,omitempty"`
	// Track: true track over ground in degrees (0-359)
	Track float64 `json:"track,omitempty"`
	// TrackRate: Rate of change of track, degrees/second
	TrackRate float64 `json:"track_rate,omitempty"`
	// Roll: Roll, degrees, negative is left roll
	Roll float64 `json:"roll,omitempty"`
	// MagneticHeading: Heading, degrees clockwise from magnetic north
	MagneticHeading float64 `json:"mag_heading,omitempty"`
	// TrueHeading: Heading, degrees clockwise from true north
	TrueHeading float64 `json:"true_heading,omitempty"`
	// BarometricRate: Rate of change of barometric altitude, feet/minute
	BarometricRate int64 `json:"baro_rate,omitempty"`
	// GeometricRate: Rate of change of geometric (GNSS / INS) altitude, feet/minute
	GeometricRate int64 `json:"geom_rate,omitempty"`
	// Squawk: Mode A code (Squawk), encoded as 4 octal digits
	Squawk string `json:"squawk,omitempty"`
	// Emergency: ADS-B emergency/priority status, a superset of the 7x00 squawks (2.2.3.2.7.8.1.1)
	Emergency string `json:"emergency,omitempty"`
	// Category: emitter category to identify particular aircraft or vehicle classes (values A0 - D7) (2.2.3.2.5.2)
	Category string `json:"category,omitempty"`
	// NavQNH: altimeter setting (QFE or QNH/QNE), hPa
	NavQNH string `json:"nav_qnh,omitempty"`
	// NavAltitudeMCP: selected altitude from the Mode Control Panel / Flight Control Unit (MCP/FCU) or equivalent equipment
	NavAltitudeMCP int64 `json:"nav_altitude_mcp,omitempty"`
	// NavAltitudeFMS: selected altitude from the Flight Manaagement System (FMS) (2.2.3.2.7.1.3.3)
	NavAltitudeFMS int64 `json:"nav_altitude_fms,omitempty"`
	// NavHeading: selected heading (True or Magnetic is not defined in DO-260B, mostly Magnetic as that is the de facto standard) (2.2.3.2.7.1.3.7)
	NavHeading float64 `json:"nav_heading,omitempty"`
	// NavModes: set of engaged automation modes: 'autopilot', 'vnav', 'althold', 'approach', 'lnav', 'tcas'
	NavModes string `json:"nav_modes,omitempty"`
	// Latitude: the aircraft position in decimal degrees
	Latitude float64 `json:"lat,omitempty"`
	// Longitude: the aircraft longitude in decimal degrees
	Longitude float64 `json:"lon,omitempty"`
	// Nic: Navigation Integrity Category (2.2.3.2.7.2.6)
	Nic string `json:"nic,omitempty"`
	// RadiusOfContainment: Radius of Containment, meters; a measure of position integrity derived from NIC & supplementary bits. (2.2.3.2.7.2.6, Table 2-69)
	RadiusOfContainment int64 `json:"rc,omitempty"`
	// SeenPos: how long ago (in seconds before "now") the position was last updated
	SeenPos float64 `json:"seen_pos,omitempty"`
	// Version: ADS-B Version Number 0, 1, 2 (3-7 are reserved) (2.2.3.2.7.5)
	Version int64 `json:"version,omitempty"`
	// NicBaro: Navigation Integrity Category for Barometric Altitude (2.2.5.1.35)
	NicBaro int64 `json:"nic_baro,omitempty"`
	// NacP: Navigation Accuracy for Position (2.2.5.1.35)
	NacP int64 `json:"nac_p,omitempty"`
	// NacV: Navigation Accuracy for Velocity (2.2.5.1.19)
	NacV string `json:"nac_v,omitempty"`
	// Sil: Source Integity Level (2.2.5.1.40)
	Sil int64 `json:"sil,omitempty"`
	// SilType: interpretation of SIL: unknown, perhour, persample
	SilType string `json:"sil_type,omitempty"`
	// GVA: Geometric Vertical Accuracy  (2.2.3.2.7.2.8)
	GVA int64 `json:"gva,omitempty"`
	// SDA: System Design Assurance (2.2.3.2.7.2.4.6)
	SDA int64 `json:"sda,omitempty"`
	// MLAT: list of fields derived from MLAT data
	//MLAT string `json:"mlat"`
	// TISB: list of fields derived from TIS-B data
	//TISB string `json:"tisb"`

	// Messages: total number of Mode S messages received from this aircraft
	Messages int64 `json:"messages"`
	// Seen: how long ago (in seconds before "now") a message was last received from this aircraft
	Seen int64 `json:"seen"`
	// Rssi: recent average RSSI (signal power), in dbFS; this will always be negative.
	Rssi float64 `json:"rssi"`
}

// UpdateWithState updates JSONAircraft with the latest state
func (j *JSONAircraft) UpdateWithState(state *pb.State) {
	j.Flight = state.CallSign
	if state.LastSignal != nil {
		j.Rssi = state.LastSignal.Rssi
	}

	if state.HaveLocation {
		j.Latitude = state.Latitude
		j.Longitude = state.Longitude
	}
	if state.HaveCategory {
		j.Category = state.Category
	}
	j.Squawk = state.Squawk
	j.MagneticHeading = state.Track
	j.Track = state.Track
	j.BarometricAltitude = state.AltitudeBarometric
	j.GeometricAltitude = state.AltitudeGeometric
	if state.HaveVerticalRateBarometric {
		j.BarometricRate = state.VerticalRateBarometric
	}
	if state.HaveVerticalRateGeometric {
		j.GeometricRate = state.VerticalRateGeometric
	}
	if state.HaveFmsAltitude {
		j.NavAltitudeFMS = state.FmsAltitude
	}
	if state.HaveNavHeading {
		j.NavHeading = state.NavHeading
	}
	if state.HaveGroundSpeed {
		j.GroundSpeed = state.GroundSpeed
	}
	if state.HaveTrueAirSpeed {
		j.TrueAirSpeed = state.TrueAirSpeed
	}
	if state.HaveIndicatedAirSpeed {
		j.IndicatedAirSpeed = state.IndicatedAirSpeed
	}
	if state.HaveMach {
		j.Mach = state.Mach
	}
}

// AircraftMap is the main service for receiving project+aircraft
// updates. Implements MapAccess.
type AircraftMap struct {
	s               *http.Server
	r               *mux.Router
	historyInterval time.Duration
	ac              map[string]*JSONAircraft
	acMu            sync.RWMutex
	projMu          sync.RWMutex
	projects        map[string]*projectState
	services        []MapService
	wg              sync.WaitGroup
	canceller       func()
	messages        int64
}

// NewAircraftMap initializes a new AircraftMap using
// configuration
func NewAircraftMap(cfg *config.MapSettings) (*AircraftMap, error) {
	m := &AircraftMap{
		ac:              make(map[string]*JSONAircraft),
		projects:        make(map[string]*projectState),
		r:               mux.NewRouter(),
		historyInterval: DefaultHistoryInterval,
	}
	if cfg.HistoryInterval != 0 {
		m.historyInterval = time.Second * time.Duration(cfg.HistoryInterval)
	}
	port := uint16(8080)
	if cfg.Port != 0 {
		port = cfg.Port
	}
	m.s = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Interface, port),
		Handler: handlers.CORS()(m.r),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return m, nil
}

// RegisterMapService uses the MapService.RegisterRoutes
// to register each service's routes
func (m *AircraftMap) RegisterMapService(services ...MapService) error {
	for _, service := range services {
		sub := m.r.PathPrefix("/" + service.MapService()).Subrouter()
		err := service.RegisterRoutes(sub)
		if err != nil {
			return err
		}
	}
	m.services = append(m.services, services...)
	return nil
}

// registerProject creates a new project entry with
// no aircraft associated
func (m *AircraftMap) registerProject(p *Project) error {
	m.projMu.Lock()
	defer m.projMu.Unlock()
	if !p.ShouldMap {
		return nil
	}
	if _, ok := m.projects[p.Name]; ok {
		return errors.New("duplicate project")
	}
	m.projects[p.Name] = &projectState{
		name:     p.Name,
		aircraft: []string{},
	}
	return nil
}

// deregisterProject deletes a project and dereferences
// each aircraft.
func (m *AircraftMap) deregisterProject(p *Project) error {
	if !p.ShouldMap {
		return nil
	}
	m.projMu.Lock()
	defer m.projMu.Unlock()
	m.acMu.Lock()
	defer m.acMu.Unlock()

	proj, ok := m.projects[p.Name]
	if !ok {
		return errors.New("unknown project")
	}
	numAC := len(proj.aircraft)
	for i := 0; i < numAC; i++ {
		m.dereferenceAircraft(proj.aircraft[i])
	}
	proj.aircraft = nil
	delete(m.projects, p.Name)
	return nil
}

// dereferenceAircraft takes an ICAO of a known aircraft, and
// decrements it's reference count. The aircraft record will not
// be deleted unless it's reference count reaches zero. Returns
// true if was deleted, false if references still exist.
func (m *AircraftMap) dereferenceAircraft(icao string) bool {
	m.ac[icao].referenceCount--
	if m.ac[icao].referenceCount == 0 {
		delete(m.ac, icao)
		return true
	}
	return false
}

// projectNewAircraft associates a new aircraft with
// an existing project.
func (m *AircraftMap) projectNewAircraft(p *Project, s *Sighting) error {
	if !p.ShouldMap {
		return nil
	}
	m.projMu.RLock()
	defer m.projMu.RUnlock()

	m.acMu.Lock()
	if _, ok := m.ac[s.State.Icao]; !ok {
		ac := &JSONAircraft{
			lastMsgTime: time.Now(),
			Hex:         s.State.Icao,
			Messages:    1,
		}
		ac.UpdateWithState(&s.State)
		m.ac[s.State.Icao] = ac
	}
	m.ac[s.State.Icao].referenceCount++
	m.acMu.Unlock()

	// Associate record with project
	state := m.projects[p.Name]
	state.Lock()
	state.aircraft = append(state.aircraft, s.State.Icao)
	state.Unlock()

	atomic.AddInt64(&m.messages, 1)
	return nil
}

// projectUpdateAircraft updates the json aircraft record
// with the current aircraft state
func (m *AircraftMap) projectUpdatedAircraft(p *Project, s *Sighting) error {
	if !p.ShouldMap {
		return nil
	}
	m.acMu.RLock()
	defer m.acMu.RUnlock()

	acRecord := m.ac[s.State.Icao]
	acRecord.Lock()
	defer acRecord.Unlock()
	locationUpdated := s.State.Latitude != acRecord.Latitude || s.State.Longitude != acRecord.Longitude

	acRecord.Messages++
	acRecord.lastMsgTime = time.Now()
	if locationUpdated {
		acRecord.lastPosTime = acRecord.lastMsgTime
	}
	acRecord.UpdateWithState(&s.State)
	atomic.AddInt64(&m.messages, 1)
	return nil
}

// projectLostAircraft disassociates an aircraft from a project
// (and dereferences it)
func (m *AircraftMap) projectLostAircraft(p *Project, s *Sighting) error {
	if !p.ShouldMap {
		return nil
	}
	m.projMu.RLock()
	defer m.projMu.RUnlock()

	projRecord := m.projects[p.Name]
	projRecord.Lock()
	numAc := int64(len(projRecord.aircraft))
	var toDelete int64 = -1
	for i := int64(0); i < numAc && toDelete == -1; i++ {
		if projRecord.aircraft[i] == s.State.Icao {
			toDelete = i
		}
	}
	projRecord.aircraft = append(projRecord.aircraft[:toDelete], projRecord.aircraft[toDelete+1:]...)
	projRecord.Unlock()

	// m.ac write
	m.acMu.Lock()
	m.dereferenceAircraft(s.State.Icao)
	m.acMu.Unlock()
	return nil
}

// GetProjectAircraft - Implements MapAccess.GetProjectAircraft.
func (m *AircraftMap) GetProjectAircraft(projectName string, f func(int64, []*JSONAircraft) error) error {
	m.projMu.RLock()
	defer m.projMu.RUnlock()
	m.acMu.RLock()
	defer m.acMu.RUnlock()

	proj, ok := m.projects[projectName]
	if !ok {
		return ErrUnknownProject
	}

	proj.RLock()
	defer proj.RUnlock()

	numAC := len(proj.aircraft)
	l := make([]*JSONAircraft, numAC)
	for i := 0; i < numAC; i++ {
		l[i] = m.ac[proj.aircraft[i]]
		l[i].RLock()
	}
	defer func() {
		for i := 0; i < numAC; i++ {
			l[i].RUnlock()
		}
	}()

	err := f(atomic.LoadInt64(&m.messages), l)
	if err != nil {
		return err
	}
	return nil
}

// Serve launches background services
// - aircraft Seen / SeenPos updates each second
// - triggers history updates on registered MapServices every m.historyInterval
func (m *AircraftMap) Serve() {
	m.wg.Add(1)
	ctx, canceller := context.WithCancel(context.Background())
	m.canceller = canceller
	go m.updateJSON(ctx)
	go func() {
		m.s.ListenAndServe()
	}()
}

// updateJSON does time based functions - every second it
// adjusts Seen and SeenPos to the corrent number of seconds
// since the last message/position. Every `m.historyInterval`
// it writes a new history file.
func (m *AircraftMap) updateJSON(ctx context.Context) {
	defer m.wg.Done()
	lastHistoryUpdate := time.Now()
	firstRun := true
	for {
		// Stop if stop signal received
		select {
		default:
		case <-ctx.Done():
			return
		}

		<-time.After(time.Second)
		m.acMu.RLock()
		for _, ac := range m.ac {
			ac.Lock()
			ac.Seen = int64(time.Since(ac.lastMsgTime).Seconds())
			ac.SeenPos = time.Since(ac.lastPosTime).Seconds()
			ac.Unlock()
		}
		m.acMu.RUnlock()
		if firstRun || time.Since(lastHistoryUpdate) > m.historyInterval {
			if firstRun {
				firstRun = false
			}

			m.projMu.RLock()
			projects := make([]string, 0, len(m.projects))
			for projName := range m.projects {
				projects = append(projects, projName)
			}
			for sk := range m.services {
				err := m.services[sk].UpdateHistory(projects)
				if err != nil {
					panic(err)
				}
			}
			m.projMu.RUnlock()
			lastHistoryUpdate = time.Now()
		}
	}
}

// Stop sends the stop signal to coroutines and waits for them
// to finish
func (m *AircraftMap) Stop() error {
	m.canceller()
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	err := m.s.Shutdown(ctxShutDown)
	if err != nil && err != http.ErrServerClosed {
		return errors.Wrapf(err, "in map server shutdown")
	}
	m.wg.Wait()
	return nil
}
