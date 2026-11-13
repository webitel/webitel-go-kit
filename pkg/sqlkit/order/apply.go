package order

type ApplyFunc[Q any] func(q Q, orders Clauses) Q

func Apply[Q any](s *Sorter, q Q, provided []string, ap ApplyFunc[Q]) (Q, error) {
	orders, err := s.Resolve(provided)
	if err != nil {
		return q, err
	}

	return ap(q, orders), nil
}
