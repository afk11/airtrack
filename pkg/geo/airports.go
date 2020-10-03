package geo

import (
	"github.com/mmcloughlin/geohash"
	"sync"
)

// AirportRecord - defines an airport and information about
// its location
type AirportRecord struct {
	// Name of airport, eg, Dublin
	Name string
	// Code is the ICAO issued airport code, eg, EIDW
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

// NearestAirportGeocoder - implements Geocoder.
// Shards locations based on a geohash.
type NearestAirportGeocoder struct {
	// geoBuckets maps a geohash to a list of airports in that region
	geoBuckets map[string][]AirportRecord
	// geoHashChars sets out how many characters to use in geohashes.
	geoHashChars uint
	// mu - mutex for read/write access to geoBuckets
	mu sync.RWMutex
}

// Register adds a list of AirportRecords to the Geocoder.
func (g *NearestAirportGeocoder) Register(locs []AirportRecord) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range locs {
		gh := geohash.EncodeWithPrecision(locs[i].Latitude, locs[i].Longitude, g.geoHashChars)
		_, ok := g.geoBuckets[gh]
		if !ok {
			g.geoBuckets[gh] = make([]AirportRecord, 0)
		}
		g.geoBuckets[gh] = append(g.geoBuckets[gh], locs[i])
	}
	return nil
}

// findNearbyLocations takes a geohash, and returns airports in that bucket,
// plus any airports in neighbour geohashes.
func (g *NearestAirportGeocoder) findNearbyLocations(gh string) []AirportRecord {
	var airports []AirportRecord
	if inBucket, ok := g.geoBuckets[gh]; ok {
		airports = append(airports, inBucket...)
	}
	for _, ngh := range geohash.Neighbors(gh) {
		if inBucket, ok := g.geoBuckets[ngh]; ok {
			airports = append(airports, inBucket...)
		}
	}
	return airports
}

// ReverseGeocode - see Geocoder.ReverseGeocode
// Converts location to geohash, finds nearby locations, and returns
// the location with the shortest distance to (lat, lon) if there was
// any airports nearby.
func (g *NearestAirportGeocoder) ReverseGeocode(lat float64, lon float64) (string, float64) {
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
	return nearestAirport, nearestAirportDistance
}

// NewNearestAirportGeocoder takes ghlen and returns an initialized NearestAirportGeocoder
func NewNearestAirportGeocoder(ghlen uint) *NearestAirportGeocoder {
	return &NearestAirportGeocoder{
		geoHashChars: ghlen,
		geoBuckets:   make(map[string][]AirportRecord),
	}
}
