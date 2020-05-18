package geo

import (
	"github.com/mmcloughlin/geohash"
	"sync"
)

type AirportRecord struct {
	// FilePath of airport, eg, Dublin
	Name string
	// ICAO airport code, eg, EIDW
	Code string
	// CountryCode is two letter country code, eg, IE
	CountryCode string
	// Latitude is milli degrees, with N/S suffix, eg, 5325.278N
	Latitude float64
	// Latitude is milli degrees, with E/W suffix, eg, 00616.206W
	Longitude float64
	// Elevation in meters, with meter suffix, eg, 9.0m
	Elevation float64
	// Style of airport
	Style string
}

type NearestAirportGeocoder struct {
	geoBuckets   map[string][]AirportRecord
	geoHashChars uint
	mu           sync.RWMutex
}

func (g *NearestAirportGeocoder) Register(locs []AirportRecord) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, loc := range locs {
		gh := geohash.EncodeWithPrecision(loc.Latitude, loc.Longitude, g.geoHashChars)
		_, ok := g.geoBuckets[gh]
		if !ok {
			g.geoBuckets[gh] = make([]AirportRecord, 0)
		}
		g.geoBuckets[gh] = append(g.geoBuckets[gh], loc)
	}
	return nil
}
func (g *NearestAirportGeocoder) findNearbyLocations(gh string) []AirportRecord {
	var airports []AirportRecord
	if inBucket, ok := g.geoBuckets[gh]; ok {
		for _, airport := range inBucket {
			airports = append(airports, airport)
		}
	}
	for _, ngh := range geohash.Neighbors(gh) {
		if inBucket, ok := g.geoBuckets[ngh]; ok {
			for _, airport := range inBucket {
				airports = append(airports, airport)
			}
		}
	}
	return airports
}
func (g *NearestAirportGeocoder) ReverseGeocode(lat float64, lon float64) (string, float64, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	lgh := geohash.EncodeWithPrecision(lat, lon, g.geoHashChars)
	airports := g.findNearbyLocations(lgh)
	var nearestAirport string
	var nearestAirportDistance float64
	n := len(airports)
	for i := 0; i < n; i++ {
		airport := airports[i]
		distance := Distance(airport.Latitude, airport.Longitude, lat, lon)
		if nearestAirport == "" || nearestAirportDistance > distance {
			nearestAirport = airport.Name
			nearestAirportDistance = distance
		}
	}
	return nearestAirport, nearestAirportDistance, nil
}
func NewNearestAirportGeocoder(ghlen uint) *NearestAirportGeocoder {
	return &NearestAirportGeocoder{
		geoHashChars: ghlen,
		geoBuckets:   make(map[string][]AirportRecord),
	}
}
