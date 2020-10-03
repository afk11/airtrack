package iso3166

// Country - contains information about the country
type Country struct {
	name string
}

// Name returns the country name
func (c Country) Name() string {
	return c.name
}
