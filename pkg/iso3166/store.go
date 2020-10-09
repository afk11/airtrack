package iso3166

import (
	"fmt"
	"github.com/pkg/errors"
)

type countryCodes struct {
	alpha2 AlphaTwoCountryCode
	alpha3 AlphaThreeCountryCode
}

// Store contains indexes allowing countries to be searched by
// their alpha2 or alpha3 code.
type Store struct {
	countries map[string]countryCodes
	alpha2    map[AlphaTwoCountryCode]*Country
	alpha3    map[AlphaThreeCountryCode]*Country
}

// GetAlphaTwoCodes returns a list of all alpha-2 codes.
func (s *Store) GetAlphaTwoCodes() []AlphaTwoCountryCode {
	codes := make([]AlphaTwoCountryCode, 0, len(s.alpha2))
	for code := range s.alpha2 {
		codes = append(codes, code)
	}
	return codes
}

// GetCountryCode searches for country info using the alpha-2 code cc.
// The second argument indicates whether the search was successful.
func (s *Store) GetCountryCode(cc AlphaTwoCountryCode) (*Country, bool) {
	code, ok := s.alpha2[cc]
	if !ok {
		return nil, false
	}
	return code, true
}

// emptyStore initializes an empty Store
func emptyStore() *Store {
	return &Store{
		countries: make(map[string]countryCodes),
		alpha2:    make(map[AlphaTwoCountryCode]*Country),
		alpha3:    make(map[AlphaThreeCountryCode]*Country),
	}
}

// New creates a Store initialized with the parsed country file in list.
func New(list [][3]string) (*Store, error) {
	s := emptyStore()
	for _, v := range list {
		if len(v[0]) != 2 {
			return nil, errors.New("alpha2 countryCodes code should be two characters")
		} else if len(v[1]) != 3 {
			return nil, errors.Errorf("alpha3 countryCodes code should be three characters (%s)", v[1])
		}
		country := Country{
			name: v[2],
		}
		a2 := AlphaTwoCountryCode(v[0])
		a3 := AlphaThreeCountryCode(v[1])
		_, a2Known := s.alpha2[a2]
		_, a3Known := s.alpha3[a3]
		if a2Known {
			return nil, fmt.Errorf("cannot use duplicate alpha2 countryCodes codes (%s)", a2)
		} else if a3Known {
			return nil, fmt.Errorf("cannot use duplicate alpha3 countryCodes codes (%s)", a3)
		}
		s.alpha2[a2] = &country
		s.alpha3[a3] = &country
		s.countries[v[2]] = countryCodes{alpha2: a2, alpha3: a3}
	}
	return s, nil
}
