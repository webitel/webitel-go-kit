package sqx

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/webitel/webitel-go-kit/pkg/sqlkit/order"
)

func TestOrderBy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseSQL sq.SelectBuilder
		orders  []order.Clause
		wantSQL string
	}{
		{
			name:    "Empty orders slice leaves query unchanged",
			baseSQL: sq.Select("*").From("users"),
			orders:  []order.Clause{},
			wantSQL: "SELECT * FROM users",
		},
		{
			name:    "Single clause ascending",
			baseSQL: sq.Select("*").From("users"),
			orders: []order.Clause{
				{Expr: "users.name", Dir: order.Asc},
			},
			wantSQL: "SELECT * FROM users ORDER BY users.name ASC",
		},
		{
			name:    "Multiple clauses with mixed directions",
			baseSQL: sq.Select("id", "name").From("users"),
			orders: []order.Clause{
				{Expr: "users.id", Dir: order.Asc},
				{Expr: "posts.created_at", Dir: order.Desc},
			},
			wantSQL: "SELECT id, name FROM users ORDER BY users.id ASC, posts.created_at DESC",
		},
		{
			name:    "SQLi: Semicolon stack query in clause expression",
			baseSQL: sq.Select("*").From("users"),
			orders: []order.Clause{
				{Expr: "users.id; DROP TABLE users;--", Dir: order.Asc},
			},
			wantSQL: "SELECT * FROM users ORDER BY users.id; DROP TABLE users;-- ASC",
		},
		{
			name:    "SQLi: Union-based injection payload",
			baseSQL: sq.Select("*").From("users"),
			orders: []order.Clause{
				{Expr: "id UNION SELECT * FROM passwords", Dir: order.Desc},
			},
			wantSQL: "SELECT * FROM users ORDER BY id UNION SELECT * FROM passwords DESC",
		},
		{
			name:    "SQLi: Inline comment bypass attempt",
			baseSQL: sq.Select("*").From("users"),
			orders: []order.Clause{
				{Expr: "name/*comment*/", Dir: order.Asc},
			},
			wantSQL: "SELECT * FROM users ORDER BY name/*comment*/ ASC",
		},
		{
			name:    "SQLi: Boolean tautology with quotes",
			baseSQL: sq.Select("*").From("users"),
			orders: []order.Clause{
				{Expr: "' OR '1'='1", Dir: order.Asc},
			},
			wantSQL: "SELECT * FROM users ORDER BY ' OR '1'='1 ASC",
		},
		{
			name:    "Extreme: Extremely long expression string (DoS check)",
			baseSQL: sq.Select("*").From("users"),
			orders: []order.Clause{
				{Expr: string(bytes.Repeat([]byte("a"), 5000)), Dir: order.Asc},
			},
			wantSQL: "SELECT * FROM users ORDER BY " + string(bytes.Repeat([]byte("a"), 5000)) + " ASC",
		},
		{
			name:    "Extreme: Null bytes in expression",
			baseSQL: sq.Select("*").From("users"),
			orders: []order.Clause{
				{Expr: "users.id\x00", Dir: order.Asc},
			},
			wantSQL: "SELECT * FROM users ORDER BY users.id\x00 ASC",
		},
		{
			name:    "Extreme: Massive number of clauses (Stress / Performance boundary)",
			baseSQL: sq.Select("*").From("users"),
			orders: func() []order.Clause {
				clauses := make([]order.Clause, 100)
				for i := 0; i < 100; i++ {
					clauses[i] = order.Clause{Expr: fmt.Sprintf("col_%d", i), Dir: order.Asc}
				}
				return clauses
			}(),
			wantSQL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resBuilder := OrderBy(tt.baseSQL, tt.orders)
			gotSQL, args, err := resBuilder.ToSql()

			if err != nil {
				t.Fatalf("unexpected error compiling SQL: %v", err)
			}

			if tt.name == "Extreme: Massive number of clauses (Stress / Performance boundary)" {
				if gotSQL == "" {
					t.Error("expected non-empty SQL for massive clauses")
				}
			} else {
				if gotSQL != tt.wantSQL {
					t.Errorf("OrderBy() SQL = %q, want %q", gotSQL, tt.wantSQL)
				}
			}

			if len(args) != 0 {
				t.Errorf("OrderBy() unexpected bind arguments: %v", args)
			}
		})
	}
}

func TestApplyOrder(t *testing.T) {
	t.Parallel()

	allowedMap := map[string]string{
		"id":         "users.id",
		"name":       "users.name",
		"created_at": "posts.created_at",
	}

	tests := []struct {
		name     string
		provided []string
		baseSQL  sq.SelectBuilder
		wantSQL  string
		wantErr  error
	}{
		{
			name:     "Success: single field ascending sort",
			provided: []string{"name"},
			baseSQL:  sq.Select("*").From("users"),
			wantSQL:  "SELECT * FROM users ORDER BY users.name ASC",
			wantErr:  nil,
		},
		{
			name:     "Success: multiple fields with mixed directions",
			provided: []string{"id", "-created_at"},
			baseSQL:  sq.Select("id", "name").From("users"),
			wantSQL:  "SELECT id, name FROM users ORDER BY users.id ASC, posts.created_at DESC",
			wantErr:  nil,
		},
		{
			name:     "Error: disallowed field returns error",
			provided: []string{"password_hash"},
			baseSQL:  sq.Select("*").From("users"),
			wantSQL:  "",
			wantErr:  order.ErrFieldNotAllowed,
		},
		{
			name:     "Error: SQL Injection attempt returns error",
			provided: []string{"id; DROP TABLE users;--"},
			baseSQL:  sq.Select("*").From("users"),
			wantSQL:  "",
			wantErr:  order.ErrFieldNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := order.New(allowedMap)
			resBuilder, err := ApplyOrder(s, tt.baseSQL, tt.provided)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Apply() expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Apply() error = %v, expected to unwrap to %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("Apply() unexpected error: %+v", err)
				}

				gotSQL, args, err := resBuilder.ToSql()
				if err != nil {
					t.Fatalf("failed to compile resulting SQL: %v", err)
				}

				if gotSQL != tt.wantSQL {
					t.Errorf("Apply() SQL = %q, want %q", gotSQL, tt.wantSQL)
				}

				if len(args) != 0 {
					t.Errorf("Apply() unexpected arguments in ORDER BY: %v", args)
				}
			}
		})
	}
}
