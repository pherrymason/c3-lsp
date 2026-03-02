package utils

import "testing"

func TestIsFeatureEnabled(t *testing.T) {
	tests := []struct {
		name     string
		feature  string
		expected bool
	}{
		{
			name:     "USE_SEARCH_V2 is disabled by default",
			feature:  "USE_SEARCH_V2",
			expected: false,
		},
		{
			name:     "SIZE_ON_HOVER is disabled by default",
			feature:  "SIZE_ON_HOVER",
			expected: false,
		},
		{
			name:     "Unknown feature returns false",
			feature:  "UNKNOWN_FEATURE",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFeatureEnabled(tt.feature)
			if result != tt.expected {
				t.Errorf("IsFeatureEnabled(%q) = %v, want %v", tt.feature, result, tt.expected)
			}
		})
	}
}
