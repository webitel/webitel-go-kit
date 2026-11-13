package order

import "testing"

func TestError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  Error
		want string
	}{
		{
			name: "without field, empty field error",
			err:  Error{Field: "", Err: ErrEmptyField},
			want: "order: empty field",
		},
		{
			name: "with field name, field not allowed error",
			err:  Error{Field: "password_hash", Err: ErrFieldNotAllowed},
			want: `order: "password_hash" field not allowed`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.err.Error(); err != tt.want {
				t.Errorf("Error(%+v) want: %q got: %q", tt.err, tt.want, err)
			}
		})
	}
}
