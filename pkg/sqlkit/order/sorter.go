package order

import (
	"maps"
	"slices"
	"strings"
)

type Option func(*Sorter)

func WithDefaults(f ...Field) Option { return func(s *Sorter) { s.defaults = slices.Clone(f) } }
func WithMaxFields(n int) Option     { return func(s *Sorter) { s.maxFields = n } }

type Sorter struct {
	allowed   map[string]string
	defaults  []Field
	maxFields int
}

func New(allowed map[string]string, opts ...Option) *Sorter {
	s := &Sorter{allowed: maps.Clone(allowed), maxFields: 3}
	for _, o := range opts {
		o(s)
	}

	return s
}

func (s *Sorter) Resolve(provided []string) (Clauses, error) {
	fields := make([]Field, 0, len(provided))
	seen := make(map[string]struct{}, len(provided))

	for _, raw := range provided {
		if strings.TrimSpace(raw) == "" {
			continue
		}

		f, err := ParseField(raw)
		if err != nil {
			return nil, err
		}

		if _, exists := seen[f.Name]; exists {
			return nil, &Error{Field: f.Name, Err: ErrDuplicateField}
		}

		seen[f.Name] = struct{}{}

		fields = append(fields, f)
	}

	if len(fields) > s.maxFields {
		return nil, &Error{Err: ErrTooManyFields}
	}

	if len(fields) == 0 {
		fields = s.defaults
	}

	orders := make([]Clause, 0, len(fields)+1)

	for _, f := range fields {
		expr, ok := s.allowed[f.Name]
		if !ok {
			return nil, &Error{Field: f.Name, Err: ErrFieldNotAllowed}
		}

		orders = append(orders, Clause{
			Expr: expr,
			Dir:  f.Dir,
		})
	}

	return orders, nil
}

func (s *Sorter) SQL(provided []string) (string, error) {
	orders, err := s.Resolve(provided)
	if err != nil {
		return "", err
	}

	return orders.SQL(), nil
}
