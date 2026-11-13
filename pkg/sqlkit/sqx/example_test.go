package sqx_test

import (
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/webitel/webitel-go-kit/pkg/sqlkit/order"
	"github.com/webitel/webitel-go-kit/pkg/sqlkit/sqx"
)

func newSorter() *order.Sorter {
	return order.New(
		map[string]string{
			"created": "o.created_at",
			"id":      "o.id",
			"total":   "o.total_amount",
		},
		order.WithDefaults(order.CreateField("created", order.Desc)),
	)
}

func baseQuery() sq.SelectBuilder {
	return sq.Select("o.id", "o.total_amount").
		From("orders o").
		Where(sq.Eq{"o.status": "open"}).
		PlaceholderFormat(sq.Dollar)
}

func ExampleApplyOrder() {
	// Typically: r.URL.Query()["sort"] for "?sort=-total&sort=id".
	provided := []string{"-total", "id"}

	qb, err := sqx.ApplyOrder(newSorter(), baseQuery(), provided)
	if err != nil {
		fmt.Println(err)
		return
	}

	sql, args, _ := qb.ToSql()

	fmt.Println(sql)
	fmt.Println(args)

	// Output:
	// SELECT o.id, o.total_amount FROM orders o WHERE o.status = $1 ORDER BY o.total_amount DESC, o.id ASC
	// [open]
}

func ExampleApplyOrder_defaults() {
	// No sort parameters, so the sorter's defaults are applied.
	qb, _ := sqx.ApplyOrder(newSorter(), baseQuery(), nil)

	sql, _, _ := qb.ToSql()

	fmt.Println(sql)

	// Output:
	// SELECT o.id, o.total_amount FROM orders o WHERE o.status = $1 ORDER BY o.created_at DESC
}

func ExampleApplyOrder_invalid() {
	qb, err := sqx.ApplyOrder(newSorter(), baseQuery(), []string{"password"})

	fmt.Println(err)
	fmt.Println(errors.Is(err, order.ErrFieldNotAllowed))

	// The builder is returned unchanged: no ORDER BY was added.
	sql, _, _ := qb.ToSql()
	fmt.Println(sql)

	// Output:
	// order: "password" field not allowed
	// true
	// SELECT o.id, o.total_amount FROM orders o WHERE o.status = $1
}

func ExampleApplyOrder_existingOrderBy() {
	// Clauses are appended to an existing ORDER BY, not replaced.
	qb := baseQuery().OrderBy("o.priority DESC")

	qb, _ = sqx.ApplyOrder(newSorter(), qb, []string{"id"})

	sql, _, _ := qb.ToSql()

	fmt.Println(sql)

	// Output:
	// SELECT o.id, o.total_amount FROM orders o WHERE o.status = $1 ORDER BY o.priority DESC, o.id ASC
}

func ExampleOrderBy() {
	// Clauses resolved by hand, e.g. to reuse them for several queries.
	clauses, err := newSorter().Resolve([]string{"!total", "+id"})
	if err != nil {
		fmt.Println(err)
		return
	}

	sql, _, _ := sqx.OrderBy(baseQuery(), clauses).ToSql()

	fmt.Println(sql)

	// Output:
	// SELECT o.id, o.total_amount FROM orders o WHERE o.status = $1 ORDER BY o.total_amount DESC, o.id ASC
}

func ExampleOrderBy_withApply() {
	// OrderBy satisfies order.ApplyFunc[sq.SelectBuilder], so it can be
	// passed to the generic order.Apply directly.
	qb, err := order.Apply(newSorter(), baseQuery(), []string{"-created"}, sqx.OrderBy)
	if err != nil {
		fmt.Println(err)
		return
	}

	sql, _, _ := qb.ToSql()

	fmt.Println(sql)

	// Output:
	// SELECT o.id, o.total_amount FROM orders o WHERE o.status = $1 ORDER BY o.created_at DESC
}
