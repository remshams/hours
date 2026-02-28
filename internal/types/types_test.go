package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTaskStatus(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    TaskStatus
		expectError bool
	}{
		{
			name:     "active",
			input:    TSValueActive,
			expected: TaskStatusActive,
		},
		{
			name:     "inactive",
			input:    TSValueInactive,
			expected: TaskStatusInactive,
		},
		{
			name:     "any",
			input:    TSValueAny,
			expected: TaskStatusAny,
		},
		{
			name:        "unknown value returns error",
			input:       "unknown",
			expectError: true,
		},
		{
			name:        "empty string returns error",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaskStatus(tt.input)
			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrIncorrectTaskStatusProvided)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestHumanizeDuration(t *testing.T) {
	testCases := []struct {
		name     string
		input    int
		expected string
	}{
		{
			name:     "0 seconds",
			input:    0,
			expected: "0s",
		},
		{
			name:     "30 seconds",
			input:    30,
			expected: "30s",
		},
		{
			name:     "60 seconds",
			input:    60,
			expected: "1m",
		},
		{
			name:     "1805 seconds",
			input:    1805,
			expected: "30m",
		},
		{
			name:     "3605 seconds",
			input:    3605,
			expected: "1h",
		},
		{
			name:     "4200 seconds",
			input:    4200,
			expected: "1h 10m",
		},
		{
			name:     "87000 seconds",
			input:    87000,
			expected: "24h 10m",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := HumanizeDuration(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
