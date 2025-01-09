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

func TestFindSymbol_ignores_language_keywords(t *testing.T) {
	cases := []struct {
		source string
	}{
		{"void"}, {"bool"}, {"char"}, {"double"},
		{"float"}, {"float16"}, {"int128"}, {"ichar"},
		{"int"}, {"iptr"}, {"isz"}, {"long"},
		{"short"}, {"uint128"}, {"uint"}, {"ulong"},
		{"uptr"}, {"ushort"}, {"usz"}, {"float128"},
		{"any"}, {"anyfault"}, {"typeid"}, {"assert"},
		{"asm"}, {"bitstruct"}, {"break"}, {"case"},
		{"catch"}, {"const"}, {"continue"}, {"def"},
		{"default"}, {"defer"}, {"distinct"}, {"do"},
		{"else"}, {"enum"}, {"extern"}, {"false"},
		{"fault"}, {"for"}, {"foreach"}, {"foreach_r"},
		{"fn"}, {"tlocal"}, {"if"}, {"inline"},
		{"import"}, {"macro"}, {"module"}, {"nextcase"},
		{"null"}, {"return"}, {"static"}, {"struct"},
		{"switch"}, {"true"}, {"try"}, {"union"},
		{"var"}, {"while"},
		{"$alignof"}, {"$assert"}, {"$case"}, {"$default"},
		{"$defined"}, {"$echo"}, {"$embed"}, {"$exec"},
		{"$else"}, {"$endfor"}, {"$endforeach"}, {"$endif"},
		{"$endswitch"}, {"$eval"}, {"$evaltype"}, {"$error"},
		{"$extnameof"}, {"$for"}, {"$foreach"}, {"$if"},
		{"$include"}, {"$nameof"}, {"$offsetof"}, {"$qnameof"},
		{"$sizeof"}, {"$stringify"}, {"$switch"}, {"$typefrom"},
		{"$typeof"}, {"$vacount"}, {"$vatype"}, {"$vaconst"},
		{"$varef"}, {"$vaarg"}, {"$vaexpr"}, {"$vasplat"},
	}

	for _, tt := range cases {
		t.Run(tt.source, func(t *testing.T) {
			fileName := tt.source
			tree := getTree("module foo;"+tt.source, fileName)
			symbolTable := BuildSymbolTable(tree, "")

			cursorPosition := lsp.Position{Line: 0, Column: 12}
			symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)

			assert.True(t, symbolOpt.IsNone())
		})
	}
}

func TestFindsSymbol_Declaration_variable(t *testing.T) {
	t.Run("Find global variable declaration in same module", func(t *testing.T) {
		source := `int number = 0;
		fn void main(){number + 2;}`

		tree := getTree(source, "app.c3")
		symbolTable := BuildSymbolTable(tree, "")

		cursorPosition := lsp.Position{Line: 1, Column: 18}
		symbolOpt := FindSymbolAtPosition(cursorPosition, "app.c3", symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "number", symbol.Name)
		assert.Equal(t, lsp.NewRange(0, 0, 0, 15), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.Type.Name)
	})

	t.Run("Find local variable declaration in the right scope", func(t *testing.T) {
		source := `int number = 0;
		fn void main(){
			float number = 2; 
			number + 2;
		}`

		tree := getTree(source, "app.c3")
		symbolTable := BuildSymbolTable(tree, "")

		cursorPosition := lsp.Position{Line: 3, Column: 4}
		symbolOpt := FindSymbolAtPosition(cursorPosition, "app.c3", symbolTable, tree)
		symbol := symbolOpt.Get()

		assert.Equal(t, "number", symbol.Name)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 20), symbol.NodeDecl.GetRange())
		assert.Equal(t, "float", symbol.Type.Name)
	})

	t.Run("Find local variable declaration in the right expression block", func(t *testing.T) {
		source := `
		fn void main(int number){
			float number = 2; 
		    {|
				int number = 10;
				number + 12;
			|}
		}`

		tree := getTree(source, "app.c3")
		symbolTable := BuildSymbolTable(tree, "")

		cursorPosition := lsp.Position{Line: 5, Column: 5}
		symbolOpt := FindSymbolAtPosition(cursorPosition, "app.c3", symbolTable, tree)
		symbol := symbolOpt.Get()

		assert.Equal(t, "number", symbol.Name)
		assert.Equal(t, lsp.NewRange(4, 4, 4, 20), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.Type.Name)
	})

	t.Run("Find local variable declaration in function arguments", func(t *testing.T) {
		source := `
		char tick;
		fn void main(int tick){
			tick = tick + 3;
		}`

		tree := getTree(source, "app.c3")
		symbolTable := BuildSymbolTable(tree, "")

		cursorPosition := lsp.Position{Line: 3, Column: 4}
		symbolOpt := FindSymbolAtPosition(cursorPosition, "app.c3", symbolTable, tree)
		symbol := symbolOpt.Get()

		assert.Equal(t, "tick", symbol.Name)
		assert.Equal(t, lsp.NewRange(2, 15, 2, 23), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.Type.Name)
	})

	t.Run("Find global exported variable declaration", func(t *testing.T) {
		source := `module foo;
		char tick;
		module foo2;
		import foo;
		char fps = tick * 60;
		`

		tree := getTree(source, "app.c3")
		symbolTable := BuildSymbolTable(tree, "")

		cursorPosition := lsp.Position{Line: 4, Column: 14}
		symbolOpt := FindSymbolAtPosition(cursorPosition, "app.c3", symbolTable, tree)
		symbol := symbolOpt.Get()

		assert.Equal(t, "tick", symbol.Name)
		assert.Equal(t, lsp.NewRange(2, 15, 2, 23), symbol.NodeDecl.GetRange())
		assert.Equal(t, "char", symbol.Type.Name)
		assert.Equal(t, "foo", symbol.Module)
	})
}
