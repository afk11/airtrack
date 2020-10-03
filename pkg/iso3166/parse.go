package iso3166

import (
	"bufio"
	"github.com/pkg/errors"
	"io"
	"unicode"
)

// isWord returns true if the string is not empty and consists
// only of letters.
func isWord(w string) bool {
	for i := range w {
		if !unicode.IsLetter(rune(w[i])) {
			return false
		}
	}
	return len(w) > 0
}

// ParseColumnFormat parses r as a 3 column file with a whitespace character as
// a separator. Column 1 is the alpha-2 code, column 2 is the alpha-3 code,
// column 3 is the countries name.
//
// Example:
// DJ DJI Djibouti
// SE SWE Sweden
// IR IRN Iran, Islamic Republic
// ZW ZWE Zimbabwe
func ParseColumnFormat(r io.Reader) ([][3]string, error) {
	scanner := bufio.NewScanner(r)
	var list [][3]string
	var lineCount int
	var alpha2, alpha3 string
	for scanner.Scan() {
		line := scanner.Text()
		// Permitting a country field with length 1, the shortest
		// possible line is 8 characters.
		if len(line) < 8 {
			return nil, errors.New("line must contain more than 8 characters")
		} else if !(line[2] == ' ' && line[6] == ' ') {
			return nil, errors.Errorf("invalid whitespace (line %d)", lineCount)
		}
		alpha2 = line[0:2]
		alpha3 = line[3:6]
		if !isWord(alpha2) {
			return nil, errors.Errorf("invalid alpha-2 (line %d)", lineCount)
		} else if !isWord(alpha3) {
			return nil, errors.Errorf("invalid alpha-3 (line %d)", lineCount)
		}
		list = append(list, [3]string{alpha2, alpha3, line[7:]})
		lineCount++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
