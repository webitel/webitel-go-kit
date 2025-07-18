// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package stdout // import "github.com/webitel/webitel-go-kit/infra/otel/log/stdout"

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	testCases := []struct {
		name     string
		options  []Option
		expected config
	}{
		{
			name: "default",
			expected: config{
				Output:      os.Stdout,
				PrettyPrint: false,
				Timestamps:  true,
			},
		},
		{
			name:    "WithWriter",
			options: []Option{WithWriter(os.Stderr)},
			expected: config{
				Output:      os.Stderr,
				PrettyPrint: false,
				Timestamps:  true,
			},
		},
		{
			name:    "WithPrettyPrint",
			options: []Option{WithPrettyPrint()},
			expected: config{
				Output:      os.Stdout,
				PrettyPrint: true,
				Timestamps:  true,
			},
		},
		{
			name:    "WithoutTimestamps",
			options: []Option{WithoutTimestamps()},
			expected: config{
				Output:      os.Stdout,
				PrettyPrint: false,
				Timestamps:  false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newConfig(tc.options)
			assert.Equal(t, tc.expected, cfg)
		})
	}
}
