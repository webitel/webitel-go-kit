package codec

import (
	"testing"
	"time"
)

func TestTimeStampIsValid(t *testing.T) {
	type args struct {
		layout string
		skrew  time.Duration
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{
			name: "common",
			args: args{
				layout: "2006-01-02 15:04:05.000",
				skrew:  time.Second,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TimeStampIsValid(tt.args.layout, tt.args.skrew); got != tt.want {
				t.Errorf("TimeStampIsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
