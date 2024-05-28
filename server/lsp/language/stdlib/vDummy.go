package stdlib

import parser "github.com/pherrymason/c3-lsp/lsp/parser"

func Load_vdummy_stdlib() *parser.ParsedModules {
	parsedModules := parser.NewParsedModules("_stdlib")
	return &parsedModules
}
