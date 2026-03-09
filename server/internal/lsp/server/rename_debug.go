package server

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/commonlog"
)

func renameDebugEnabled() bool {
	return utils.IsFeatureEnabled("RENAME_DEBUG")
}

func renameDebugf(logger commonlog.Logger, format string, args ...any) {
	if !renameDebugEnabled() || logger == nil {
		return
	}

	logger.Info("[rename-debug]", "detail", fmt.Sprintf(format, args...))
}
