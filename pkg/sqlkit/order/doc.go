// Package order turns untrusted sort parameters, such as the values of a
// "?sort=-created&sort=id" query string, into safe SQL ORDER BY clauses.
//
// The package has no third-party dependencies and knows nothing about query
// builders.
//
// # Input format
//
// Each sort parameter is a single string consisting of an optional direction
// prefix followed by a field name:
//
//	created     ascending  (default)
//	+created    ascending  (explicit)
//	-created    descending
//	!created    descending (alias for "-", handy where "-" is awkward in URLs)
//
// Surrounding whitespace is ignored. Empty parameters are skipped, while a
// parameter that has a prefix but no name (for example "-") is rejected with
// [ErrEmptyField].
//
// # Field names and expressions
//
// Names supplied by the client are never placed into SQL directly. A [Sorter]
// is created with an allow-list that maps public field names to [ExprFunc]
// builders producing trusted SQL expressions:
//
//	sorter := order.New(order.Columns(map[string]string{
//	    "created": "o.created_at",
//	    "id":      "o.id",
//	}))
//
// A name that is not in the list is rejected with [ErrFieldNotAllowed].
// Directions are normalized to [Asc] or [Desc], so nothing else can end up in
// the generated SQL.
//
// # Resolving and Customization
//
// [Sorter.Resolve] parses the input, validates it and returns [Clauses].
// It rejects duplicate fields ([ErrDuplicateField]) and more fields than
// the configured maximum ([ErrTooManyFields], see [WithMaxFields]).
// If the input contains no fields, the defaults from [WithDefaults] are used.
//
// Custom or multi-expression SQL mappings per field can be defined directly in
// the allow-list map using [ExprFunc] (either alongside [Columns] or via a
// custom helper).
//
// All validation failures are reported as *[Error], which supports
// errors.Is against the Err* sentinels and errors.As for access to the
// offending field.
//
// # Rendering
//
// Clauses can be rendered directly into a SQL string:
//
//	clause, err := sorter.SQL(r.URL.Query()["sort"])
//	// "o.created_at DESC, o.id ASC"
package order
