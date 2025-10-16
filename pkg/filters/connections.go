package filters

type Comparison int64

const (
	Equal Comparison = iota
	GreaterThan
	GreaterThanOrEqual
	LessThan
	LessThanOrEqual
	NotEqual
	Like
	ILike
	IsNull
	NotNull
	Contains    // applied to arrays
	NotContains // applied to arrays
)

type MultiComparison int64

const (
	In MultiComparison = iota
	NotIn

	// Quantified comparisons (column op ANY/ALL(values))
	EqAny
	EqAll
	GtAny
	GtAll
	GeAny
	GeAll
	LtAny
	LtAll
	LeAny
	LeAll

	// Pattern membership
	LikeAny
	ILikeAny
	NotLikeAny
)

type ConnectionType int64

const (
	And ConnectionType = 0
	Or  ConnectionType = 1
)
