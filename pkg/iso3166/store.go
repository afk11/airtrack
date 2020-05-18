package iso3166

import (
	"errors"
	"fmt"
)

type Store struct {
	countries map[CountryCode]Country
}

func (s *Store) GetCountryCode(cc CountryCode) (*Country, bool) {
	code, ok := s.countries[cc]
	if !ok {
		return nil, false
	}
	return &code, true
}
func emptyStore() *Store {
	return &Store{
		countries: make(map[CountryCode]Country),
	}
}
func New(list [][2]string) (*Store, error) {
	s := emptyStore()
	for _, v := range list {
		if len(v[0]) != 2 {
			return nil, errors.New("country code should be two characters")
		}
		cc := CountryCode(v[0])
		_, ok := s.countries[cc]
		if ok {
			return nil, fmt.Errorf("cannot use duplicate country codes (%s)", v[0])
		}
		s.countries[cc] = Country{
			name: v[1],
		}
	}
	return s, nil
}
