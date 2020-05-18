package iso3166

type Country struct {
	name string
}

func (c Country) Name() string {
	return c.name
}
