package iso3166

import (
	"bufio"
	"io"
)

func ParseColumnFormat(r io.Reader) (*Store, error) {
	scanner := bufio.NewScanner(r)
	var list [][2]string
	for scanner.Scan() {
		line := scanner.Text()
		code := line[0:2]
		country := line[3:]
		list = append(list, [2]string{code, country})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return New(list)
}
