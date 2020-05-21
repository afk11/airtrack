package iso3166

import (
	"bufio"
	"io"
)

func ParseColumnFormat(r io.Reader) (*Store, error) {
	scanner := bufio.NewScanner(r)
	var list [][3]string
	for scanner.Scan() {
		line := scanner.Text()
		list = append(list, [3]string{line[0:2], line[3:6], line[7:]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return New(list)
}
