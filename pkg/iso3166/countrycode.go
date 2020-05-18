package iso3166

type CountryCode string

func (cc CountryCode) String() string {
	return string(cc)
}
