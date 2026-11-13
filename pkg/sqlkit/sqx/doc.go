// Package sqx integrates the sqlkit packages with squirrel
// (github.com/Masterminds/squirrel).
//
// It is a thin adapter layer: all parsing, validation and allow-listing is
// done by the dependency-free [order] package, while sqx only knows how to
// apply the result to a [squirrel.SelectBuilder]. Code that does not use
// squirrel can import order alone and never pull squirrel into its build.
//
// # Ordering
//
// [ApplyOrder] is the one-call entry point. It resolves untrusted sort
// parameters with an [order.Sorter] and appends the resulting ORDER BY
// clauses to the builder:
//
//	sorter := order.New(map[string]string{
//		"created": "o.created_at",
//		"id":      "o.id",
//	})
//
//	qb := sq.Select("*").From("orders o").Where(sq.Eq{"o.status": "open"})
//
//	qb, err := sqx.ApplyOrder(sorter, qb, r.URL.Query()["sort"])
//	if err != nil {
//		// The client sent an invalid sort, usually a 400 response.
//		var oe *order.Error
//		if errors.As(err, &oe) { ... }
//	}
//
//	sql, args, err := qb.ToSql()
//
// If the parameters are invalid, the builder is returned unchanged together
// with the error, so ignoring the error can never produce a half-built query.
//
// [OrderBy] is the lower-level building block. It takes already resolved
// [order.Clauses] and is useful when you resolve them yourself, for example
// to reuse the same clauses for several queries or to inspect them first.
// It is also the [order.ApplyFunc] for squirrel, so it can be passed to
// [order.Apply] directly.
//
// Both functions append to any ORDER BY the builder already has; they never
// replace it.
//
// # Other query builders
//
// sqx is a template for further adapters. Support for another builder
// (gorm, bun, ...) is one function with the signature
// func(q Q, cs order.Clauses) Q passed to [order.Apply], and requires no
// changes to the order package.
package sqx
