package order

import "testing"

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, input string
		want        Direction
	}{
		{name: "valid desc direction", input: "DESC", want: Desc},
		{name: "input direction", input: "", want: Asc},
		{name: "invalid direction", input: "descending", want: Asc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if d := normalize(Direction(tt.input)); d != tt.want {
				t.Errorf("normalize(%q) want: %q got: %q", tt.input, tt.want, string(d))
			}
		})
	}

}
