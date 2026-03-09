package search

import (
	"os"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/utils"
)

func TestSymbolSearchTimeout_fromFeatureFlag(t *testing.T) {
	_ = os.Unsetenv("C3LSP_SEARCH_TIMEOUT_MS")
	utils.SetRuntimeFeatureFlags(map[string]bool{"SEARCH_TIMEOUT": true})
	t.Cleanup(func() { utils.SetRuntimeFeatureFlags(nil) })

	if got := symbolSearchTimeout(); got != 1500*time.Millisecond {
		t.Fatalf("expected 1500ms timeout from feature flag, got %s", got)
	}
}

func TestSymbolSearchTimeout_fromEnv(t *testing.T) {
	if err := os.Setenv("C3LSP_SEARCH_TIMEOUT_MS", "25"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("C3LSP_SEARCH_TIMEOUT_MS") })

	if got := symbolSearchTimeout(); got != 25*time.Millisecond {
		t.Fatalf("expected 25ms timeout from env, got %s", got)
	}
}

func TestSymbolSearchMaxDepth_default(t *testing.T) {
	_ = os.Unsetenv("C3LSP_SEARCH_MAX_DEPTH")
	utils.SetRuntimeFeatureFlags(nil)

	if got := symbolSearchMaxDepth(); got != 64 {
		t.Fatalf("expected default max depth 64, got %d", got)
	}
}
