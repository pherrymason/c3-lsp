package search

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/utils"
)

func symbolSearchTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("C3LSP_SEARCH_TIMEOUT_MS"))
	if raw != "" {
		ms, err := strconv.Atoi(raw)
		if err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}

	if utils.IsFeatureEnabled("SEARCH_TIMEOUT") {
		return 1500 * time.Millisecond
	}

	return 0
}

func symbolSearchMaxDepth() int {
	raw := strings.TrimSpace(os.Getenv("C3LSP_SEARCH_MAX_DEPTH"))
	if raw != "" {
		value, err := strconv.Atoi(raw)
		if err == nil && value > 0 {
			return value
		}
	}

	if utils.IsFeatureEnabled("SEARCH_DEPTH_CAP") {
		return 128
	}

	return 64
}
