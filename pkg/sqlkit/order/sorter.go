package order

import (
	"slices"
	"strings"
)

type Option func(*Sorter)

func WithDefaults(f ...Field) Option       { return func(s *Sorter) { s.defaults = slices.Clone(f) } }
func WithMaxFields(n int) Option           { return func(s *Sorter) { s.maxFields = n } }
func WithExprFactory(f ExprFactory) Option { return func(s *Sorter) { s.exprFactory = f } }

type Sorter struct {
	allowed     map[string]ExprFunc
	defaults    []Field
	maxFields   int
	exprFactory ExprFactory
}

func New(allowed map[string]string, opts ...Option) *Sorter {
	s := &Sorter{
		allowed:     make(map[string]ExprFunc, len(allowed)),
		maxFields:   3,
		exprFactory: func(_, expr string) ExprFunc { return Expr(expr) },
	}

	for _, o := range opts {
		o(s)
	}

	for f, expr := range allowed {
		s.allowed[f] = s.exprFactory(f, expr)
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

	clauses := make(Clauses, 0, len(fields))

	for _, f := range fields {
		expand, ok := s.allowed[f.Name]
		if !ok {
			return nil, &Error{Field: f.Name, Err: ErrFieldNotAllowed}
		}

		clauses = append(clauses, expand(f.Dir)...)
	}

	return clauses, nil
}

func (s *Sorter) SQL(provided ...string) (string, error) {
	orders, err := s.Resolve(provided)
	if err != nil {
		return "", err
	}

	return orders.SQL(), nil
}
