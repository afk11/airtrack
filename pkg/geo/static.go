package geo

import (
	"math"
)

type Geocoder interface {
	// ReverseGeocode takes a lat/lon and tries to return a location
	// name, and the distance to the location from the input point.
	// The final error value is set upon failure. If no error is returned,
	// the address + distance fields may be zero values if no results
	// are found
	ReverseGeocode(lat float64, lon float64) (string, float64, error)
}

// haversin(θ) function
func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

// distance between points in meters
func Distance(lat1, lon1, lat2, lon2 float64) float64 {
	// convert to radians
	// must cast radius as float to multiply later
	var la1, lo1, la2, lo2, r float64
	la1 = lat1 * math.Pi / 180
	lo1 = lon1 * math.Pi / 180
	la2 = lat2 * math.Pi / 180
	lo2 = lon2 * math.Pi / 180

	r = 6378100 // Earth radius in METERS

	// calculate
	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * r * math.Asin(math.Sqrt(h))
}
