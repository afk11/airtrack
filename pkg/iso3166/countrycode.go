package iso3166

type (
	// AlphaTwoCountryCode - data type containing two character ISO3166 alpha-2 code
	AlphaTwoCountryCode string
	// AlphaTwoCountryCode - data type containing three character ISO3166 alpha-3 code
	AlphaThreeCountryCode string
)

// String - implements Stringer. Returns the two character code
// as a string.
func (cc AlphaTwoCountryCode) String() string {
	return string(cc)
}

// String - implements Stringer. Returns the three character code
// as a string.
func (cc AlphaThreeCountryCode) String() string {
	return string(cc)
}
