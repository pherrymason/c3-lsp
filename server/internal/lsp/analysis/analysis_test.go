package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func getTree(source string, fileName string) ast.File {
	astConverter := factory.NewASTConverter()
	tree := astConverter.ConvertToAST(factory.GetCST(source), source, fileName)

	return tree
}

func TestFindsSymbol_Declaration_variable(t *testing.T) {

	t.Run("Find global variable declaration in same module", func(t *testing.T) {
		source := `int number = 0;
		fn void main(){number + 2;}`

		tree := getTree(source, "app.c3")
		symbolTable := BuildSymbolTable(tree)

		cursorPosition := lsp.Position{Line: 1, Column: 18}
		symbol := FindSymbolAtPosition(cursorPosition, symbolTable, tree)

		assert.Equal(t, "number", symbol.Name)
		assert.Equal(t, lsp.NewRange(0, 4, 0, 10), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.Type.Name)
	})
}
