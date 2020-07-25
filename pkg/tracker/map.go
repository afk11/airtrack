package tracker

import (
	"encoding/json"
	"fmt"
	"github.com/afk11/airtrack/pkg/config"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
	"sync"
	"time"
)

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
	name string
	aircraft []string
}

type jsonAircraft struct {
	Now float64 `json:"now"`
	Messages int64 `json:"messages"`
	Aircraft []*jsonAircraftField `json:"aircraft"`
}
type jsonAircraftField struct {
	referenceCount int64
	lastPosTime time.Time
	lastMsgTime time.Time
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
	BarometricRate float64 `json:"baro_rate,omitempty"`
	// geom_rate: Rate of change of geometric (GNSS / INS) altitude, feet/minute
	GeometricRate float64 `json:"geom_rate,omitempty"`
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
	Latitude float64 `json:"lat,omitempty"`
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
	s *http.Server
	ac map[string]*jsonAircraftField
	mu sync.RWMutex
	projects map[string]*projectState
	messages int64
}

func NewAircraftMap(cfg config.MapSettings) *AircraftMap {
	m := &AircraftMap{
		ac: make(map[string]*jsonAircraftField),
		projects: make(map[string]*projectState),
	}
	port := uint16(8080)
	if cfg.Port != 0 {
		port = cfg.Port
	}
	r := mux.NewRouter()
	r.HandleFunc("/{project}/data/aircraft.json", m.aircraftJsonHandler)
	m.s = &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Address, port),
		Handler: handlers.CORS()(r),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return m
}
func (m *AircraftMap) registerProject(p *Project) error {
	fmt.Println("registering map project: "+p.Name)
	return nil
}
func (m *AircraftMap) deregisterProject(p *Project) error {
	fmt.Println("deregistering map project: "+p.Name)
	return nil
}
func (m *AircraftMap) projectNewAircraft(p *Project, s *Sighting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fmt.Printf("project %s - new aircraft on map (%s)\n", p.Name, s.State.Icao)
	if _, ok := m.ac[s.State.Icao]; !ok {
		m.ac[s.State.Icao] = &jsonAircraftField{
			referenceCount: 1,
			lastMsgTime: time.Now(),
			Hex: s.State.Icao,
			Flight: s.State.CallSign,
			Latitude: s.State.Latitude,
			Longitude: s.State.Longitude,
			Squawk: s.State.Squawk,
			MagneticHeading: s.State.Track,
			Track: s.State.Track,
			GroundSpeed: s.State.GroundSpeed,
			Messages: 1,
		}
	}

	// Associate record with project (init project if necessary)
	if state, ok := m.projects[p.Name]; !ok {
		m.projects[p.Name] = &projectState{
			name: p.Name,
			aircraft: []string{s.State.Icao},
		}
	} else {
		state.aircraft = append(state.aircraft, s.State.Icao)
	}
	m.messages++
	return nil
}
func (m *AircraftMap) projectUpdatedAircraft(p *Project, s *Sighting) error {
	m.mu.Lock()
	defer m.mu.Unlock()


	//fmt.Printf("project %s - updated aircraft on map (%s)\n", p.Name, s.State.Icao)
	acRecord := m.ac[s.State.Icao]

	locationUpdated := s.State.Latitude != acRecord.Latitude || s.State.Longitude != acRecord.Longitude

	acRecord.Messages++
	acRecord.lastMsgTime = time.Now()
	if locationUpdated {
		acRecord.lastPosTime = time.Now()
	}
	acRecord.Flight = s.State.CallSign
	acRecord.BarometricAltitude = s.State.Altitude
	acRecord.Latitude = s.State.Latitude
	acRecord.Longitude = s.State.Longitude
	acRecord.Squawk = s.State.Squawk
	acRecord.MagneticHeading = s.State.Track
	acRecord.Track = s.State.Track
	acRecord.GroundSpeed = s.State.GroundSpeed
	m.messages++
	return nil
}
func (m *AircraftMap) projectLostAircraft(p *Project, s *Sighting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fmt.Printf("project %s - lost aircraft on map (%s)\n", p.Name, s.State.Icao)

	projRecord := m.projects[p.Name]
	numAc := int64(len(projRecord.aircraft))
	var toDelete int64 = -1
	for i := int64(0); i < numAc && toDelete == -1; i++ {
		if projRecord.aircraft[i] == s.State.Icao {
			toDelete = i
		}
	}
	projRecord.aircraft = append(projRecord.aircraft[:toDelete], projRecord.aircraft[toDelete+1:]...)
	m.ac[s.State.Icao].referenceCount--
	if m.ac[s.State.Icao].referenceCount == 0 {
		delete(m.ac, s.State.Icao)
	}
	return nil
}
func (m *AircraftMap) aircraftJsonHandler(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	vars := mux.Vars(r)
	project := vars["project"]

	numAc := len(m.ac)
	l := make([]*jsonAircraftField, 0, numAc)
	for _, v := range m.ac {
		v.Seen = int64(time.Since(v.lastMsgTime).Seconds())
		v.SeenPos = time.Since(v.lastMsgTime).Seconds()
		l = append(l, v)
	}
	ac := jsonAircraft{
		Now: float64(time.Now().Unix()),
		Messages: m.messages,
		Aircraft: l,
	}
	data, err := json.Marshal(ac)
	if err != nil {
		panic(err)
	}
	_, err = w.Write(data)
	if err != nil {
		panic(err)
	}
}
func (m *AircraftMap) Serve() {
	go func() {
		m.s.ListenAndServe()
	}()
}