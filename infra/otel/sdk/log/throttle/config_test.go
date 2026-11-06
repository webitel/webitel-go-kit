package throttle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	def := DefaultConfig()

	tests := []struct {
		in      string
		want    *Config
		wantErr bool
	}{
		{in: "off", want: nil},
		{in: "OFF", want: nil},
		{in: "", want: &def},
		{in: "on", want: &def},
		{in: "window=30s", want: &Config{Window: 30 * time.Second, First: 100, Every: 100, MaxKeys: 10000}},
		{in: " window=2m , first=5, every=10, keys=50 ", want: &Config{Window: 2 * time.Minute, First: 5, Every: 10, MaxKeys: 50}},
		{in: "window=0s", wantErr: true},
		{in: "first=0", wantErr: true},
		{in: "every=-1", wantErr: true},
		{in: "every=x", wantErr: true},
		{in: "rate=10", wantErr: true},
		{in: "first", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseConfig(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_withDefaults(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		assert.Equal(t, DefaultConfig(), Config{}.withDefaults())
	})

	t.Run("invalid fields", func(t *testing.T) {
		got := Config{Window: time.Second, Every: -1}.withDefaults()
		assert.Equal(t, Config{Window: time.Second, First: 100, Every: 100, MaxKeys: 10000}, got)
	})
}
