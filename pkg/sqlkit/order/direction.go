package order

import "strings"

type Direction string

const (
	Asc  Direction = "ASC"
	Desc Direction = "DESC"
)

func normalize(d Direction) Direction {
	if strings.EqualFold(string(d), string(Desc)) {
		return Desc
	}

	return Asc
}
