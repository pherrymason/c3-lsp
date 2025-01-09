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

	assert.Equal(t, 9, len(result.symbols))
	assert.Equal(t, "cat", result.symbols[0].Name)
	assert.Equal(t, "dog", result.symbols[1].Name)
	assert.Equal(t, "Colors", result.symbols[2].Name)
	assert.Equal(t, "Obj", result.symbols[3].Name)
	assert.Equal(t, "Err", result.symbols[4].Name)
	assert.Equal(t, "A_CONSTANT", result.symbols[5].Name)
	assert.Equal(t, "Int32", result.symbols[6].Name)
	assert.Equal(t, "foo", result.symbols[7].Name)
	assert.Equal(t, "method", result.symbols[8].Name)

	for _, symbol := range result.symbols {
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

	assert.Equal(t, 2, len(result.symbols))
	assert.Equal(t, "main", result.symbols[0].Name)
	assert.Equal(t, "cat", result.symbols[1].Name)

	for _, symbol := range result.symbols {
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

	assert.Equal(t, 2, len(result.symbols))
	assert.Equal(t, "Obj", result.symbols[0].Name)
	assert.Equal(t, Relation{SymbolID: 2, Tag: Method}, result.symbols[0].Children[0])
	assert.Equal(t, "method", result.symbols[1].Name)
}
