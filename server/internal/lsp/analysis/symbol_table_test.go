package analysis

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSymbolBuild_registers_global_declarations(t *testing.T) {
	source := `
	module foo;
	int cat = 1;
	char dog = 2;
	enum Colors:int { RED, BLUE, GREEN }
	struct Obj { int data; }
	fault Err{OOPS,FIAL}
	const int A_CONSTANT = 12;
	def Int32 = int;
	fn void foo() {} 
	fn void Obj.method() {}
	`

	astConverter := factory.NewASTConverter()
	tree := astConverter.ConvertToAST(factory.GetCST(source), source, "file.c3")

	result := BuildSymbolTable(tree, "")

	scope := result.scopeTree["file.c3"]
	assert.Equal(t, 9, len(scope.Symbols))
	assert.Equal(t, "cat", scope.Symbols[0].Name)
	assert.Equal(t, "dog", scope.Symbols[1].Name)
	assert.Equal(t, "Colors", scope.Symbols[2].Name)
	assert.Equal(t, "Obj", scope.Symbols[3].Name)
	assert.Equal(t, "Err", scope.Symbols[4].Name)
	assert.Equal(t, "A_CONSTANT", scope.Symbols[5].Name)
	assert.Equal(t, "Int32", scope.Symbols[6].Name)
	assert.Equal(t, "foo", scope.Symbols[7].Name)
	assert.Equal(t, "method", scope.Symbols[8].Name)

	for _, symbol := range scope.Symbols {
		assert.Equal(t, "file.c3", symbol.FilePath, fmt.Sprintf("Symbol %s does not have expected filepath: %s", symbol.Name, symbol.FilePath))
	}
}

func TestSymbolBuild_registers_local_declarations(t *testing.T) {
	source := `
	module foo;
	fn void main() {
		int cat = 1;
	}`

	astConverter := factory.NewASTConverter()
	tree := astConverter.ConvertToAST(factory.GetCST(source), source, "file.c3")

	result := BuildSymbolTable(tree, "")

	scope := result.scopeTree["file.c3"]
	assert.Equal(t, 1, len(scope.Symbols))
	assert.Equal(t, "main", scope.Symbols[0].Name)
	scope = scope.Children[0]
	assert.Equal(t, "cat", scope.Symbols[0].Name)

	for _, symbol := range scope.Symbols {
		assert.Equal(t, "file.c3", symbol.FilePath, fmt.Sprintf("Symbol %s does not have expected filepath: %s", symbol.Name, symbol.FilePath))
	}
}

func TestSymbolBuild_registers_methods_in_the_right_struct(t *testing.T) {
	source := `
	module foo;
	struct Obj { int data; }
	fn void Obj.method() {}
	`

	astConverter := factory.NewASTConverter()
	tree := astConverter.ConvertToAST(factory.GetCST(source), source, "file.c3")

	result := BuildSymbolTable(tree, "")

	scope := result.scopeTree["file.c3"]
	assert.Equal(t, 2, len(scope.Symbols))
	assert.Equal(t, "Obj", scope.Symbols[0].Name)
	assert.Equal(t, Relation{Child: scope.Symbols[1], Tag: Method}, scope.Symbols[0].Children[0])
	assert.Equal(t, "method", scope.Symbols[1].Name)
}