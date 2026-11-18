package order

import (
	"errors"
	"fmt"
	"testing"
)

func TestSorter_Resolve(t *testing.T) {
	t.Parallel()

	allowedMap := map[string]string{
		"id":         "users.id",
		"name":       "users.name",
		"created_at": "posts.created_at",
		"status":     "status_col",
	}

	customDefault := Field{Name: "created_at", Dir: Desc}

	tests := []struct {
		name       string
		opts       []Option
		provided   []string
		wantOrders []Clause
		wantErr    error
		wantField  string
	}{
		{
			name:     "Standard single field ascending",
			provided: []string{"name"},
			wantOrders: []Clause{
				{Expr: "users.name", Dir: Asc},
			},
			wantErr: nil,
		},
		{
			name:     "Multiple fields with mixed directions",
			provided: []string{"id", "-created_at"},
			wantOrders: []Clause{
				{Expr: "users.id", Dir: Asc},
				{Expr: "posts.created_at", Dir: Desc},
			},
			wantErr: nil,
		},
		{
			name:     "Explicit plus prefix",
			provided: []string{"+status"},
			wantOrders: []Clause{
				{Expr: "status_col", Dir: Asc},
			},
			wantErr: nil,
		},
		{
			name:     "Empty provided list falls back to defaults",
			opts:     []Option{WithDefaults(customDefault)},
			provided: []string{},
			wantOrders: []Clause{
				{Expr: "posts.created_at", Dir: Desc},
			},
			wantErr: nil,
		},
		{
			name:     "Provided with spaces and empty strings falls back to defaults",
			opts:     []Option{WithDefaults(customDefault)},
			provided: []string{"   ", ""},
			wantOrders: []Clause{
				{Expr: "posts.created_at", Dir: Desc},
			},
			wantErr: nil,
		},
		{
			name:     "Exact max fields limit (default max = 3)",
			provided: []string{"id", "name", "status"},
			wantOrders: []Clause{
				{Expr: "users.id", Dir: Asc},
				{Expr: "users.name", Dir: Asc},
				{Expr: "status_col", Dir: Asc},
			},
			wantErr: nil,
		},
		{
			name:     "Custom max fields limit respected",
			opts:     []Option{WithMaxFields(1)},
			provided: []string{"id"},
			wantOrders: []Clause{
				{Expr: "users.id", Dir: Asc},
			},
			wantErr: nil,
		},
		{
			name:     "Exceeding default max fields limit (> 3)",
			provided: []string{"id", "name", "status", "created_at"},
			wantErr:  ErrTooManyFields,
		},
		{
			name:     "Exceeding custom max fields limit",
			opts:     []Option{WithMaxFields(1)},
			provided: []string{"id", "name"},
			wantErr:  ErrTooManyFields,
		},
		{
			name:      "Duplicate columns provided",
			provided:  []string{"name", "-name"},
			wantErr:   ErrDuplicateField,
			wantField: "name",
		},
		{
			name:      "Field not allowed in schema",
			provided:  []string{"password_hash"},
			wantErr:   ErrFieldNotAllowed,
			wantField: "password_hash",
		},
		{
			name:      "Invalid raw format propagation from Parse",
			provided:  []string{"-"},
			wantErr:   ErrEmptyField,
			wantField: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := New(allowedMap, tt.opts...)
			orders, err := s.Resolve(tt.provided)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Resolve() expected error %v, got nil", tt.wantErr)
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Resolve() error = %v, expected to unwrap to %v", err, tt.wantErr)
				}

				if tt.wantField != "" {
					var sortErr *Error
					if errors.As(err, &sortErr) {
						if sortErr.Field != tt.wantField {
							t.Errorf("Error.Field = %q, want %q", sortErr.Field, tt.wantField)
						}
					} else {
						t.Errorf("expected error to be *orderby.InvalidSortError, got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("Resolve() unexpected error: %+v", err)
				}

				if len(orders) != len(tt.wantOrders) {
					t.Fatalf("Resolve() got %d orders, want %d", len(orders), len(tt.wantOrders))
				}

				for i := range orders {
					if orders[i] != tt.wantOrders[i] {
						t.Errorf("orders[%d] = %+v, want %+v", i, orders[i], tt.wantOrders[i])
					}
				}
			}
		})
	}
}

func TestSorter_SQL(t *testing.T) {
	t.Parallel()

	allowedMap := map[string]string{
		"id":         "users.id",
		"name":       "users.name",
		"created_at": "posts.created_at",
		"status":     "status_col",
	}

	customDefault := Field{Name: "created_at", Dir: Desc}

	tests := []struct {
		name     string
		opts     []Option
		provided []string
		wantSQL  string
		wantErr  error
	}{
		{
			name:     "Single field ascending SQL",
			provided: []string{"name"},
			wantSQL:  "users.name ASC",
			wantErr:  nil,
		},
		{
			name:     "Multiple fields mixed directions SQL",
			provided: []string{"id", "-created_at"},
			wantSQL:  "users.id ASC, posts.created_at DESC",
			wantErr:  nil,
		},
		{
			name:     "Explicit plus prefix SQL",
			provided: []string{"+status"},
			wantSQL:  "status_col ASC",
			wantErr:  nil,
		},
		{
			name:     "Empty provided list falls back to defaults SQL",
			opts:     []Option{WithDefaults(customDefault)},
			provided: []string{},
			wantSQL:  "posts.created_at DESC",
			wantErr:  nil,
		},
		{
			name:     "Provided spaces falls back to defaults SQL",
			opts:     []Option{WithDefaults(customDefault)},
			provided: []string{"   ", ""},
			wantSQL:  "posts.created_at DESC",
			wantErr:  nil,
		},
		{
			name:     "Max fields limit reached SQL",
			provided: []string{"id", "name", "status"},
			wantSQL:  "users.id ASC, users.name ASC, status_col ASC",
			wantErr:  nil,
		},
		{
			name:     "Exceeding max fields returns error",
			provided: []string{"id", "name", "status", "created_at"},
			wantSQL:  "",
			wantErr:  ErrTooManyFields,
		},
		{
			name:     "Disallowed field returns error",
			provided: []string{"password_hash"},
			wantSQL:  "",
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "SQL Injection attempt returns error",
			provided: []string{"id; DROP TABLE users;--"},
			wantSQL:  "",
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "Duplicate fields returns error",
			provided: []string{"name", "-name"},
			wantSQL:  "",
			wantErr:  ErrDuplicateField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := New(allowedMap, tt.opts...)
			gotSQL, err := s.SQL(tt.provided...)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Sorter.SQL() expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Sorter.SQL() error = %v, expected to unwrap to %v", err, tt.wantErr)
				}
				if gotSQL != "" {
					t.Errorf("Sorter.SQL() expected empty string on error, got %q", gotSQL)
				}
			} else {
				if err != nil {
					t.Fatalf("Sorter.SQL() unexpected error: %+v", err)
				}
				if gotSQL != tt.wantSQL {
					t.Errorf("Sorter.SQL() = %q, want %q", gotSQL, tt.wantSQL)
				}
			}
		})
	}
}

func TestSorter_CustomExpr(t *testing.T) {
	t.Parallel()

	ccLookup := func(table, field, col string) ExprFunc {
		return func(d Direction) Clauses {
			return Clauses{
				{
					Expr: fmt.Sprintf("case when not call_center.cc_is_lookup(%s, %s) then %s end", table, field, col),
					Dir:  d,
				},
				{
					Expr: fmt.Sprintf(
						"case when call_center.cc_is_lookup(%s, %s) then coalesce((%s::text)::json->>'name', (%s::text)::json->0->>'name') end",
						table, field, col, col),
					Dir: d,
				},
			}
		}
	}

	lookupFactory := func(table string) ExprFactory {
		return func(field, col string) ExprFunc {
			return ccLookup(table, field, col)
		}
	}

	sorter := New(map[string]string{
		"created_at": "t.created_at",
		"to":         "case when c.direction = 'outbound' then t.from_number else t.to_number end",
		"call_id":    "c.id",
	}, WithDefaults(Field{Name: "created_at", Dir: Desc}), WithExprFactory(lookupFactory("call_center.cc_calls_history")))

	tests := []struct {
		name    string
		input   []string
		want    string
		wantErr error
	}{
		{
			name:  "allowed_created_at_asc",
			input: []string{"created_at"},
			want: "case when not call_center.cc_is_lookup(call_center.cc_calls_history, created_at) then t.created_at end ASC, " +
				"case when call_center.cc_is_lookup(call_center.cc_calls_history, created_at) then coalesce((t.created_at::text)::json->>'name', (t.created_at::text)::json->0->>'name') end ASC",
			wantErr: nil,
		},
		{
			name:  "allowed_call_id_desc",
			input: []string{"-call_id"},
			want: "case when not call_center.cc_is_lookup(call_center.cc_calls_history, call_id) then c.id end DESC, " +
				"case when call_center.cc_is_lookup(call_center.cc_calls_history, call_id) then coalesce((c.id::text)::json->>'name', (c.id::text)::json->0->>'name') end DESC",
			wantErr: nil,
		},
		{
			name:    "not enabled field",
			input:   []string{"deleted_at"},
			want:    "",
			wantErr: ErrFieldNotAllowed,
		},
		{
			name:  "default_created_at_field",
			input: []string{},
			want: "case when not call_center.cc_is_lookup(call_center.cc_calls_history, created_at) then t.created_at end DESC, " +
				"case when call_center.cc_is_lookup(call_center.cc_calls_history, created_at) then coalesce((t.created_at::text)::json->>'name', (t.created_at::text)::json->0->>'name') end DESC",
			wantErr: nil,
		},
		{
			name:  "complicated_sort_field",
			input: []string{"-to"},
			want: "case when not call_center.cc_is_lookup(call_center.cc_calls_history, to) then case when c.direction = 'outbound' then t.from_number else t.to_number end end DESC, " +
				"case when call_center.cc_is_lookup(call_center.cc_calls_history, to) then coalesce((case when c.direction = 'outbound' then t.from_number else t.to_number end::text)::json->>'name', (case when c.direction = 'outbound' then t.from_number else t.to_number end::text)::json->0->>'name') end DESC",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stmt, err := sorter.SQL(tt.input...)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Sorter.SQL(): error got %v want %v", err, tt.wantErr)
			}

			if stmt != tt.want {
				t.Errorf("Sorter.CustomExpr() got:\n%q\nwant:\n%q", stmt, tt.want)
			}
		})
	}
}
