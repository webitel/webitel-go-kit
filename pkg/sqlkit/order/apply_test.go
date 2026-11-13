package order

import (
	"bytes"
	"errors"
	"testing"
)

func TestApply(t *testing.T) {
	t.Parallel()

	allowedMap := map[string]string{
		"id":         "users.id",
		"name":       "users.name",
		"created_at": "posts.created_at",
	}

	type MockQuery struct {
		SQLQuery string
	}

	mockApplyFunc := func(q MockQuery, orders Clauses) MockQuery {
		clausesSQL := Clauses(orders).SQL()
		q.SQLQuery = "SELECT * FROM users ORDER BY " + clausesSQL
		return q
	}

	tests := []struct {
		name      string
		opts      []Option
		provided  []string
		initial   MockQuery
		ap        ApplyFunc[MockQuery]
		wantQuery MockQuery
		wantErr   error
	}{
		{
			name:      "Success: applies single order clause to query",
			provided:  []string{"name"},
			initial:   MockQuery{SQLQuery: "SELECT * FROM users"},
			ap:        mockApplyFunc,
			wantQuery: MockQuery{SQLQuery: "SELECT * FROM users ORDER BY users.name ASC"},
			wantErr:   nil,
		},
		{
			name:      "Success: applies multiple order clauses",
			provided:  []string{"id", "-name"},
			initial:   MockQuery{SQLQuery: "SELECT * FROM users"},
			ap:        mockApplyFunc,
			wantQuery: MockQuery{SQLQuery: "SELECT * FROM users ORDER BY users.id ASC, users.name DESC"},
			wantErr:   nil,
		},
		{
			name:      "Error: invalid field returns original query and error",
			provided:  []string{"password_hash"},
			initial:   MockQuery{SQLQuery: "SELECT * FROM users"},
			ap:        mockApplyFunc,
			wantQuery: MockQuery{SQLQuery: "SELECT * FROM users"},
			wantErr:   ErrFieldNotAllowed,
		},
		{
			name:     "SQLi: Semicolon stack query",
			provided: []string{"id; DROP TABLE users;--"},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "SQLi: Union-based injection",
			provided: []string{"id UNION SELECT 1,2,3--"},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "SQLi: Inline comment bypass",
			provided: []string{"name/*comment*/"},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "SQLi: Boolean tautology with quotes",
			provided: []string{"' OR '1'='1"},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "SQLi: SQL comment dash-dash",
			provided: []string{"users.id --"},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "SQLi: Hex/Binary encoded string attempt",
			provided: []string{"0x6964"},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "Extreme: Extremely long field name (DoS protection check)",
			provided: []string{string(bytes.Repeat([]byte("a"), 10000))},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "Extreme: Null bytes injection",
			provided: []string{"id\x00"},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "Extreme: Multiple mixed directional prefixes",
			provided: []string{"++--+-name"},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "Extreme: Unicode / Emojis in field name",
			provided: []string{"🔥name🔥"},
			wantErr:  ErrFieldNotAllowed,
		},
		{
			name:     "Extreme: Max fields boundary overflow (exactly max + 1)",
			opts:     []Option{WithMaxFields(2)},
			provided: []string{"id", "name", "created_at"},
			wantErr:  ErrTooManyFields,
		},
		{
			name:     "Extreme: Case sensitivity mismatch",
			provided: []string{"ID"},
			wantErr:  ErrFieldNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ap := tt.ap
			if ap == nil {
				ap = mockApplyFunc
			}

			initial := tt.initial
			if initial.SQLQuery == "" && tt.wantErr != nil {
				initial = MockQuery{SQLQuery: "SELECT * FROM users"}
			}

			s := New(allowedMap, tt.opts...)
			gotQuery, err := Apply(s, initial, tt.provided, ap)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Apply() expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Apply() error = %v, expected to unwrap to %v", err, tt.wantErr)
				}
				if tt.wantQuery.SQLQuery != "" && gotQuery != tt.wantQuery {
					t.Errorf("Apply() query modified on error: got %+v, want %+v", gotQuery, tt.wantQuery)
				}
			} else {
				if err != nil {
					t.Fatalf("Apply() unexpected error: %+v", err)
				}
				if gotQuery != tt.wantQuery {
					t.Errorf("Apply() = %+v, want %+v", gotQuery, tt.wantQuery)
				}
			}
		})
	}
}
