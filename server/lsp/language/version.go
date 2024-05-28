package language

import (
	"github.com/pherrymason/c3-lsp/lsp/parser"
)

type stdLibFunc func() *parser.ParsedModules

type Version struct {
	number        string
	stdLibSymbols stdLibFunc
}
