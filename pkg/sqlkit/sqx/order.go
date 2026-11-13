package sqx

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/webitel/webitel-go-kit/pkg/sqlkit/order"
)

func OrderBy(sb sq.SelectBuilder, orders order.Clauses) sq.SelectBuilder {
	for _, o := range orders {
		sb = sb.OrderBy(o.SQL())
	}

	return sb
}

func ApplyOrder(s *order.Sorter, sb sq.SelectBuilder, raw []string) (sq.SelectBuilder, error) {
	return order.Apply(s, sb, raw, OrderBy)
}
