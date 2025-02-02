package analysis

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
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
	macro m(x) { return x + 2;}
	`

	astConverter := factory.NewASTConverter()
	tree := astConverter.ConvertToAST(factory.GetCST(source).RootNode(), source, "file.c3")

	result := BuildSymbolTable(tree, "")

	modulesGroup := result.scopeTree["file.c3"]
	scope := modulesGroup.GetModuleScope("foo")
	assert.Equal(t, 15, len(scope.Symbols))
	assert.Equal(t, "cat", scope.Symbols[0].Name)
	assert.Equal(t, "dog", scope.Symbols[1].Name)
	assert.Equal(t, "Colors", scope.Symbols[2].Name)
	assert.Equal(t, "RED", scope.Symbols[3].Name)
	assert.Equal(t, "BLUE", scope.Symbols[4].Name)
	assert.Equal(t, "GREEN", scope.Symbols[5].Name)
	assert.Equal(t, "Obj", scope.Symbols[6].Name)
	assert.Equal(t, "Err", scope.Symbols[7].Name)
	assert.Equal(t, "OOPS", scope.Symbols[8].Name)
	assert.Equal(t, "FIAL", scope.Symbols[9].Name)
	assert.Equal(t, "A_CONSTANT", scope.Symbols[10].Name)
	assert.Equal(t, "Int32", scope.Symbols[11].Name)
	assert.Equal(t, "foo", scope.Symbols[12].Name)
	assert.Equal(t, "method", scope.Symbols[13].Name)
	assert.Equal(t, "m", scope.Symbols[14].Name)
	assert.Equal(t, ast.Token(ast.MACRO), scope.Symbols[14].Kind)

	for _, symbol := range scope.Symbols {
		assert.Equal(t, "file.c3", symbol.URI, fmt.Sprintf("Symbol %s does not have expected filepath: %s", symbol.Name, symbol.URI))
	}
}

func TestSymbolBuild_registers_structs(t *testing.T) {
	t.Run("Registers struct member with anonymous sub struct", func(t *testing.T) {
		source := `module test;
		struct Bar {
			struct data {
			  int a;
			}
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)
		modulesGroup := symbolTable.scopeTree[fileName]
		scope := modulesGroup.GetModuleScope("test")

		assert.Equal(t, "Bar", scope.Symbols[0].Name)
	})
}

func TestSymbolBuild_registers_local_declarations(t *testing.T) {
	source := `
	module foo;
	fn void main() {
		int cat = 1;
	}`

	astConverter := factory.NewASTConverter()
	tree := astConverter.ConvertToAST(factory.GetCST(source).RootNode(), source, "file.c3")

	result := BuildSymbolTable(tree, "")

	modulesGroup := result.scopeTree["file.c3"]
	scope := modulesGroup.GetModuleScope("foo")
	assert.Equal(t, 1, len(scope.Symbols))
	assert.Equal(t, "main", scope.Symbols[0].Name)
	scope = scope.Children[0]
	assert.Equal(t, "cat", scope.Symbols[0].Name)

	for _, symbol := range scope.Symbols {
		assert.Equal(t, "file.c3", symbol.URI, fmt.Sprintf("Symbol %s does not have expected filepath: %s", symbol.Name, symbol.URI))
	}
}

func TestSymbolBuild_registers_methods_in_the_right_struct(t *testing.T) {
	source := `
	module foo;
	struct Obj { int data; }
	fn void Obj.method() {}
	`

	astConverter := factory.NewASTConverter()
	tree := astConverter.ConvertToAST(factory.GetCST(source).RootNode(), source, "file.c3")

	result := BuildSymbolTable(tree, "")

	modulesGroup := result.scopeTree["file.c3"]
	scope := modulesGroup.GetModuleScope("foo")
	assert.Equal(t, 2, len(scope.Symbols))
	assert.Equal(t, "Obj", scope.Symbols[0].Name)
	assert.Equal(t, Relation{Child: scope.Symbols[1], Tag: Method}, scope.Symbols[0].Children[0])
	assert.Equal(t, "method", scope.Symbols[1].Name)
}
