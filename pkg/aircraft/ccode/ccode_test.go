package ccode

import (
	"github.com/afk11/airtrack/pkg/iso3166"
	"strings"
	"testing"
)

func checkAircraftIcao(t *testing.T, store *iso3166.Store, searcher CountryAllocationSearcher, expectedCountry string, icao string) {
	acCode, err := searcher.DetermineCountryCode(icao)
	if err != nil {
		t.Errorf("error loading country country: %s", err.Error())
	} else if acCode == nil {
		t.Errorf("failed to find country country for icao %s", icao)
	}
	country, found := store.GetCountryCode(*acCode)
	if !found {
		t.Errorf("aircraft country code not found")
	} else if country.Name() != expectedCountry {
		t.Errorf("aircraft icao country didn't resolve as expected (%s != %s)", country.Name(), expectedCountry)
	}
}
func checkAircraftIcaoRange(t *testing.T, store *iso3166.Store, searcher CountryAllocationSearcher, expectedCountry string, icaos []string) {
	for _, icao := range icaos {
		checkAircraftIcao(t, store, searcher, expectedCountry, icao)
	}
}
func TestLoadCountryAllocations(t *testing.T) {
	sample := `000000 	003FFF 	(unallocated 1)
098000 	0983FF 	DJ
4A8000 	4AFFFF 	SE
730000 	737FFF 	IR
004000 	0043FF 	ZW
09A000 	09AFFF 	GM`
	store, err := iso3166.New([][3]string{
		{"DJ", "DJI", "Djibouti"},
		{"SE", "SWE", "Sweden"},
		{"IR", "IRN", "Iran, Islamic Republic"},
		{"ZW", "ZWE", "Zimbabwe"},
		{"GM", "GMB", "Gambia"},
	})
	if err != nil {
		t.Errorf("error building country store for test: %s", err.Error())
	}

	searcher, err := LoadCountryAllocations(strings.NewReader(sample), store)
	if err != nil {
		t.Errorf("error loading country allocations: %s", err.Error())
	}

	djibouti := []string{
		"098000", "098001", "098010", "098100", "0980ff",
		"09800f", "0982ff", "098300", "098301", "0983ff"}
	checkAircraftIcaoRange(t, store, searcher, "Djibouti", djibouti)

	sweden := []string{
		"4A8000", "4A8001", "4A8011", "4A8111", "4A8FFF",
		"4A9000", "4AA000", "4AB000", "4AC000", "4AD000",
		"4AE000", "4AF000",
		"4A9FFF", "4AAFFF", "4ABFFF", "4ACFFF", "4ADFFF",
		"4AEFFF", "4AFFFF",
	}
	checkAircraftIcaoRange(t, store, searcher, "Sweden", sweden)

	iran := []string{
		"730000", "737FFF", "731000", "731100", "731110",
		"731111",
	}
	checkAircraftIcaoRange(t, store, searcher, "Iran, Islamic Republic", iran)

	acCode, err := searcher.DetermineCountryCode("097000")
	if err != nil {
		t.Errorf("unexpected error %s", err.Error())
	} else if acCode != nil {
		t.Errorf("test failure - should not have found a country code...")
	}
	acCode, err = searcher.DetermineCountryCode("980000")
	if err != nil {
		t.Errorf("unexpected error %s", err.Error())
	} else if acCode != nil {
		t.Errorf("test failure - should not have found a country code...")
	}
}
