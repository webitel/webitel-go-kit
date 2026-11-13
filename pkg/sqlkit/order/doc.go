// Package order turns untrusted sort parameters, such as the values of a
// "?sort=-created,id" query string, into safe SQL ORDER BY clauses.
//
// The package has no third-party dependencies and knows nothing about query
// builders. Integrations live in separate packages (for example sqx for
// squirrel) and plug in through [ApplyFunc].
//
// # Input format
//
// Each sort parameter is a single string consisting of an optional direction
// prefix followed by a field name:
//
//	created    ascending  (default)
//	+created   ascending  (explicit)
//	-created   descending
//	!created   descending (alias for "-", handy where "-" is awkward in URLs)
//
// Surrounding whitespace is ignored. Empty parameters are skipped, while a
// parameter that has a prefix but no name (for example "-") is rejected with
// [ErrEmptyField].
//
// # Field names and expressions
//
// Names supplied by the client are never placed into SQL directly. A [Sorter]
// is created with an allow-list that maps public field names to trusted SQL
// expressions:
//
//	sorter := order.New(map[string]string{
//		"created": "o.created_at",
//		"id":      "o.id",
//	})
//
// A name that is not in the list is rejected with [ErrFieldNotAllowed].
// Directions are normalized to [Asc] or [Desc], so nothing else can end up in
// the generated SQL.
//
// # Resolving
//
// [Sorter.Resolve] parses the input, validates it and returns [Clauses].
// It rejects duplicate fields ([ErrDuplicateField]) and more fields than
// the configured maximum ([ErrTooManyFields], see [WithMaxFields]).
// If the input contains no fields, the defaults from [WithDefaults] are used.
//
// All validation failures are reported as *[Error], which supports
// errors.Is against the Err* sentinels and errors.As for access to the
// offending field.
//
// # Rendering
//
// Clauses can be rendered without any query builder:
//
//	clause, err := sorter.SQL(r.URL.Query()["sort"])
//	// "o.created_at DESC, o.id ASC"
//
// or applied to a query builder of any type through [Apply] and an
// [ApplyFunc]. See the sqx package for squirrel.
package order
