package stdlib

import (
	"github.com/pherrymason/c3-lsp/lsp/symbols_table"
)

func Load_vdummy_stdlib() symbols_table.UnitModules {
	docId := "_stdlib"
	parsedModules := symbols_table.NewParsedModules(&docId)
	return parsedModules
}
