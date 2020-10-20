package aircraftdb

import (
	"github.com/afk11/airtrack/pkg/pb"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
)

// OperatorCountryFixupMap - map of mictronics country names to
// names in our list
var OperatorCountryFixupMap = map[string]string{
	"Netherlands":             "Netherlands, Kingdom of",
	"Tanzania":                "United Republic of Tanzania",
	"CÃ´te d'Ivoire":          "Côte d Ivoire",
	"Macedonia":               "North Macedonia",
	"Brunei":                  "Brunei Darussalam",
	"Syria":                   "Syrian Arab Republic",
	"Iran":                    "Iran, Islamic Republic",
	"Russia":                  "Russian Federation",
	"Moldova":                 "Republic of Moldova",
	"Laos":                    "Lao People's Democratic Republic",
	"SÃ£o TomÃ© and Principe": "Sao Tome",
	"South Korea":             "Republic of Korea",
	"North Korea":             "Democratic People's Republic of Korea",
	"Congo (Brazzaville)":     "Democratic Republic of the Congo",
	"PerÃº":                   "Peru",
}

// Operator contains information about an airline operator
type Operator struct {
	Name    string `json:"n"`
	Country string `json:"c"`
	R       string `json:"r"`
}

// Db provides access to the readsb aircraft database
type Db struct {
	initialized bool
	aircraft    map[string]pb.AircraftInfo
	operators   map[string]pb.Operator
}

// New returns an initialized (but empty) Db
func New() *Db {
	return &Db{
		aircraft:  make(map[string]pb.AircraftInfo),
		operators: make(map[string]pb.Operator),
	}
}

// GetAircraft searches for aircraft information using icao. The second return
// argument indicates whether the search was successful. If its false
// the aircraft info will be nil.
func (d *Db) GetAircraft(icao string) (*pb.AircraftInfo, bool) {
	ac, ok := d.aircraft[icao]
	if !ok {
		return nil, false
	}
	return &ac, true
}

// GetOperator searches for operator information using code. The second return
// argument indicates whether the search was successful. If its false
// the operator info will be nil.
func (d *Db) GetOperator(code string) (*pb.Operator, bool) {
	op, ok := d.operators[code]
	if !ok {
		return nil, false
	}
	return &op, true
}

//easyjson:json
type aircraftFile map[string][4]string

//easyjson:json
type operatorFile map[string]Operator

//easyjson:json
type shardFile []string

// LoadAssets loads the readsb database from internal files into Db.
func LoadAssets(db *Db, Asset func(string) ([]byte, error)) error {
	if db.initialized {
		return nil
	}
	filesJSON, err := Asset("files.json")
	if err != nil {
		return errors.Wrapf(err, "reading files.json asset")
	}
	var files shardFile
	err = easyjson.Unmarshal(filesJSON, &files)
	if err != nil {
		return errors.Wrapf(err, "unmarshal files.json")
	}

	for _, filePrefix := range files {
		d, err := Asset(filePrefix + ".json")
		if err != nil {
			return errors.Wrapf(err, "reading %s.json asset", filePrefix)
		}
		var tmp aircraftFile
		err = easyjson.Unmarshal(d, &tmp)
		if err != nil {
			return errors.Wrapf(err, "unmarshal %s.json", filePrefix)
		}
		for icaoSuffix := range tmp {
			if icaoSuffix == "children" {
				continue
			}
			db.aircraft[filePrefix+icaoSuffix] = pb.AircraftInfo{
				Registration: tmp[icaoSuffix][0],
				TypeCode:     tmp[icaoSuffix][1],
				F:            tmp[icaoSuffix][2],
				Description:  tmp[icaoSuffix][3],
			}
		}
	}

	operatorsJSON, err := Asset("operators.json")
	if err != nil {
		return errors.Wrapf(err, "reading operators.json asset")
	}
	var operators operatorFile
	err = easyjson.Unmarshal(operatorsJSON, &operators)
	if err != nil {
		return errors.Wrapf(err, "unmarshal operators.json")
	}
	for code := range operators {
		db.operators[code] = pb.Operator{
			Name:        operators[code].Name,
			CountryName: operators[code].Country,
			R:           operators[code].R,
		}
	}
	db.initialized = true
	return nil
}
