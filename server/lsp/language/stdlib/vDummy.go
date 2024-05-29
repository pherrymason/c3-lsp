package stdlib

import (
	"github.com/pherrymason/c3-lsp/lsp/unit_modules"
)

func Load_vdummy_stdlib() unit_modules.UnitModules {
	parsedModules := unit_modules.NewParsedModules("_stdlib")
	return parsedModules
}
