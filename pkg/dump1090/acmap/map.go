package acmap

import (
	"encoding/json"
	"github.com/afk11/airtrack/pkg/tracker"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// NewDump1090Map returns a new Dump1090Map.
func NewDump1090Map(ma tracker.MapAccess) *Dump1090Map {
	return &Dump1090Map{m: ma}
}

// Dump1090Map. Implements tracker.MapService.
type Dump1090Map struct {
	m tracker.MapAccess
}

// MapService returns the name of this map service. See MapService.MapService.
func (d *Dump1090Map) MapService() string {
	return "dump1090"
}

// UpdateHistory generates a new history file the provided projNames.
// See MapService.UpdateHistory.
func (d *Dump1090Map) UpdateHistory(projNames []string) error {
	for i := range projNames {
		err := d.m.GetProjectAircraft(projNames[i], func(messageCount int64, fields []*tracker.JsonAircraft) error {
			ac := jsonAircraft{
				Now:      float64(time.Now().Unix()),
				Messages: messageCount,
				Aircraft: fields,
			}
			_, err := json.Marshal(ac)
			if err != nil {
				return err
			}
			return nil
		})
		if err == tracker.UnknownProject {
			// todo: should this bubble up?
		} else if err != nil {
			panic(err)
		}
	}
	return nil
}

// assetResponseHandler provides a HTTP handler for serving static assets
type assetResponseHandler string

// responseHandler implements a HTTP handler for the configured file name.
func (h assetResponseHandler) responseHandler(w http.ResponseWriter, r *http.Request) {
	dat, err := Asset(string(h))
	if err != nil {
		panic("asset: Asset(" + string(h) + "): " + err.Error())
	}
	if len(h) > 4 && h[len(h)-4:] == ".css" {
		w.Header().Set("Content-Type", "text/css")
	} else if len(h) > 3 && h[len(h)-3:] == ".js" {
		w.Header().Set("Content-Type", "text/javascript")
	} else if len(h) > 5 && h[len(h)-5:] == ".html" {
		w.Header().Set("Content-Type", "text/html")
	}
	_, err = w.Write(dat)
	if err != nil {
		log.Infof("error writing response: %s", err.Error())
	}
}

// RegisterRoutes registers the routes for dump1090 on r.
func (d *Dump1090Map) RegisterRoutes(r *mux.Router) error {
	r.HandleFunc("/{project}/data/aircraft.json", d.AircraftJsonHandler)
	r.HandleFunc("/{project}/data/receiver.json", d.ReceiverJsonHandler)
	assetNames := AssetNames()
	for i := range assetNames {
		_, err := Asset(assetNames[i])
		if err != nil {
			return errors.Wrap(err, "loading dump1090 asset: "+assetNames[i])
		}
		handler := assetResponseHandler(assetNames[i])
		r.HandleFunc("/{project}/"+assetNames[i], handler.responseHandler)
	}
	return nil
}

// ReceiverJsonHandler implements the HTTP handler for receiver.json
func (d *Dump1090Map) ReceiverJsonHandler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("{ \"version\" : \"v3.8.3\", \"refresh\" : 1000, \"history\" : 32 }"))
	if err != nil {
		w.WriteHeader(500)
		panic(err)
	}
}

// AircraftJsonHandler implements the HTTP handler for aircraft.json
func (d *Dump1090Map) AircraftJsonHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := d.m.GetProjectAircraft(vars["project"], func(messageCount int64, fields []*tracker.JsonAircraft) error {
		ac := jsonAircraft{
			Now:      float64(time.Now().Unix()),
			Messages: messageCount,
			Aircraft: fields,
		}
		data, err := json.Marshal(ac)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if err == tracker.UnknownProject {
		w.WriteHeader(404)
	} else if err != nil {
		w.WriteHeader(500)
		panic(err)
	}
}

// jsonAircraft defines the JSON structure for aircraft.json
type jsonAircraft struct {
	Now      float64                 `json:"now"`
	Messages int64                   `json:"messages"`
	Aircraft []*tracker.JsonAircraft `json:"aircraft"`
}
