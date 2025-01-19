package analysis

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func getTree(source string, fileName string) *ast.File {
	astConverter := factory.NewASTConverter()
	tree := astConverter.ConvertToAST(factory.GetCST(source), source, fileName)

	return tree
}

func TestFindSymbol_ignores_language_keywords(t *testing.T) {
	t.Skip("Need analyzer to know context to complement it with a list of blacklist")
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

			assert.True(t, symbolOpt.IsNone(), fmt.Sprintf("Found symbol for keyword %s", tt.source))
		})
	}
}

func TestFindsSymbol_Declaration_variable(t *testing.T) {
	t.Run("Find global variable declaration in same module", func(t *testing.T) {
		source := `int number = 0;
		fn void main(){number + 2;}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 1, Column: 18}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
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

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 3, Column: 4}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
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

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 5, Column: 5}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
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

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 3, Column: 4}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		symbol := symbolOpt.Get()

		assert.Equal(t, "tick", symbol.Name)
		assert.Equal(t, lsp.NewRange(2, 15, 2, 23), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.Type.Name)
	})

	t.Run("Find global exported variable declaration", func(t *testing.T) {
		t.Skip()
		source := `module foo;
		char tick;
		module foo2;
		import foo;
		char fps = tick * 60;
		`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 4, Column: 14}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		symbol := symbolOpt.Get()

		assert.Equal(t, "tick", symbol.Name)
		assert.Equal(t, lsp.NewRange(2, 15, 2, 23), symbol.NodeDecl.GetRange())
		assert.Equal(t, "char", symbol.Type.Name)
		assert.Equal(t, "foo", symbol.Module)
	})
}

func TestFindsSymbol_Declaration_enum(t *testing.T) {
	t.Run("Find enum declaration in same module", func(t *testing.T) {
		source := `enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
		fn void main(){WindowStatus status;}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 1, Column: 18}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "WindowStatus", symbol.Name)
		assert.Equal(t, lsp.NewRange(0, 0, 0, 49), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.ENUM), symbol.Kind)
	})

	t.Run("Find enum member in same module", func(t *testing.T) {
		source := `enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
		const OPEN = 0; 		// To confuse algorithm
		const BACKGROUND = 1; 	// To confuse algorithm
		fn void main(){
			WindowStatus status = OPEN;
			WindowStatus back = WindowStatus.BACKGROUND;
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 4, Column: 26}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "OPEN", symbol.Name)
		assert.Equal(t, lsp.NewRange(0, 20, 0, 24), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)

		cursorPosition = lsp.Position{Line: 5, Column: 37}
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "BACKGROUND", symbol.Name)
		assert.Equal(t, lsp.NewRange(0, 26, 0, 36), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})

	t.Run("Find enum method in same module", func(t *testing.T) {
		source := `enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
		fn int foo() {} // To confuse algorithm
		fn void WindowStatus.foo() {}
		fn void main(){
			WindowStatus status;
			status.foo();
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 5, Column: 11}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "foo", symbol.Name)
		assert.Equal(t, lsp.NewRange(2, 2, 2, 31), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})
}

func TestFindsSymbol_Declaration_fault(t *testing.T) {
	t.Run("Find fault declaration in same module", func(t *testing.T) {
		source := `fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
		fn void main(){WindowError err;}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 1, Column: 18}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "WindowError", symbol.Name)
		assert.Equal(t, lsp.NewRange(0, 0, 0, 58), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FAULT), symbol.Kind)
	})

	t.Run("Find fault member in same module", func(t *testing.T) {
		source := `fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
		fn void main(){WindowError err = UNEXPECTED_ERROR;}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 1, Column: 36}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "UNEXPECTED_ERROR", symbol.Name)
		assert.Equal(t, lsp.NewRange(0, 20, 0, 36), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})

	t.Run("Find fault method in same module", func(t *testing.T) {
		source := `fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
		fn int foo() {} // To confuse algorithm
		fn void WindowError.foo() {}
		fn void main(){
			WindowError err;
			err.foo();
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 5, Column: 8}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "foo", symbol.Name)
		assert.Equal(t, lsp.NewRange(2, 2, 2, 30), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})
}

func TestFindsSymbol_Declaration_struct(t *testing.T) {
	t.Run("Find struct declaration in same module", func(t *testing.T) {
		source := `struct Animal { 
			int life;
		}
		fn void main(){
			Animal dog;
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 4, Column: 4}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "Animal", symbol.Name)
		assert.Equal(t, lsp.NewRange(0, 0, 2, 3), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.STRUCT), symbol.Kind)
	})

	t.Run("Find struct field definition in same module", func(t *testing.T) {
		source := `struct Animal { 
			int life;
			Taxonomy taxId;
		}
		fn void main(){
			Animal dog;
			dog.life = 3;
			dog.taxId = 1;
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 6, Column: 8}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "life", symbol.Name)
		assert.Equal(t, ModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(1, 3, 1, 12), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.Type.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)

		// -----------------------------------------------
		// Following is testing finding the symbol when cursor is in property that is not a builtin type
		// -----------------------------------------------
		cursorPosition = lsp.Position{Line: 7, Column: 8}
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "taxId", symbol.Name)
		assert.Equal(t, ModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 18), symbol.NodeDecl.GetRange())
		assert.Equal(t, "Taxonomy", symbol.Type.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)

		//-----------------------------------------------
		// Following is testing finding the symbol when cursor is in Selector.Expr
		// -----------------------------------------------
		cursorPosition = lsp.Position{Line: 6, Column: 4} // Cursor at d|og.life
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)

		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "dog", symbol.Name)
		assert.Equal(t, ModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(5, 3, 5, 14), symbol.NodeDecl.GetRange())
		assert.Equal(t, "Animal", symbol.Type.Name)
		assert.Equal(t, ast.Token(ast.VAR), symbol.Kind)
	})

	t.Run("Find struct method definition in same module", func(t *testing.T) {
		source := `struct Animal {int life;}
		fn void Animal.bark() {}
		fn void main(){
			Animal dog;
			dog.bark();
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 4, Column: 8}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "bark", symbol.Name)
		assert.Equal(t, ModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(1, 2, 1, 26), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})

	t.Run("Find indirect struct field definition in same module", func(t *testing.T) {
		source := `
		struct Being {
			int life;	
		}
		struct Animal { 
			String name;
			Being being;
		}
		fn void main(){
			Animal dog;
			dog.being.life = 3;
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 10, Column: 14}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "life", symbol.Name)
		assert.Equal(t, ModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 12), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.Type.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)

		// -----------------------------------------------
		// Following is testing finding the symbol when cursor is in Selector.Expr
		// -----------------------------------------------
		cursorPosition = lsp.Position{Line: 10, Column: 8} // Cursor is at dog.b|eing.life
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "being", symbol.Name)
		assert.Equal(t, ModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(6, 3, 6, 15), symbol.NodeDecl.GetRange())
		assert.Equal(t, "Being", symbol.Type.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})

	t.Run("Find indirect struct field in a method chain in same module", func(t *testing.T) {
		source := `
		struct Sound {
			int length;
		}
		struct Animal { 
			String name;
			Being being;
		}
		fn Sound Animal.bark() {}
		fn void main(){
			Animal dog;
			dog.bark().length = 3;
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 11, Column: 15}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "length", symbol.Name)
		assert.Equal(t, ModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 14), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.Type.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})

	t.Run("Find indirect struct field in a function chain in same module", func(t *testing.T) {
		source := `
		struct Sound {
			int length;
		}
		fn Sound bark() {}
		fn void main(){
			Animal dog;
			bark().length = 3;
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 7, Column: 11}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "length", symbol.Name)
		assert.Equal(t, ModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 14), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.Type.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})
}

func TestFindsSymbol_Declaration_def(t *testing.T) {
	t.Run("Find enum declaration in same module", func(t *testing.T) {
		source := `def Kilo = int;
		Kilo value = 3;`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 1, Column: 3}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "Kilo", symbol.Name)
		assert.Equal(t, lsp.NewRange(0, 0, 0, 15), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.DEF), symbol.Kind)
	})
}

func TestFindsSymbol_Declaration_function(t *testing.T) {
	t.Run("Find local function definition", func(t *testing.T) {
		source := `
	fn void run(int tick) {}
	fn void main() {
		run(3);
	}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 3, Column: 3}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "run", symbol.Name)
		assert.Equal(t, lsp.NewRange(1, 1, 1, 25), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})

	t.Run("Should find function definition without body", func(t *testing.T) {
		source := `
	fn void init_window(int width, int height, char* title) @extern("InitWindow");
	init_window(200, 200, "hello");`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 2, Column: 2}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "init_window", symbol.Name)
		assert.Equal(t, lsp.NewRange(1, 1, 1, 79), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})

	// This test is interesting because we are playing with the cursor placed in SelectorExpr.X
	t.Run("Find function returning a struct", func(t *testing.T) {
		source := `
		struct Sound {
			int length;
		}
		fn Sound bark() {}
		fn void main(){
			Animal dog;
			bark().length = 3;
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 7, Column: 4}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "bark", symbol.Name)
		assert.Equal(t, ModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(4, 2, 4, 20), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})
}
