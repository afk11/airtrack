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
	Feature           string
	EmailNotification string
	Project           struct {
		Name string
		// ShouldMap indicates whether the map should be built for this project
		ShouldMap          bool
		Site               *db.CollectionSite
		Session            *db.CollectionSession
		Filter             string
		Program            cel.Program
		Features           []Feature
		NotifyEmail        string
		EmailNotifications []EmailNotification

		ReopenSightings         bool
		ReopenSightingsInterval time.Duration
		OnGroundUpdateThreshold int64

		obsMu        sync.RWMutex
		Observations map[string]*ProjectObservation
	}
)

const (
	TrackTxTypes     Feature = "track_tx_types"
	TrackCallSigns   Feature = "track_callsigns"
	TrackSquawks     Feature = "track_squawks"
	TrackKmlLocation Feature = "track_kml"
	TrackTakeoff     Feature = "track_takeoff"
	GeocodeEndpoints Feature = "geocode_endpoints"

	MapProduced           EmailNotification = "map_produced"
	SpottedInFlight       EmailNotification = "spotted_in_flight"
	TakeoffStart          EmailNotification = "takeoff_start"
	TakeoffComplete       EmailNotification = "takeoff_complete"
	TakeoffUnknownAirport EmailNotification = "takeoff_unknown_airport"

	DefaultSightingReopenInterval = time.Minute * 5
)

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
	return "", errors.New("unknown feature")
}

func EmailNotificationFromString(n string) (EmailNotification, error) {
	switch n {
	case string(MapProduced):
		return MapProduced, nil
	case string(SpottedInFlight):
		return SpottedInFlight, nil
	case string(TakeoffStart):
		return TakeoffStart, nil
	case string(TakeoffComplete):
		return TakeoffComplete, nil
	}
	return "", errors.New("unknown notification")
}

func (p *Project) IsFeatureEnabled(f Feature) bool {
	for _, pf := range p.Features {
		if pf == f {
			return true
		}
	}
	return false
}

func (p *Project) IsEmailNotificationEnabled(n EmailNotification) bool {
	for _, ni := range p.EmailNotifications {
		if ni == n {
			return true
		}
	}
	return false
}

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
		p.ShouldMap = cfg.Map.Disabled == false
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
			return nil, errors.Wrapf(err, "unknown feature: %s", f)
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
				return nil, errors.Wrapf(err, "unknown email notification: %s", n)
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
					nil)))

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
