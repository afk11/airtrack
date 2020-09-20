package mictronics

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

type Aircraft struct {
	Registration string `json:"r"`
	TypeCode     string `json:"t"`
	F            string `json:"f"`
	Description  string `json:"d"`
}

type Operator struct {
	Name    string `json:"n"`
	Country string `json:"c"`
	R       string `json:"r"`
}
