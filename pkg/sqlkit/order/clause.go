package order

import "strings"

type Clause struct {
	Expr string
	Dir  Direction
}

func (o Clause) SQL() string { return o.Expr + " " + string(o.Dir) }

type Clauses []Clause

func (o Clauses) Clauses() []string {
	out := make([]string, len(o))

	for i := range o {
		out[i] = o[i].SQL()
	}

	return out
}

func (o Clauses) SQL() string { return strings.Join(o.Clauses(), ", ") }
