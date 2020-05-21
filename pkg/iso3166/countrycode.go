package iso3166

type (
	AlphaTwoCountryCode   string
	AlphaThreeCountryCode string
)

func (cc AlphaTwoCountryCode) String() string {
	return string(cc)
}

func (cc AlphaThreeCountryCode) String() string {
	return string(cc)
}
