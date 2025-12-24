package utils

import (
	"math"
	"testing"
)

func TestNearestRound(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected int
	}{
		{
			name:     "decimal less than 0.6 rounds down",
			input:    1.5,
			expected: 2,
		},
		{
			name:     "decimal equal to 0.6 rounds up",
			input:    1.6,
			expected: 2,
		},
		{
			name:     "decimal greater than 0.6 rounds up",
			input:    2.7,
			expected: 3,
		},
		{
			name:     "decimal less than 0.6 rounds down",
			input:    3.4,
			expected: 3,
		},
		{
			name:     "whole number returns itself",
			input:    5.0,
			expected: 5,
		},
		{
			name:     "negative number with decimal less than 0.6",
			input:    -2.3,
			expected: -2,
		},
		{
			name:     "negative number with decimal equal to 0.6",
			input:    -3.6,
			expected: -4,
		},
		{
			name:     "negative number with decimal greater than 0.6",
			input:    -1.8,
			expected: -2,
		},
		{
			name:     "very small decimal",
			input:    0.1,
			expected: 0,
		},
		{
			name:     "decimal close to 1",
			input:    4.99,
			expected: 5,
		},
		{
			name:     "decimal close to 1",
			input:    50.008,
			expected: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := math.Round(tt.input)
			if result != float64(tt.expected) {
				t.Errorf("NearestRound(%f) = %d, want %d", tt.input, int(result), tt.expected)
			}
		})
	}
}
