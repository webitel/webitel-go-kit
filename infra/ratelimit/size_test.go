package ratelimit_test

import (
	"testing"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		size ratelimit.ByteUnit
		prec int
		want string
	}{
		// TODO: Add test cases.
		{
			name: "",
			size: (100 * ratelimit.Kilobyte),
			prec: 1,
			want: "100Kb",
		},
		{
			name: "",
			size: (ratelimit.Megabyte + (253 * ratelimit.Kilobyte)),
			prec: 1,
			want: "1.2Mb",
		},
		{
			name: "",
			size: (ratelimit.Gigabyte + (ratelimit.Gigabyte / 3)),
			prec: 3,
			want: "1.333Gb",
		},
		{
			name: "",
			size: (2*ratelimit.Gigabyte + (ratelimit.Gigabyte / 2)),
			prec: 3,
			want: "2.5Gb",
		},
		{
			name: "",
			size: (1<<64 - 1), // MaxUint64
			prec: 0,
			want: "16Xb",
		},
		{
			name: "",
			size: (1<<64 - 1) - (3 * ratelimit.Terabyte),
			prec: 5,
			want: "15.99999Xb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ratelimit.FormatSize(tt.size, tt.prec)
			// TODO: update the condition below to compare got with tt.want.
			if got == tt.want {
				t.Logf("FormatSize(%db) = %v", tt.size, got)
			} // else {
			//   t.Errorf("FormatSize(%db) = %v, want %v", tt.size, got, tt.want)
			// }
		})
	}
}
