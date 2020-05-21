package iso3166

import (
	"fmt"
	"github.com/pkg/errors"
)

type Store struct {
	alpha2 map[AlphaTwoCountryCode]*Country
	alpha3 map[AlphaThreeCountryCode]*Country
}

func (s *Store) GetCountryCode(cc AlphaTwoCountryCode) (*Country, bool) {
	code, ok := s.alpha2[cc]
	if !ok {
		return nil, false
	}
	return code, true
}
func emptyStore() *Store {
	return &Store{
		alpha2: make(map[AlphaTwoCountryCode]*Country),
		alpha3: make(map[AlphaThreeCountryCode]*Country),
	}
}
func New(list [][3]string) (*Store, error) {
	s := emptyStore()
	for _, v := range list {
		if len(v[0]) != 2 {
			return nil, errors.New("alpha2 country code should be two characters")
		} else if len(v[1]) != 3 {
			return nil, errors.Errorf("alpha3 country code should be three characters (%s)", v[1])
		}
		country := Country{
			name: v[2],
		}
		a2 := AlphaTwoCountryCode(v[0])
		a3 := AlphaThreeCountryCode(v[1])
		_, a2Known := s.alpha2[a2]
		_, a3Known := s.alpha3[a3]
		if a2Known {
			return nil, fmt.Errorf("cannot use duplicate alpha2 country codes (%s)", a2)
		} else if a3Known {
			return nil, fmt.Errorf("cannot use duplicate alpha3 country codes (%s)", a3)
		}
		s.alpha2[a2] = &country
		s.alpha3[a3] = &country
	}
	return s, nil
}
