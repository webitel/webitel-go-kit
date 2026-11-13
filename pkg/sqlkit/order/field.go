package order

import "strings"

type Field struct {
	Name string
	Dir  Direction
}

func CreateField(name string, direction Direction) Field {
	return Field{Name: name, Dir: normalize(direction)}
}

func ParseField(raw string) (Field, error) {
	s := strings.TrimSpace(raw)
	dir := Asc

	switch {
	case strings.HasPrefix(s, "-"), strings.HasPrefix(s, "!"):
		s, dir = s[1:], Desc
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	}

	if s = strings.TrimSpace(s); s == "" {
		return Field{}, &Error{Field: raw, Err: ErrEmptyField}
	}

	return Field{Name: s, Dir: dir}, nil
}
