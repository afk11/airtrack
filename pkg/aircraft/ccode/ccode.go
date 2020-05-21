package ccode

import (
	"bufio"
	"encoding/hex"
	"github.com/afk11/airtrack/pkg/iso3166"
	iradix "github.com/hashicorp/go-immutable-radix"
	"github.com/pkg/errors"
	"io"
)

type allocationRange struct {
	low   uint32
	high  uint32
	owner iso3166.AlphaTwoCountryCode
}

func icaoToInt(icao string) (uint32, error) {
	bs, err := hex.DecodeString(icao)
	if err != nil {
		return 0, err
	}
	return uint32(bs[0])<<16 |
		uint32(bs[1])<<8 |
		uint32(bs[2]), nil
}

func LoadCountryAllocations(r io.Reader, store *iso3166.Store) (CountryAllocationSearcher, error) {
	scanner := bufio.NewScanner(r)
	ccodes := iradix.New()
	for scanner.Scan() {
		line := scanner.Text()
		begin := line[0:6]
		end := line[8 : 8+6]

		allocation := line[16:]
		// unallocated ranges don't have two letter codes, skip these
		if len(allocation) != 2 {
			continue
		}
		cc := iso3166.AlphaTwoCountryCode(allocation)
		_, found := store.GetCountryCode(cc)
		if !found {
			return nil, errors.Errorf("unknown country code in allocations file (%s)", cc)
		}

		numLow, err := icaoToInt(begin)
		if err != nil {
			return nil, err
		}

		numHigh, err := icaoToInt(end)
		if err != nil {
			return nil, err
		}

		keyLen := -1
		for i := 0; i < 6; i++ {
			if begin[i] != end[i] {
				keyLen = i
				break
			}
		}
		if keyLen == -1 {
			panic("invalid start/end ")
		}
		key := begin[0:keyLen]

		ccodes, _, _ = ccodes.Insert([]byte(key), allocationRange{
			owner: cc,
			low:   numLow,
			high:  numHigh,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &RadixCountryAllocationSearcher{tree: ccodes}, nil
}

type CountryAllocationSearcher interface {
	DetermineCountryCode(icao string) (*iso3166.AlphaTwoCountryCode, error)
}

type RadixCountryAllocationSearcher struct {
	tree *iradix.Tree
}

func (cc *RadixCountryAllocationSearcher) DetermineCountryCode(k string) (*iso3166.AlphaTwoCountryCode, error) {
	icao, err := icaoToInt(k)
	if err != nil {
		panic(err)
	}
	var country *iso3166.AlphaTwoCountryCode
	kb := []byte(k)
	cc.tree.Root().WalkPath(kb, func(k []byte, v interface{}) bool {
		r := v.(allocationRange)
		if r.low <= icao && icao <= r.high {
			country = &r.owner
			return true
		}
		return false
	})
	return country, nil
}
