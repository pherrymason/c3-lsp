package utils

import "testing"

func TestIsFeatureEnabled(t *testing.T) {
	SetRuntimeFeatureFlags(nil)
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

func TestIsFeatureEnabled_runtime_flags_override_defaults(t *testing.T) {
	SetRuntimeFeatureFlags(map[string]bool{
		"SIZE_ON_HOVER": true,
		"PERF_TRACE":    true,
		"USE_SEARCH_V2": false,
	})
	t.Cleanup(func() { SetRuntimeFeatureFlags(nil) })

	if !IsFeatureEnabled("SIZE_ON_HOVER") {
		t.Fatalf("expected SIZE_ON_HOVER to be enabled by runtime flags")
	}

	if !IsFeatureEnabled("PERF_TRACE") {
		t.Fatalf("expected PERF_TRACE to be enabled by runtime flags")
	}

	if IsFeatureEnabled("USE_SEARCH_V2") {
		t.Fatalf("expected USE_SEARCH_V2 to remain disabled when not enabled in runtime flags")
	}
}

func TestEnabledFeatureFlags_returns_sorted_enabled_flags(t *testing.T) {
	SetRuntimeFeatureFlags(map[string]bool{
		"PERF_TRACE":       true,
		"REQUEST_TIMEOUT":  true,
		"REQUEST_SLOW_LOG": true,
	})
	t.Cleanup(func() { SetRuntimeFeatureFlags(nil) })

	flags := EnabledFeatureFlags()
	if len(flags) != 3 {
		t.Fatalf("expected 3 enabled flags, got %d: %v", len(flags), flags)
	}

	expected := []string{"PERF_TRACE", "REQUEST_SLOW_LOG", "REQUEST_TIMEOUT"}
	for i := range expected {
		if flags[i] != expected[i] {
			t.Fatalf("unexpected enabled flags order/content: got %v expected %v", flags, expected)
		}
	}
}
