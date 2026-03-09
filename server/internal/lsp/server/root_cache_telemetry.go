package server

import "fmt"

type rootCacheTelemetry struct {
	Hits        uint64
	Misses      uint64
	DeltaHits   uint64
	DeltaMisses uint64
	HitRate     float64
}

func computeRootCacheTelemetry(startHits uint64, startMisses uint64, endHits uint64, endMisses uint64) rootCacheTelemetry {
	deltaHits := uint64(0)
	if endHits >= startHits {
		deltaHits = endHits - startHits
	}

	deltaMisses := uint64(0)
	if endMisses >= startMisses {
		deltaMisses = endMisses - startMisses
	}

	hitRate := float64(0)
	if total := endHits + endMisses; total > 0 {
		hitRate = float64(endHits) / float64(total)
	}

	return rootCacheTelemetry{
		Hits:        endHits,
		Misses:      endMisses,
		DeltaHits:   deltaHits,
		DeltaMisses: deltaMisses,
		HitRate:     hitRate,
	}
}

func formatRootCacheTelemetry(t rootCacheTelemetry) string {
	return fmt.Sprintf(
		"root_cache_hits=%d root_cache_misses=%d root_cache_hits_delta=%d root_cache_misses_delta=%d root_cache_hit_rate=%.4f",
		t.Hits,
		t.Misses,
		t.DeltaHits,
		t.DeltaMisses,
		t.HitRate,
	)
}
