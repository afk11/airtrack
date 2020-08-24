package airtrackqa

//
//import (
//	"bytes"
//	"encoding/json"
//	"fmt"
//	asset "github.com/afk11/airtrack/pkg/assets"
//	"github.com/afk11/airtrack/pkg/iso3166"
//	"github.com/afk11/airtrack/pkg/mictronics"
//)
//
//type MictronicsOperatorCountryQA struct {
//}
//
//func (cmd *MictronicsOperatorCountryQA) Run(ctx *Context) error {
//	operatorMap := make(map[string]mictronics.Operator)
//	fileContents, err := asset.Asset("assets/mictronics_db/operators.json")
//	if err != nil {
//		return err
//	}
//	err = json.Unmarshal(fileContents, &operatorMap)
//	if err != nil {
//		return err
//	}
//
//	fixup := map[string]string{
//		"Netherlands":             "Netherlands, Kingdom of",
//		"Tanzania":                "United Republic of Tanzania",
//		"CÃ´te d'Ivoire":          "Côte d Ivoire",
//		"Macedonia":               "North Macedonia",
//		"Brunei":                  "Brunei Darussalam",
//		"Syria":                   "Syrian Arab Republic",
//		"Iran":                    "Iran, Islamic Republic",
//		"Russia":                  "Russian Federation",
//		"Moldova":                 "Republic of Moldova",
//		"Laos":                    "Lao People's Democratic Republic",
//		"SÃ£o TomÃ© and Principe": "Sao Tome",
//		"South Korea":             "Republic of Korea",
//		"North Korea":             "Democratic People's Republic of Korea",
//		"Congo (Brazzaville)":     "Democratic Republic of the Congo",
//		"PerÃº":                   "Peru",
//	}
//	countryCodesData, err := asset.Asset("assets/iso3166_country_codes.txt")
//	if err != nil {
//		panic(err)
//	}
//	countryCodeRows, err := iso3166.ParseColumnFormat(bytes.NewBuffer(countryCodesData))
//	if err != nil {
//		panic(err)
//	}
//	countryMap := make(map[string]struct{})
//	for i := 0; i < len(countryCodeRows); i++ {
//		countryMap[countryCodeRows[i][2]] = struct{}{}
//	}
//
//	found := 0
//	notFound := 0
//	unknownCountries := make(map[string]struct{})
//	for operatorCode, operator := range operatorMap {
//		if replace, ok := fixup[operator.Country]; ok {
//			operator.Country = replace
//		}
//
//		_, ok := countryMap[operator.Country]
//		if !ok {
//			notFound++
//			fmt.Printf("operator code: %s has country not found in our db: %s\n", operatorCode, operator.Country)
//			if _, ok = unknownCountries[operator.Country]; !ok {
//				unknownCountries[operator.Country] = struct{}{}
//			}
//		} else {
//			found++
//		}
//	}
//	fmt.Printf("records with a valid country: %d\n", found)
//	fmt.Printf("records with unknown/invalid country: %d\n", notFound)
//	fmt.Printf("invalid countries:\n")
//	for invalidCountry := range unknownCountries {
//		fmt.Println("`" + invalidCountry + "`")
//	}
//	return nil
//}
