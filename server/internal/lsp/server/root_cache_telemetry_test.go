package server

import (
	"strings"
	"testing"
)

func TestFormatRootCacheTelemetry_includesExpectedFieldsAndRateFormat(t *testing.T) {
	telemetry := computeRootCacheTelemetry(5, 3, 8, 5)
	formatted := formatRootCacheTelemetry(telemetry)

	for _, key := range []string{
		"root_cache_hits=8",
		"root_cache_misses=5",
		"root_cache_hits_delta=3",
		"root_cache_misses_delta=2",
		"root_cache_hit_rate=0.6154",
	} {
		if !strings.Contains(formatted, key) {
			t.Fatalf("expected telemetry output to contain %q, got: %s", key, formatted)
		}
	}
}

func TestComputeRootCacheTelemetry_handlesZeroTotals(t *testing.T) {
	telemetry := computeRootCacheTelemetry(0, 0, 0, 0)
	if telemetry.HitRate != 0 {
		t.Fatalf("expected zero hit rate for zero totals, got %f", telemetry.HitRate)
	}

	formatted := formatRootCacheTelemetry(telemetry)
	if !strings.Contains(formatted, "root_cache_hit_rate=0.0000") {
		t.Fatalf("expected fixed precision hit rate formatting, got: %s", formatted)
	}
}
