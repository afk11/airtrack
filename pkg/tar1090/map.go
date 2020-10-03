package tar1090

import (
	"encoding/json"
	"fmt"
	"github.com/afk11/airtrack/pkg/readsb/aircraft_db"
	"github.com/afk11/airtrack/pkg/tracker"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ProjectHistory contains an in-memory store of history
// files for a particular project.
type ProjectHistory struct {
	// nextIdx counts from zero to maxHistory. When nextIdx == maxHistory
	// it gets reset to zero so old files are overwritten.
	nextIdx int
	// historyLen increases until it reaches maxHistory
	historyLen int
	// history - the list of JSON files (contains at most maxHistory)
	history [][]byte
}

// newProjectHistory initializes the history store for a project
// containing at most maxHistory files.
func newProjectHistory(maxHistory int) *ProjectHistory {
	ph := &ProjectHistory{
		history: make([][]byte, maxHistory),
	}
	return ph
}

// AddNextFile takes a new history data and adds it to the store.
func (ph *ProjectHistory) AddNextFile(maxHistory int, data []byte) error {
	if ph.nextIdx == maxHistory {
		ph.nextIdx = 0
	}
	if ph.historyLen < maxHistory-1 {
		ph.historyLen++
	}
	ph.history[ph.nextIdx] = data
	ph.nextIdx++
	return nil
}

// History contains ProjectHistory structures for all projects
type History struct {
	// maxHistory - max history files to store for each project
	maxHistory int
	// history - map of project name to ProjectHistory
	history map[string]*ProjectHistory
	// mu - read/write access over history map
	mu sync.RWMutex
}

// NewHistory returns a History initialized to store maxHistory
// files per project.
func NewHistory(maxHistory int) *History {
	return &History{
		maxHistory: maxHistory,
		history:    make(map[string]*ProjectHistory),
	}
}

// SaveAircraftFile takes a new history data and adds it to the
// projects history.
func (h *History) SaveAircraftFile(project string, data []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	hist, ok := h.history[project]
	if !ok {
		hist = newProjectHistory(h.maxHistory)
		h.history[project] = hist
	}
	return hist.AddNextFile(h.maxHistory, data)
}

// GetHistoryCount returns the number of history files the project currently
// has, or returns an error if the project is unknown.
func (h *History) GetHistoryCount(project string) (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hist, ok := h.history[project]
	if !ok {
		return 0, errors.New("unknown project")
	}
	return hist.historyLen, nil
}

// GetHistoryFile returns n'th history file for project, or returns
// an error if the project is unknown or the history file is invalid.
func (h *History) GetHistoryFile(project string, n int64) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	hist, ok := h.history[project]
	if !ok {
		return nil, errors.New("unknown project")
	}
	if n > int64(hist.historyLen) {
		return nil, errors.New("history file is out of range")
	}

	return hist.history[n], nil
}

// jsonAircraft defines the JSON structure returned for aircraft.json
type jsonAircraft struct {
	Now      float64                 `json:"now"`
	Messages int64                   `json:"messages"`
	Aircraft []*tracker.JsonAircraft `json:"aircraft"`
}

// assetResponseHandler provides a HTTP handler to serve a static asset
type assetResponseHandler struct {
	name   string
	loader func(string) ([]byte, error)
}

// responseHandler responds with the asset
func (h assetResponseHandler) responseHandler(w http.ResponseWriter, r *http.Request) {
	dat, err := h.loader(h.name)
	if err != nil {
		panic("asset: Asset(" + h.name + "): " + err.Error())
	}

	if len(h.name) > 4 && h.name[len(h.name)-4:] == ".css" {
		w.Header().Set("Content-Type", "text/css")
	} else if len(h.name) > 3 && h.name[len(h.name)-3:] == ".js" {
		w.Header().Set("Content-Type", "text/javascript")
	} else if len(h.name) > 5 && h.name[len(h.name)-5:] == ".json" {
		w.Header().Set("Content-Type", "application/json")
	} else if len(h.name) > 5 && h.name[len(h.name)-5:] == ".html" {
		w.Header().Set("Content-Type", "text/html")
	}
	_, err = w.Write(dat)
	if err != nil {
		log.Infof("error writing response: %s", err.Error())
	}
}

// NewTar1090Map returns a new Map.
func NewTar1090Map(ma tracker.MapAccess, maxHistory int) *Map {
	return &Map{
		m: ma,
		h: NewHistory(maxHistory),
	}
}

// Map - provides the tar1090 map. Implements MapService.
type Map struct {
	m tracker.MapAccess
	h *History
}

// MapService returns the name of the map service. See MapService.MapService.
func (t *Map) MapService() string {
	return "tar1090"
}

// UpdateHistory generates a new history file for each project in projNames.
// See MapService.UpdateHistory.
func (t *Map) UpdateHistory(projNames []string) error {
	var data []byte
	for i := range projNames {
		err := t.m.GetProjectAircraft(projNames[i], func(messageCount int64, fields []*tracker.JsonAircraft) error {
			ac := jsonAircraft{
				Now:      float64(time.Now().Unix()),
				Messages: messageCount,
				Aircraft: fields,
			}
			var err error
			data, err = json.Marshal(ac)
			if err != nil {
				return err
			}
			return nil
		})
		if err == tracker.UnknownProject {
			panic(err)
		} else if err != nil {
			panic(err)
		}
		err = t.h.SaveAircraftFile(projNames[i], data)
		if err != nil {
			return err
		}
	}
	return nil
}

// RegisterRoutes registers handler functions for tar1090 routes on r.
func (t *Map) RegisterRoutes(r *mux.Router) error {
	r.HandleFunc("/{project}/data/aircraft.json", t.AircraftJsonHandler)
	r.HandleFunc("/{project}/data/history_{file}.json", t.HistoryJsonHandler)
	r.HandleFunc("/{project}/data/receiver.json", t.ReceiverJsonHandler)
	r.HandleFunc("/{project}/db2/icao_aircraft_types.js", assetResponseHandler{"types.json", aircraft_db.Asset}.responseHandler)
	r.HandleFunc("/{project}/db2/files.js", assetResponseHandler{"files.json", aircraft_db.Asset}.responseHandler)

	dat, err := aircraft_db.Asset("files.json")
	if err != nil {
		return errors.Wrapf(err, "reading aircraft_db asset files.json")
	}
	var shards []string
	err = json.Unmarshal(dat, &shards)
	if err != nil {
		return errors.Wrapf(err, "decoding aircraft_db asset files.json")
	}

	for _, shard := range shards {
		r.HandleFunc("/{project}/db2/"+shard+".js", assetResponseHandler{shard + ".json", aircraft_db.Asset}.responseHandler)
	}

	assetNames := AssetNames()
	for i := range assetNames {
		_, err := Asset(assetNames[i])
		if err != nil {
			return errors.Wrapf(err, "loading dump1090 asset: "+assetNames[i])
		}
		r.HandleFunc("/{project}/"+assetNames[i], assetResponseHandler{assetNames[i], Asset}.responseHandler)
	}
	return nil
}
func (t *Map) ReceiverJsonHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectName := vars["project"]
	count, err := t.h.GetHistoryCount(projectName)
	if err != nil {
		fmt.Println("UNKNOWN PROJECT - receiver handler")
		w.WriteHeader(404)
		return
	}
	data := []byte("{ \"version\" : \"v3.8.3\", \"refresh\" : 1000, \"history\" : " + strconv.Itoa(count) + " }")
	_, err = w.Write(data)
	if err != nil {
		w.WriteHeader(500)
		panic(err)
	}
}
func (t *Map) AircraftJsonHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := t.m.GetProjectAircraft(vars["project"], func(messageCount int64, fields []*tracker.JsonAircraft) error {
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
func (t *Map) HistoryJsonHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectName := vars["project"]
	i, err := strconv.ParseInt(vars["file"], 10, 32)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	history, err := t.h.GetHistoryFile(projectName, i)
	if err == tracker.UnknownProject {
		w.WriteHeader(404)
		return
	} else if err != nil {
		w.WriteHeader(500)
		panic(err)
	}
	_, err = w.Write(history)
	if err != nil {
		log.Infof("error writing request")
	}
}
