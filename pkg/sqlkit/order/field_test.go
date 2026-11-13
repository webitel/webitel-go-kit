package order

import (
	"errors"
	"testing"
)

func TestParseField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, input, wantRawField string
		wantField                 Field
		wantErr                   error
	}{
		{
			name:      "Standard ascending field without prefix",
			input:     "created_at",
			wantField: CreateField("created_at", Asc),
			wantErr:   nil,
		},
		{
			name:      "Ascending field with plus prefix",
			input:     "+name",
			wantField: CreateField("name", Asc),
			wantErr:   nil,
		},
		{
			name:      "Descending field with minus prefix",
			input:     "-id",
			wantField: CreateField("id", Desc),
			wantErr:   nil,
		},
		{
			name:      "Descending field with exclamation prefix",
			input:     "!status",
			wantField: CreateField("status", Desc),
			wantErr:   nil,
		},
		{
			name:      "Field with leading and trailing spaces",
			input:     "  email  ",
			wantField: CreateField("email", Asc),
			wantErr:   nil,
		},
		{
			name:      "Field with prefix and surrounding spaces",
			input:     " -price ",
			wantField: CreateField("price", Desc),
			wantErr:   nil,
		},
		{
			name:         "Empty string input",
			input:        "",
			wantErr:      ErrEmptyField,
			wantRawField: "",
		},
		{
			name:         "Whitespace only input",
			input:        "   ",
			wantErr:      ErrEmptyField,
			wantRawField: "   ",
		},
		{
			name:         "Only minus prefix",
			input:        "-",
			wantErr:      ErrEmptyField,
			wantRawField: "-",
		},
		{
			name:         "Only exclamation prefix",
			input:        "!",
			wantErr:      ErrEmptyField,
			wantRawField: "!",
		},
		{
			name:         "Only plus prefix",
			input:        "+",
			wantErr:      ErrEmptyField,
			wantRawField: "+",
		},
		{
			name:         "Prefix isolated with spaces",
			input:        "  -   ",
			wantErr:      ErrEmptyField,
			wantRawField: "  -   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field, err := ParseField(tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Parse(%q) expected error %v, got nil", tt.input, tt.wantErr)
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Parse(%q) error = %v, expected to unwrap to %v", tt.input, err, tt.wantErr)
				}

				var sortErr *Error
				if errors.As(err, &sortErr) {
					if sortErr.Field != tt.wantRawField {
						t.Errorf("InvalidSortError.Field = %q, want %q", sortErr.Field, tt.wantRawField)
					}
				} else {
					t.Errorf("expected error to be of type *InvalidSortError, got %T", err)
				}
			} else {
				if err != nil {
					t.Fatalf("Parse(%q) unexpected error: %+v", tt.input, err)
				}

				if field != tt.wantField {
					t.Errorf("Parse(%q) = %+v, want %+v", tt.input, field, tt.wantField)
				}
			}
		})
	}
}
