package stdlib

import (
	"github.com/pherrymason/c3-lsp/lsp/symbols_table"
)

func Load_vdummy_stdlib() symbols_table.UnitModules {
	parsedModules := symbols_table.NewParsedModules("_stdlib")
	return parsedModules
}
