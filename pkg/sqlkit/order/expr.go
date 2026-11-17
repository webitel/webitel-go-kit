package order

type ExprFunc func(d Direction) Clauses

type ExprFactory func(name, expr string) ExprFunc

func Expr(expr string) ExprFunc {
	return func(d Direction) Clauses {
		return Clauses{{
			Expr: expr,
			Dir:  d,
		}}
	}
}
