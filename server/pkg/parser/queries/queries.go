package queries

import (
	_ "embed"
	"fmt"

	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	sitter "github.com/smacker/go-tree-sitter"
)

//go:embed symbols.scm
var symbolsQueryRaw []byte

//go:embed local-var-declaration.scm
var localVarDeclQueryRaw []byte

var SymbolsQuery, LocalVarDeclQuery *sitter.Query

func init() {
	var err error
	if SymbolsQuery, err = sitter.NewQuery(symbolsQueryRaw, cst.Language); err != nil {
		panic(fmt.Errorf("could not create query symbols: %v", err))
	}
	if LocalVarDeclQuery, err = sitter.NewQuery(localVarDeclQueryRaw, cst.Language); err != nil {
		panic(fmt.Errorf("could not create query local var declaration: %v", err))
	}
}
