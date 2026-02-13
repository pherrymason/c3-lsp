package project_state

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/stdlib"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/tliron/commonlog"
)

// SupportedC3Version: The C3 language version supported by the LSP.
const SupportedC3Version = "0.7.10"

// LoadStdLib loads the standard library symbols for the given version.
// It will attempt to load the stdlib cache for the specified version or build it if not found.
func LoadStdLib(logger commonlog.Logger, version string, c3cLibPath string) symbols_table.UnitModules {
	// Get detected c3c binary version if available
	detectedVersion := stdlib.GetDetectedC3Version()

	// Warn if user's version doesn't match the detected binary version
	if detectedVersion != "" && detectedVersion != version {
		logger.Warningf("Requested C3 version %s does not match detected c3c binary version %s", version, detectedVersion)
		logger.Warning("This may cause inconsistencies. Consider updating your configuration.")
	}

	// Attempt to load stdlib for the requested version
	// This will try to load from cache, or build it if c3c path is configured
	logger.Infof("Loading stdlib for C3 version %s...", version)
	return stdlib.LoadStdlib(logger, version, c3cLibPath)
}
