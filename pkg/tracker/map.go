package tracker

import "C"
import (
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	DefaultHistoryInterval = time.Second * 30
)

var mapServices struct {
	sync.RWMutex
	s []MapService
}

// UnknownProject is returned by MapAccess.GetProjectAircraft
var UnknownProject = errors.New("unknown project")

type MapAccess interface {
	GetProjectAircraft(projectName string, f func(int64, []*JsonAircraft) error) error
}
type MapHistoryUpdateScheduler interface {
	UpdateHistory(projects []string) error
}
type MapService interface {
	MapService() string
	RegisterRoutes(r *mux.Router) error
	UpdateScheduler() MapHistoryUpdateScheduler
}

func RegisterMapBackend(service MapService) error {
	mapServices.Lock()
	defer mapServices.Unlock()
	n := len(mapServices.s)
	name := service.MapService()
	for i := 0; i < n; i++ {
		if mapServices.s[i].MapService() == name {
			return errors.Errorf("%s map service already registered", name)
		}
	}

	mapServices.s = append(mapServices.s, service)
	return nil
}
func GetMapBackend(service string) (MapService, bool) {
	mapServices.RLock()
	defer mapServices.RUnlock()
	n := len(mapServices.s)
	for i := 0; i < n; i++ {
		if mapServices.s[i].MapService() == service {
			return mapServices.s[i], true
		}
	}
	return nil, false
}

type MapProjectStatusListener struct {
	m *AircraftMap
}

func NewMapProjectStatusListener(m *AircraftMap) *MapProjectStatusListener {
	return &MapProjectStatusListener{m: m}
}
func (p *MapProjectStatusListener) Activated(project *Project) {
	p.m.registerProject(project)
}
func (p *MapProjectStatusListener) Deactivated(project *Project) {
	p.m.deregisterProject(project)
}

type MapProjectAircraftUpdateListener struct {
	m *AircraftMap
}

func NewMapProjectAircraftUpdateListener(m *AircraftMap) *MapProjectAircraftUpdateListener {
	return &MapProjectAircraftUpdateListener{m}
}

func (l *MapProjectAircraftUpdateListener) NewAircraft(p *Project, s *Sighting) {
	l.m.projectNewAircraft(p, s)
}
func (l *MapProjectAircraftUpdateListener) UpdatedAircraft(p *Project, s *Sighting) {
	l.m.projectUpdatedAircraft(p, s)
}
func (l *MapProjectAircraftUpdateListener) LostAircraft(p *Project, s *Sighting) {
	l.m.projectLostAircraft(p, s)
}

type projectState struct {
	sync.RWMutex
	name     string
	aircraft []string
}

type JsonAircraft struct {
	sync.RWMutex
	referenceCount int64
	lastPosTime    time.Time
	lastMsgTime    time.Time
	// hex: the 24-bit ICAO identifier of the aircraft, as 6 hex digits. The identifier may start with '~', this means that the address is a non-ICAO address (e.g. from TIS-B).
	Hex string `json:"hex"`
	// type: type of underlying message, one of:
	// adsb_icao: messages from a Mode S or ADS-B transponder, using a 24-bit ICAO address
	// adsb_icao_nt: messages from an ADS-B equipped "non-transponder" emitter e.g. a ground vehicle, using a 24-bit ICAO address
	// adsr_icao: rebroadcast of ADS-B messages originally sent via another data link e.g. UAT, using a 24-bit ICAO address
	// tisb_icao: traffic information about a non-ADS-B target identified by a 24-bit ICAO address, e.g. a Mode S target tracked by secondary radar
	// adsb_other: messages from an ADS-B transponder using a non-ICAO address, e.g. anonymized address
	// adsr_other: rebroadcast of ADS-B messages originally sent via another data link e.g. UAT, using a non-ICAO address
	// tisb_other: traffic information about a non-ADS-B target using a non-ICAO address
	// tisb_trackfile: traffic information about a non-ADS-B target using a track/file identifier, typically from primary or Mode A/C radar
	Type string `json:"type,omitempty"`
	// flight: callsign, the flight name or aircraft registration as 8 chars (2.2.8.2.6)
	Flight string `json:"flight,omitempty"`
	// alt_baro: the aircraft barometric altitude in feet
	BarometricAltitude int64 `json:"alt_baro,omitempty"`
	// alt_geom: geometric (GNSS / INS) altitude in feet referenced to the WGS84 ellipsoid
	GeometricAltitude string `json:"alt_geom,omitempty"`
	// gs: ground speed in knots
	GroundSpeed float64 `json:"gs,omitempty"`
	// ias: indicated air speed in knots
	IndicatedAirSpeed int64 `json:"ias,omitempty"`
	// tas: true air speed in knots
	TrueAirSpeed int64 `json:"tas,omitempty"`
	// mach: Mach number
	Mach float64 `json:"mach,omitempty"`
	// track: true track over ground in degrees (0-359)
	Track float64 `json:"track,omitempty"`
	// track_rate: Rate of change of track, degrees/second
	TrackRate float64 `json:"track_rate,omitempty"`
	// roll: Roll, degrees, negative is left roll
	Roll float64 `json:"roll,omitempty"`
	// mag_heading: Heading, degrees clockwise from magnetic north
	MagneticHeading float64 `json:"mag_heading,omitempty"`
	// true_heading: Heading, degrees clockwise from true north
	TrueHeading float64 `json:"true_heading,omitempty"`
	// baro_rate: Rate of change of barometric altitude, feet/minute
	BarometricRate int64 `json:"baro_rate,omitempty"`
	// geom_rate: Rate of change of geometric (GNSS / INS) altitude, feet/minute
	GeometricRate int64 `json:"geom_rate,omitempty"`
	// squawk: Mode A code (Squawk), encoded as 4 octal digits
	Squawk string `json:"squawk,omitempty"`
	// emergency: ADS-B emergency/priority status, a superset of the 7x00 squawks (2.2.3.2.7.8.1.1)
	Emergency string `json:"emergency,omitempty"`
	// category: emitter category to identify particular aircraft or vehicle classes (values A0 - D7) (2.2.3.2.5.2)
	Category string `json:"category,omitempty"`
	// nav_qnh: altimeter setting (QFE or QNH/QNE), hPa
	NavQNH string `json:"nav_qnh,omitempty"`
	// nav_altitude_mcp: selected altitude from the Mode Control Panel / Flight Control Unit (MCP/FCU) or equivalent equipment
	NavAltitudeMCP int64 `json:"nav_altitude_mcp,omitempty"`
	// nav_altitude_fms: selected altitude from the Flight Manaagement System (FMS) (2.2.3.2.7.1.3.3)
	NavAltitudeFMS int64 `json:"nav_altitude_fms,omitempty"`
	// nav_heading: selected heading (True or Magnetic is not defined in DO-260B, mostly Magnetic as that is the de facto standard) (2.2.3.2.7.1.3.7)
	NavHeading string `json:"nav_heading,omitempty"`
	// nav_modes: set of engaged automation modes: 'autopilot', 'vnav', 'althold', 'approach', 'lnav', 'tcas'
	NavModes string `json:"nav_modes,omitempty"`
	// lat, lon: the aircraft position in decimal degrees
	Latitude  float64 `json:"lat,omitempty"`
	Longitude float64 `json:"lon,omitempty"`
	// nic: Navigation Integrity Category (2.2.3.2.7.2.6)
	Nic string `json:"nic,omitempty"`
	// rc: Radius of Containment, meters; a measure of position integrity derived from NIC & supplementary bits. (2.2.3.2.7.2.6, Table 2-69)
	RadiusOfContainment int64 `json:"rc,omitempty"`
	// seen_pos: how long ago (in seconds before "now") the position was last updated
	SeenPos float64 `json:"seen_pos,omitempty"`
	// version: ADS-B Version Number 0, 1, 2 (3-7 are reserved) (2.2.3.2.7.5)
	Version int64 `json:"version,omitempty"`
	// nic_baro: Navigation Integrity Category for Barometric Altitude (2.2.5.1.35)
	NicBaro int64 `json:"nic_baro,omitempty"`
	// nac_p: Navigation Accuracy for Position (2.2.5.1.35)
	NacP int64 `json:"nac_p,omitempty"`
	// nac_v: Navigation Accuracy for Velocity (2.2.5.1.19)
	NacV string `json:"nac_v,omitempty"`
	// sil: Source Integity Level (2.2.5.1.40)
	Sil int64 `json:"sil,omitempty"`
	// sil_type: interpretation of SIL: unknown, perhour, persample
	SilType string `json:"sil_type,omitempty"`
	// gva: Geometric Vertical Accuracy  (2.2.3.2.7.2.8)
	GVA int64 `json:"gva,omitempty"`
	// sda: System Design Assurance (2.2.3.2.7.2.4.6)
	SDA int64 `json:"sda,omitempty"`
	// mlat: list of fields derived from MLAT data
	//MLAT string `json:"mlat"`
	// tisb: list of fields derived from TIS-B data
	//TISB string `json:"tisb"`

	// messages: total number of Mode S messages received from this aircraft
	Messages int64 `json:"messages"`
	// seen: how long ago (in seconds before "now") a message was last received from this aircraft
	Seen int64 `json:"seen"`
	// rssi: recent average RSSI (signal power), in dbFS; this will always be negative.
	Rssi float64 `json:"rssi"`
}
type AircraftMap struct {
	s               *http.Server
	r               *mux.Router
	historyInterval time.Duration
	ac              map[string]*JsonAircraft
	acMu            sync.RWMutex
	projMu          sync.RWMutex
	projects        map[string]*projectState
	services        []MapService
	messages        int64
}

func NewAircraftMap(cfg config.MapSettings) (*AircraftMap, error) {
	m := &AircraftMap{
		ac:              make(map[string]*JsonAircraft),
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
		Addr:    fmt.Sprintf("%s:%d", cfg.Address, port),
		Handler: handlers.CORS()(m.r),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return m, nil
}
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
func (m *AircraftMap) registerProject(p *Project) error {
	m.projMu.Lock()
	defer m.projMu.Unlock()
	if _, ok := m.projects[p.Name]; ok {
		return errors.New("duplicate project")
	}
	m.projects[p.Name] = &projectState{
		name:     p.Name,
		aircraft: []string{},
	}
	return nil
}
func (m *AircraftMap) deregisterProject(p *Project) error {
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
		icao := proj.aircraft[i]
		m.dereferenceAircraft(icao)
	}
	proj.aircraft = nil
	delete(m.projects, p.Name)
	return nil
}
func (m *AircraftMap) dereferenceAircraft(icao string) bool {
	m.ac[icao].referenceCount--
	if m.ac[icao].referenceCount == 0 {
		delete(m.ac, icao)
		return true
	}
	return false
}
func (m *AircraftMap) projectNewAircraft(p *Project, s *Sighting) error {
	m.projMu.RLock()
	defer m.projMu.RUnlock()

	m.acMu.Lock()
	if _, ok := m.ac[s.State.Icao]; !ok {
		m.ac[s.State.Icao] = &JsonAircraft{
			lastMsgTime:     time.Now(),
			Hex:             s.State.Icao,
			Flight:          s.State.CallSign,
			Latitude:        s.State.Latitude,
			Longitude:       s.State.Longitude,
			Squawk:          s.State.Squawk,
			MagneticHeading: s.State.Track,
			Track:           s.State.Track,
			BarometricRate:  s.State.VerticalRate,
			GroundSpeed:     s.State.GroundSpeed,
			Messages:        1,
		}
	}
	m.ac[s.State.Icao].referenceCount++
	m.acMu.Unlock()

	// Associate record with project (init project if necessary)
	state := m.projects[p.Name]
	state.Lock()
	state.aircraft = append(state.aircraft, s.State.Icao)
	state.Unlock()

	atomic.AddInt64(&m.messages, 1)
	return nil
}
func (m *AircraftMap) projectUpdatedAircraft(p *Project, s *Sighting) error {
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
	acRecord.Flight = s.State.CallSign
	acRecord.BarometricAltitude = s.State.Altitude
	acRecord.Latitude = s.State.Latitude
	acRecord.Longitude = s.State.Longitude
	acRecord.Squawk = s.State.Squawk
	acRecord.MagneticHeading = s.State.Track
	acRecord.Track = s.State.Track
	acRecord.GroundSpeed = s.State.GroundSpeed
	atomic.AddInt64(&m.messages, 1)
	return nil
}

func (m *AircraftMap) GetProjectAircraft(projectName string, f func(int64, []*JsonAircraft) error) error {
	m.projMu.RLock()
	defer m.projMu.RUnlock()
	m.acMu.RLock()
	defer m.acMu.RUnlock()

	proj, ok := m.projects[projectName]
	if !ok {
		return UnknownProject
	}

	proj.RLock()
	defer proj.RUnlock()

	numAC := len(proj.aircraft)
	l := make([]*JsonAircraft, numAC)
	for i := 0; i < numAC; i++ {
		l[i] = m.ac[proj.aircraft[i]]
		l[i].RLock()
	}
	defer func() {
		for i := 0; i < numAC; i++ {
			l[i].RUnlock()
		}
	}()

	err := f(m.messages, l)
	if err != nil {
		return err
	}
	return nil
}
func (m *AircraftMap) projectLostAircraft(p *Project, s *Sighting) error {
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

func (m *AircraftMap) Serve() {
	go func() {
		lastHistoryUpdate := time.Now()
		firstRun := true
		for {
			<-time.After(time.Second)
			m.acMu.RLock()
			for _, ac := range m.ac {
				ac.Lock()
				ac.Seen = int64(time.Since(ac.lastMsgTime).Seconds())
				ac.SeenPos = time.Since(ac.lastMsgTime).Seconds()
				ac.Unlock()
			}
			m.acMu.RUnlock()
			if firstRun || time.Since(lastHistoryUpdate) > m.historyInterval {
				if firstRun {
					firstRun = false
				}

				m.projMu.RLock()
				projects := make([]string, 0, len(m.projects))
				for _, project := range m.projects {
					projects = append(projects, project.name)
				}
				for _, service := range m.services {
					service.UpdateScheduler().UpdateHistory(projects)
				}
				m.projMu.RUnlock()
				lastHistoryUpdate = time.Now()
			}
		}
	}()
	go func() {
		m.s.ListenAndServe()
	}()
}
