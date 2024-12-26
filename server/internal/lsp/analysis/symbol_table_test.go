package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConvertToAST_registers_global_declarations(t *testing.T) {
	source := `
	module foo;
	int cat = 1;
	char dog = 2;
	enum Colors:int { RED, BLUE, GREEN }
	struct MyStruct { int data; }
	fault Err{OOPS,FIAL}
	const int A_CONSTANT = 12;
	`

	tree := factory.ConvertToAST(factory.GetCST(source), source, "file.c3")

	result := BuildSymbolTable(tree)

	assert.Equal(t, 6, len(result.symbols))
	assert.Equal(t, "cat", result.symbols[0].Name)
	assert.Equal(t, "dog", result.symbols[1].Name)
	assert.Equal(t, "Colors", result.symbols[2].Name)
	assert.Equal(t, "MyStruct", result.symbols[3].Name)
	assert.Equal(t, "Err", result.symbols[4].Name)
	assert.Equal(t, "A_CONSTANT", result.symbols[5].Name)
}
