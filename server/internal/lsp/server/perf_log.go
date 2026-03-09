package server

import (
	"fmt"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/commonlog"
)

func perfEnabled() bool {
	return utils.IsFeatureEnabled("PERF_TRACE")
}

func perfLogf(logger commonlog.Logger, operation string, startedAt time.Time, format string, args ...any) {
	if !perfEnabled() {
		return
	}

	message := fmt.Sprintf(format, args...)
	logger.Info("[perf] operation completed", "operation", operation, "duration", time.Since(startedAt).String(), "detail", message)
}
