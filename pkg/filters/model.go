package filters

type isFilter interface {
	isFilter()
}

// FilterExpr is a wrapper for either a Filter or a FilterNode.
type FilterExpr struct {
	filter isFilter
}

func (f *FilterExpr) GetFilter() *Filter {
	if x, ok := f.filter.(*Filter); ok {
		return x
	}
	return nil
}

func (f *FilterExpr) GetFilterNode() *FilterNode {
	if x, ok := f.filter.(*FilterNode); ok {
		return x
	}
	return nil
}

func (f *FilterExpr) GetMultiFilter() *MultiFilter[any] {
	if x, ok := f.filter.(*MultiFilter[any]); ok {
		return x
	}
	return nil
}

// Filter is a leave node in a filter tree.
// It represents a single condition that can be applied to a query.
type Filter struct {
	Column         string
	Value          any
	ComparisonType Comparison
}

func (f *Filter) isFilter() {}

// FilterNode is a node in a filter tree.
// It can contain multiple Filter or FilterNode instances and represents a logical connection (And/Or)
type FilterNode struct {
	Nodes      []*FilterExpr
	Connection ConnectionType
}

func (f *FilterNode) isFilter() {}

// MultiFilter is a leave node in a filter tree.
// It represents a condition that checks if a column's value is in a list of values or matches any/all of them.
type MultiFilter[T any] struct {
	Column         string
	Values         []T
	ComparisonType MultiComparison
}

func (f *MultiFilter[T]) isFilter() {}
