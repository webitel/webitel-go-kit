package order

import (
	"reflect"
	"testing"
)

func TestClause_SQL(t *testing.T) {
	t.Parallel()

	test := []struct {
		name, want string
		input      Clause
	}{
		{
			name: "simple column ascending",
			input: Clause{
				Expr: "id",
				Dir:  Asc,
			},
			want: "id ASC",
		},
		{
			name: "simple column descending",
			input: Clause{
				Expr: "created_at",
				Dir:  Desc,
			},
			want: "created_at DESC",
		},
		{
			name: "expression with SQL function",
			input: Clause{
				Expr: "LOWER(email)",
				Dir:  Asc,
			},
			want: "LOWER(email) ASC",
		},
		{
			name: "complex expression with table prefix",
			input: Clause{
				Expr: "users.score",
				Dir:  Desc,
			},
			want: "users.score DESC",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if sql := tt.input.SQL(); sql != tt.want {
				t.Errorf("Clause.SQL() got: %q, want: %q", sql, tt.want)
			}
		})
	}
}

func TestClauses_Clauses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input Clauses
		want  []string
	}{
		{
			name:  "Empty clauses slice",
			input: Clauses{},
			want:  []string{},
		},
		{
			name: "Single clause in slice",
			input: Clauses{
				{Expr: "id", Dir: Asc},
			},
			want: []string{"id ASC"},
		},
		{
			name: "Multiple clauses in slice",
			input: Clauses{
				{Expr: "created_at", Dir: Desc},
				{Expr: "name", Dir: Asc},
			},
			want: []string{"created_at DESC", "name ASC"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.input.Clauses()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Clauses.Clauses() got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func TestClauses_SQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input Clauses
		want  string
	}{
		{
			name:  "Empty clauses SQL",
			input: Clauses{},
			want:  "",
		},
		{
			name: "Single clause SQL",
			input: Clauses{
				{Expr: "id", Dir: Asc},
			},
			want: "id ASC",
		},
		{
			name: "Multiple clauses SQL joined correctly",
			input: Clauses{
				{Expr: "created_at", Dir: Desc},
				{Expr: "score", Dir: Asc},
			},
			want: "created_at DESC, score ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.input.SQL(); got != tt.want {
				t.Errorf("Clauses.SQL() got: %q, want: %q", got, tt.want)
			}
		})
	}
}
