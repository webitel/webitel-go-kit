package order_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/sqlkit/order"
)

func newSorter() *order.Sorter {
	return order.New(
		map[string]string{
			"created": "o.created_at",
			"id":      "o.id",
			"total":   "o.total_amount",
		},
		order.WithDefaults(order.CreateField("created", order.Desc)),
		order.WithMaxFields(2),
	)
}

func ExampleParseField() {
	for _, raw := range []string{"created", "+created", "-created", "!created", "  -created  "} {
		f, err := order.ParseField(raw)
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf("%q -> %s %s\n", raw, f.Name, f.Dir)
	}

	// Output:
	// "created" -> created ASC
	// "+created" -> created ASC
	// "-created" -> created DESC
	// "!created" -> created DESC
	// "  -created  " -> created DESC
}

func ExampleParseField_empty() {
	_, err := order.ParseField("-")

	fmt.Println(err)
	fmt.Println(errors.Is(err, order.ErrEmptyField))

	// Output:
	// order: "-" empty field
	// true
}

func ExampleSorter_Resolve() {
	clauses, err := newSorter().Resolve([]string{"-created", "id"})
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, c := range clauses {
		fmt.Println(c.Expr, c.Dir)
	}

	// Output:
	// o.created_at DESC
	// o.id ASC
}

func ExampleSorter_Resolve_defaults() {
	// No sort parameters were provided, so the defaults are used.
	clauses, _ := newSorter().Resolve(nil)

	fmt.Println(clauses.SQL())

	// Output:
	// o.created_at DESC
}

func ExampleSorter_Resolve_errors() {
	s := newSorter()

	// A field that is not in the allow-list.
	_, err := s.Resolve([]string{"password"})
	fmt.Println(err)
	fmt.Println(errors.Is(err, order.ErrFieldNotAllowed))

	var oe *order.Error
	if errors.As(err, &oe) {
		fmt.Println("field:", oe.Field)
	}

	// The same field twice.
	_, err = s.Resolve([]string{"id", "-id"})
	fmt.Println(err)

	// More fields than WithMaxFields allows.
	_, err = s.Resolve([]string{"created", "id", "total"})
	fmt.Println(err)

	// Output:
	// order: "password" field not allowed
	// true
	// field: password
	// order: "id" duplicate field
	// order: too many fields
}

func ExampleSorter_SQL() {
	clause, err := newSorter().SQL([]string{"-total", "id"})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("SELECT * FROM orders o ORDER BY " + clause)

	// Output:
	// SELECT * FROM orders o ORDER BY o.total_amount DESC, o.id ASC
}

// query is a minimal stand-in for a query builder.
type query struct{ orderBy []string }

func ExampleApply() {
	// An ApplyFunc adapts the resolved clauses to any builder type.
	orderBy := func(q *query, cs order.Clauses) *query {
		q.orderBy = append(q.orderBy, cs.Clauses()...)

		return q
	}

	q, err := order.Apply(newSorter(), &query{}, []string{"!total", "+id"}, orderBy)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(strings.Join(q.orderBy, " | "))

	// Output:
	// o.total_amount DESC | o.id ASC
}

func ExampleApply_invalid() {
	orderBy := func(q *query, cs order.Clauses) *query { return q }

	q := &query{}

	// On error the original query is returned unchanged.
	got, err := order.Apply(newSorter(), q, []string{"unknown"}, orderBy)

	fmt.Println(err)
	fmt.Println(got == q)

	// Output:
	// order: "unknown" field not allowed
	// true
}
