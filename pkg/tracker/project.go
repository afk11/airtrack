package tracker

import (
	"github.com/afk11/airtrack/pkg/config"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type (
	// Feature represents a tracking capability to use in a project
	Feature string
	// EmailNotification represents an email topic to which the project is subscribed.
	EmailNotification string

	// Project represents an active tracking project
	Project struct {
		// Name of the project
		Name string
		// ShouldMap indicates whether the map should be built for this project
		ShouldMap bool
		// Project - the db record for this project
		Project *db.Project
		// Session - the db record for the project session
		Session *db.Session
		// Filter - a CEL expression for filtering aircraft (can be empty)
		Filter string
		// Program - a parsed CEL expression to evaluate later
		Program cel.Program
		// Features is the list of tracking features enabled in this project
		Features []Feature
		// NotifyEmail - the destination for email notifications
		NotifyEmail string
		// EmailNotifications - list of topics the project is subscribed to
		EmailNotifications []EmailNotification

		// ReopenSightings - whether to reopen a sighting if it was seen within `ReopenSightingsInterval`
		ReopenSightings bool
		// ReopenSightingsInterval is a duration within which a previously closed sighting can
		// be reopened
		ReopenSightingsInterval time.Duration
		// OnGroundUpdateThreshold - how many consecutive messages we receive with a new on_ground
		// flag before we accept it
		OnGroundUpdateThreshold int64

		// Observations is a map of aircraft ICAO to it's state
		Observations map[string]*ProjectObservation

		obsMu sync.RWMutex
	}
)

const (
	// TrackTxTypes - track ADSB message types (only for BEAST messages)
	TrackTxTypes Feature = "track_tx_types"
	// TrackCallsigns - track current callsign and maintain history
	TrackCallSigns Feature = "track_callsigns"
	// TrackSquawks - track current squawk and maintain history
	TrackSquawks Feature = "track_squawks"
	// TrackKmlLocation - track location history for aircraft
	TrackKmlLocation Feature = "track_kml"
	// TrackTakeoff - (logs only) log when a takeoff begins/ends
	TrackTakeoff Feature = "track_takeoff"
	// GeocodeEndpoints - (logs only) geolocate the source + destination airport
	GeocodeEndpoints Feature = "geocode_endpoints"

	// MapProduced - the notification about a new map
	MapProduced EmailNotification = "map_produced"
	// SpottedInFlight - the notification about an aircraft spotted in air
	SpottedInFlight EmailNotification = "spotted_in_flight"
	// TakeoffFromAirport - the notification about an aircraft that just lifted off
	TakeoffFromAirport EmailNotification = "takeoff_from_airport"
	// TakeoffUnknownAirport - the notification about an aircraft that just lifted off from an unknown airport
	TakeoffUnknownAirport EmailNotification = "takeoff_unknown_airport"
	// TakeoffComplete - the notification about an aircraft that levels off after takeoff
	TakeoffComplete EmailNotification = "takeoff_complete"

	// DefaultSightingReopenInterval - default interval for sighting reopen behavior
	DefaultSightingReopenInterval = time.Minute * 5
)

// FeatureFromString parses the Feature type from the provided string
func FeatureFromString(f string) (Feature, error) {
	switch f {
	case string(TrackTxTypes):
		return TrackTxTypes, nil
	case string(TrackCallSigns):
		return TrackCallSigns, nil
	case string(TrackSquawks):
		return TrackSquawks, nil
	case string(TrackKmlLocation):
		return TrackKmlLocation, nil
	case string(TrackTakeoff):
		return TrackTakeoff, nil
	case string(GeocodeEndpoints):
		return GeocodeEndpoints, nil
	}
	return "", errors.Errorf("unknown feature: %s", f)
}

// EmailNotificationFromString parses the EmailNotification type from the provided string
func EmailNotificationFromString(n string) (EmailNotification, error) {
	switch n {
	case string(MapProduced):
		return MapProduced, nil
	case string(SpottedInFlight):
		return SpottedInFlight, nil
	case string(TakeoffFromAirport):
		return TakeoffFromAirport, nil
	case string(TakeoffComplete):
		return TakeoffComplete, nil
	case string(TakeoffUnknownAirport):
		return TakeoffUnknownAirport, nil
	}
	return "", errors.Errorf("unknown email notification: %s", n)
}

// IsFeatureEnabled returns whether the project has Feature f enabled
func (p *Project) IsFeatureEnabled(f Feature) bool {
	for _, pf := range p.Features {
		if pf == f {
			return true
		}
	}
	return false
}

// IsEmailNotificationEnabled returns whether the project has EmailNotification n enabled
func (p *Project) IsEmailNotificationEnabled(n EmailNotification) bool {
	for _, ni := range p.EmailNotifications {
		if ni == n {
			return true
		}
	}
	return false
}

// InitProject initializes a project from its configuration or an error upon failure.
func InitProject(cfg config.Project) (*Project, error) {
	if cfg.Disabled {
		return nil, errors.New("cannot init disabled project")
	}
	p := Project{
		Name:                    cfg.Name,
		Filter:                  cfg.Filter,
		Features:                make([]Feature, 0, len(cfg.Features)),
		ReopenSightings:         cfg.ReopenSightings,
		ReopenSightingsInterval: DefaultSightingReopenInterval,
		OnGroundUpdateThreshold: DefaultOnGroundUpdateThreshold,
		ShouldMap:               true,
		Observations:            make(map[string]*ProjectObservation),
	}
	if cfg.Map != nil {
		p.ShouldMap = !cfg.Map.Disabled
	}
	if p.ReopenSightings {
		p.ReopenSightingsInterval = time.Duration(cfg.ReopenSightingsInterval) * time.Second
	}
	if cfg.OnGroundUpdateThreshold != nil {
		p.OnGroundUpdateThreshold = *cfg.OnGroundUpdateThreshold
	}
	for _, f := range cfg.Features {
		feature, err := FeatureFromString(f)
		if err != nil {
			return nil, err
		}
		p.Features = append(p.Features, feature)
	}

	if cfg.Notifications != nil {
		if cfg.Notifications.Email == "" {
			return nil, errors.Errorf("notifications missing value for email")
		}
		p.NotifyEmail = cfg.Notifications.Email
		for _, n := range cfg.Notifications.Enabled {
			notification, err := EmailNotificationFromString(n)
			if err != nil {
				return nil, err
			}
			p.EmailNotifications = append(p.EmailNotifications, notification)
		}
	}

	if p.Filter != "" {
		env, err := cel.NewEnv(
			cel.Types(&pb.Source{}, &pb.Message{}, &pb.State{}),
			cel.Declarations(
				decls.NewIdent("msg",
					decls.NewObjectType("airtrack.Message"),
					nil),
				decls.NewIdent("state",
					decls.NewObjectType("airtrack.State"),
					nil),
				decls.NewVar("AdsbExchangeSource", decls.Int),
				decls.NewVar("BeastSource", decls.Int),
			))

		if err != nil {
			return nil, err
		}
		parsed, issues := env.Parse(cfg.Filter)
		if issues != nil && issues.Err() != nil {
			return nil, errors.Wrap(issues.Err(), "failed to parse filter expression")
		}
		checked, issues := env.Check(parsed)
		if issues != nil && issues.Err() != nil {
			return nil, errors.Wrap(issues.Err(), "type errors in filter expression")
		}
		prg, err := env.Program(checked)
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize filter")
		}
		p.Program = prg
	}
	return &p, nil
}
