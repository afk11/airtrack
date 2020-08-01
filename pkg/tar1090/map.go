package tar1090

import (
	"encoding/json"
	"fmt"
	"github.com/afk11/airtrack/pkg/tracker"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type ProjectHistory struct {
	// nextIdx counts from zero to maxHistory. When nextIdx == maxHistory
	// it gets reset to zero
	nextIdx    int
	historyLen int
	history    [][]byte
}

func newProjectHistory(maxHistory int) *ProjectHistory {
	ph := &ProjectHistory{
		history: make([][]byte, maxHistory),
	}
	return ph
}
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

type History struct {
	maxHistory int
	history    map[string]*ProjectHistory
	mu         sync.RWMutex
}

func NewHistory(maxHistory int) *History {
	return &History{
		maxHistory: maxHistory,
		history:    make(map[string]*ProjectHistory),
	}
}
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
func (h *History) GetHistoryCount(project string) (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hist, ok := h.history[project]
	if !ok {
		return 0, errors.New("unknown project")
	}
	return hist.historyLen, nil
}
func (h *History) GetHistoryFile(project string, file int64) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	hist, ok := h.history[project]
	if !ok {
		return nil, errors.New("unknown project")
	}
	if file > int64(hist.historyLen) {
		return nil, errors.New("history file is out of range")
	}
	fmt.Printf("getHistoryFile historyLen=%d file=%d\n", hist.historyLen, file)
	return hist.history[file], nil
}

type HistoryUpdateScheduler struct {
	m tracker.MapAccess
	h *History
}

func (s *HistoryUpdateScheduler) UpdateHistory(projects []string) error {
	var data []byte
	for _, project := range projects {
		err := s.m.GetProjectAircraft(project, func(messageCount int64, fields []*tracker.JsonAircraft) error {
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
		err = s.h.SaveAircraftFile(project, data)
		if err != nil {
			return err
		}
	}
	return nil
}

type jsonAircraft struct {
	Now      float64                 `json:"now"`
	Messages int64                   `json:"messages"`
	Aircraft []*tracker.JsonAircraft `json:"aircraft"`
}

type assetResponseHandler string

func (h assetResponseHandler) responseHandler(w http.ResponseWriter, r *http.Request) {
	dat := MustAsset(string(h))
	if len(h) > 4 && h[len(h)-4:] == ".css" {
		w.Header().Set("Content-Type", "text/css")
	} else if len(h) > 3 && h[len(h)-3:] == ".js" {
		w.Header().Set("Content-Type", "text/javascript")
	} else if len(h) > 5 && h[len(h)-5:] == ".html" {
		w.Header().Set("Content-Type", "text/html")
	}
	_, err := w.Write(dat)
	if err != nil {
		log.Infof("error writing response: %s", err.Error())
	}
}

func NewTar1090Map(ma tracker.MapAccess, maxHistory int) *Map {
	return &Map{
		m: ma,
		h: NewHistory(maxHistory),
	}
}

type Map struct {
	m tracker.MapAccess
	h *History
}

func (t *Map) MapService() string {
	return "tar1090"
}
func (t *Map) UpdateScheduler() tracker.MapHistoryUpdateScheduler {
	return &HistoryUpdateScheduler{t.m, t.h}
}

func (t *Map) RegisterRoutes(r *mux.Router) error {
	r.HandleFunc("/{project}/data/aircraft.json", t.AircraftJsonHandler)
	r.HandleFunc("/{project}/data/history_{file}.json", t.HistoryJsonHandler)
	r.HandleFunc("/{project}/data/receiver.json", t.ReceiverJsonHandler)
	for _, file := range t.statics() {
		assetPath := "tar1090/html/" + file
		_, err := Asset(assetPath)
		if err != nil {
			return errors.Wrap(err, "loading dump1090 asset "+file)
		}
		r.HandleFunc("/{project}/"+file, assetResponseHandler(assetPath).responseHandler)
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
func (t *Map) statics() []string {
	return []string{
		"ol/ol-layerswitcher350.js",
		"ol/ol-layerswitcher350.css",
		"ol/v631/ol.js",
		"ol/v631/ol.js.map",
		"ol/v631/ol.css.map",
		"ol/v631/ol.css",
		"flags.js",
		"dbloader.js",
		"planeObject.js",
		"registrations.js",
		"early.js",
		"defaults.js",
		"markers.js",
		"geomag2020.js",
		"images/tar1090-favicon.png",
		"images/table-icon@3x.png",
		"images/icon-information@2x.png",
		"images/box-checked@3x.png",
		"images/zoom-in@3x.png",
		"images/map-icon@2x.png",
		"images/column-adjust@2x.png",
		"images/hide_sidebar.png",
		"images/zoom-out@3x.png",
		"images/box-checked.png",
		"images/alt_legend_meters.svg",
		"images/settings-icon@3x.png",
		"images/settings-icon.png",
		"images/box-empty@3x.png",
		"images/map-icon.png",
		"images/table-icon.png",
		"images/close-settings.png",
		"images/alt_legend_feet.svg",
		"images/show_sidebar.png",
		"images/close-settings@2x.png",
		"images/zoom-in.png",
		"images/toggle-width@2x.png",
		"images/toggle-height@2x.png",
		"images/close-settings@3x.png",
		"images/zoom-out.png",
		"images/box-empty.png",
		"images/zoom-out@2x.png",
		"images/table-icon@2x.png",
		"images/map-icon@3x.png",
		"images/box-checked@2x.png",
		"images/column-adjust@3x.png",
		"images/column-adjust.png",
		"images/settings-icon@2x.png",
		"images/box-empty@2x.png",
		"images/zoom-in@2x.png",
		"colors.css",
		"formatter.js",
		"layers.js",
		"jquery/README",
		"jquery/jquery-ui-1.12.1/jquery-ui.structure.css",
		"jquery/jquery-ui-1.12.1/package.json",
		"jquery/jquery-ui-1.12.1/images/ui-icons_555555_256x240.png",
		"jquery/jquery-ui-1.12.1/images/ui-icons_444444_256x240.png",
		"jquery/jquery-ui-1.12.1/images/ui-icons_ffffff_256x240.png",
		"jquery/jquery-ui-1.12.1/images/ui-icons_777620_256x240.png",
		"jquery/jquery-ui-1.12.1/images/ui-icons_777777_256x240.png",
		"jquery/jquery-ui-1.12.1/images/ui-icons_cc0000_256x240.png",
		"jquery/jquery-ui-1.12.1/jquery-ui.structure.min.css",
		"jquery/jquery-ui-1.12.1/AUTHORS.txt",
		"jquery/jquery-ui-1.12.1/jquery-ui.min.js",
		"jquery/jquery-ui-1.12.1/jquery-ui.css",
		"jquery/jquery-ui-1.12.1/LICENSE.txt",
		"jquery/jquery-ui-1.12.1/index.html",
		"jquery/jquery-ui-1.12.1/jquery-ui.theme.css",
		"jquery/jquery-ui-1.12.1/jquery-ui.js",
		"jquery/jquery-ui-1.12.1/jquery-ui.theme.min.css",
		"jquery/jquery-ui-1.12.1/jquery-ui.min.css",
		"jquery/jquery-ui-1.12.1/external/jquery/jquery.js",
		"jquery/jquery.ui.touch-punch.min.js",
		"jquery/jquery-3.5.1.min.js",
		"style.css",
		"flags-tiny/Pitcairn_Islands.png",
		"flags-tiny/Slovenia.png",
		"flags-tiny/Guatemala.png",
		"flags-tiny/Uruguay.png",
		"flags-tiny/Niger.png",
		"flags-tiny/Seychelles.png",
		"flags-tiny/Chad.png",
		"flags-tiny/Ukraine.png",
		"flags-tiny/Philippines.png",
		"flags-tiny/Anguilla.png",
		"flags-tiny/United_States_of_America.png",
		"flags-tiny/Romania.png",
		"flags-tiny/American_Samoa.png",
		"flags-tiny/US_Virgin_Islands.png",
		"flags-tiny/Montserrat.png",
		"flags-tiny/Honduras.png",
		"flags-tiny/Mexico.png",
		"flags-tiny/Belize.png",
		"flags-tiny/Cape_Verde.png",
		"flags-tiny/Brunei.png",
		"flags-tiny/Iran.png",
		"flags-tiny/Macedonia.png",
		"flags-tiny/Japan.png",
		"flags-tiny/Burundi.png",
		"flags-tiny/Central_African_Republic.png",
		"flags-tiny/Guyana.png",
		"flags-tiny/Gabon.png",
		"flags-tiny/Portugal.png",
		"flags-tiny/Mongolia.png",
		"flags-tiny/South_Georgia.png",
		"flags-tiny/Uzbekistan.png",
		"flags-tiny/Saudi_Arabia.png",
		"flags-tiny/Togo.png",
		"flags-tiny/Sri_Lanka.png",
		"flags-tiny/Ethiopia.png",
		"flags-tiny/Nigeria.png",
		"flags-tiny/Egypt.png",
		"flags-tiny/Turkey.png",
		"flags-tiny/Chile.png",
		"flags-tiny/Bhutan.png",
		"flags-tiny/Papua_New_Guinea.png",
		"flags-tiny/Dominican_Republic.png",
		"flags-tiny/UAE.png",
		"flags-tiny/Slovakia.png",
		"flags-tiny/Grenada.png",
		"flags-tiny/Vietnam.png",
		"flags-tiny/Fiji.png",
		"flags-tiny/Bermuda.png",
		"flags-tiny/Cote_d_Ivoire.png",
		"flags-tiny/Lebanon.png",
		"flags-tiny/Luxembourg.png",
		"flags-tiny/Serbia.png",
		"flags-tiny/Guinea.png",
		"flags-tiny/South_Africa.png",
		"flags-tiny/Antigua_and_Barbuda.png",
		"flags-tiny/Senegal.png",
		"flags-tiny/Peru.png",
		"flags-tiny/Saint_Pierre.png",
		"flags-tiny/Iceland.png",
		"flags-tiny/Somalia.png",
		"flags-tiny/Switzerland.png",
		"flags-tiny/Gibraltar.png",
		"flags-tiny/Pakistan.png",
		"flags-tiny/Cayman_Islands.png",
		"flags-tiny/Cuba.png",
		"flags-tiny/Eritrea.png",
		"flags-tiny/Macao.png",
		"flags-tiny/Palau.png",
		"flags-tiny/Nauru.png",
		"flags-tiny/blank.png",
		"flags-tiny/Greenland.png",
		"flags-tiny/Cameroon.png",
		"flags-tiny/Iraq.png",
		"flags-tiny/Sweden.png",
		"flags-tiny/Cyprus.png",
		"flags-tiny/Laos.png",
		"flags-tiny/Russian_Federation.png",
		"flags-tiny/Malta.png",
		"flags-tiny/China.png",
		"flags-tiny/Equatorial_Guinea.png",
		"flags-tiny/Saint_Vicent_and_the_Grenadines.png",
		"flags-tiny/Belarus.png",
		"flags-tiny/United_Kingdom.png",
		"flags-tiny/Bangladesh.png",
		"flags-tiny/Georgia.png",
		"flags-tiny/Ireland.png",
		"flags-tiny/Singapore.png",
		"flags-tiny/Indonesia.png",
		"flags-tiny/Hong_Kong.png",
		"flags-tiny/Argentina.png",
		"flags-tiny/Bulgaria.png",
		"flags-tiny/Ecuador.png",
		"flags-tiny/Montenegro.png",
		"flags-tiny/Madagascar.png",
		"flags-tiny/Rwanda.png",
		"flags-tiny/Guam.png",
		"flags-tiny/Germany.png",
		"flags-tiny/Tunisia.png",
		"flags-tiny/Myanmar.png",
		"flags-tiny/Yemen.png",
		"flags-tiny/Uganda.png",
		"flags-tiny/Greece.png",
		"flags-tiny/Samoa.png",
		"flags-tiny/Wallis_and_Futuna.png",
		"flags-tiny/Brazil.png",
		"flags-tiny/Algeria.png",
		"flags-tiny/Norway.png",
		"flags-tiny/Sao_Tome_and_Principe.png",
		"flags-tiny/New_Zealand.png",
		"flags-tiny/Moldova.png",
		"flags-tiny/Libya.png",
		"flags-tiny/Lithuania.png",
		"flags-tiny/Timor-Leste.png",
		"flags-tiny/Australia.png",
		"flags-tiny/Sudan.png",
		"flags-tiny/Jordan.png",
		"flags-tiny/Monaco.png",
		"flags-tiny/Liechtenstein.png",
		"flags-tiny/Dominica.png",
		"flags-tiny/Saint_Kitts_and_Nevis.png",
		"flags-tiny/Turks_and_Caicos_Islands.png",
		"flags-tiny/Tuvalu.png",
		"flags-tiny/Armenia.png",
		"flags-tiny/Panama.png",
		"flags-tiny/Burkina_Faso.png",
		"flags-tiny/Bahamas.png",
		"flags-tiny/Italy.png",
		"flags-tiny/Trinidad_and_Tobago.png",
		"flags-tiny/Maldives.png",
		"flags-tiny/Marshall_Islands.png",
		"flags-tiny/Costa_Rica.png",
		"flags-tiny/Zimbabwe.png",
		"flags-tiny/Bahrain.png",
		"flags-tiny/San_Marino.png",
		"flags-tiny/French_Polynesia.png",
		"flags-tiny/Yugoslavia.png",
		"flags-tiny/Nicaragua.png",
		"flags-tiny/Suriname.png",
		"flags-tiny/Syria.png",
		"flags-tiny/Jamaica.png",
		"flags-tiny/Bosnia.png",
		"flags-tiny/Benin.png",
		"flags-tiny/Martinique.png",
		"flags-tiny/Malaysia.png",
		"flags-tiny/Kazakhstan.png",
		"flags-tiny/Cambodia.png",
		"flags-tiny/Estonia.png",
		"flags-tiny/North_Korea.png",
		"flags-tiny/France.png",
		"flags-tiny/Qatar.png",
		"flags-tiny/Zambia.png",
		"flags-tiny/Canada.png",
		"flags-tiny/Micronesia.png",
		"flags-tiny/Czech_Republic.png",
		"flags-tiny/Sierra_Leone.png",
		"flags-tiny/Niue.png",
		"flags-tiny/Paraguay.png",
		"flags-tiny/Malawi.png",
		"flags-tiny/Hungary.png",
		"flags-tiny/Afghanistan.png",
		"flags-tiny/Republic_of_the_Congo.png",
		"flags-tiny/Soviet_Union.png",
		"flags-tiny/Belgium.png",
		"flags-tiny/Kenya.png",
		"flags-tiny/El_Salvador.png",
		"flags-tiny/Colombia.png",
		"flags-tiny/British_Virgin_Islands.png",
		"flags-tiny/Namibia.png",
		"flags-tiny/Puerto_Rico.png",
		"flags-tiny/Haiti.png",
		"flags-tiny/Mozambique.png",
		"flags-tiny/Venezuela.png",
		"flags-tiny/Finland.png",
		"flags-tiny/Angola.png",
		"flags-tiny/Bolivia.png",
		"flags-tiny/Botswana.png",
		"flags-tiny/Israel.png",
		"flags-tiny/Kyrgyzstan.png",
		"flags-tiny/Tibet.png",
		"flags-tiny/Norfolk_Island.png",
		"flags-tiny/South_Korea.png",
		"flags-tiny/Spain.png",
		"flags-tiny/Taiwan.png",
		"flags-tiny/Tajikistan.png",
		"flags-tiny/Comoros.png",
		"flags-tiny/Nepal.png",
		"flags-tiny/Swaziland.png",
		"flags-tiny/Croatia.png",
		"flags-tiny/Christmas_Island.png",
		"flags-tiny/Austria.png",
		"flags-tiny/Netherlands.png",
		"flags-tiny/Aruba.png",
		"flags-tiny/Djibouti.png",
		"flags-tiny/Guinea_Bissau.png",
		"flags-tiny/Cyprus_Northern.png",
		"flags-tiny/Poland.png",
		"flags-tiny/Lesotho.png",
		"flags-tiny/Andorra.png",
		"flags-tiny/Faroe_Islands.png",
		"flags-tiny/India.png",
		"flags-tiny/Oman.png",
		"flags-tiny/Democratic_Republic_of_the_Congo.png",
		"flags-tiny/Albania.png",
		"flags-tiny/Denmark.png",
		"flags-tiny/README.txt",
		"flags-tiny/Azerbaijan.png",
		"flags-tiny/Turkmenistan.png",
		"flags-tiny/Tanzania.png",
		"flags-tiny/Tonga.png",
		"flags-tiny/Netherlands_Antilles.png",
		"flags-tiny/Mali.png",
		"flags-tiny/Saint_Lucia.png",
		"flags-tiny/Vatican_City.png",
		"flags-tiny/Liberia.png",
		"flags-tiny/Ghana.png",
		"flags-tiny/Thailand.png",
		"flags-tiny/Kiribati.png",
		"flags-tiny/Mauritania.png",
		"flags-tiny/Latvia.png",
		"flags-tiny/Gambia.png",
		"flags-tiny/Vanuatu.png",
		"flags-tiny/Kuwait.png",
		"flags-tiny/Cook_Islands.png",
		"flags-tiny/Mauritius.png",
		"flags-tiny/Barbados.png",
		"flags-tiny/Morocco.png",
		"flags-tiny/Falkland_Islands.png",
		"flags-tiny/Soloman_Islands.png",
		"index.html",
		"geojson/US_ARTCC_boundaries.geojson",
		"geojson/UK_Mil_AWACS_Orbits.geojson",
		"geojson/UK_Mil_RC.geojson",
		"geojson/L3Harris/USAFA_Training_Areas.geojson",
		"geojson/L3Harris/L3Harris_VNAV.geojson",
		"geojson/L3Harris/L3Harris_Training_Areas.geojson",
		"geojson/US_A2A_refueling.geojson",
		"geojson/UK_Mil_AAR_Zones.geojson",
		"config.js",
		"script.js",
	}
}
