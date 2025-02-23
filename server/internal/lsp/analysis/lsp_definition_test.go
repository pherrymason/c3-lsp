package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
	"testing"
)

func findSymbol(source string, pos lsp.Position) option.Option[*Symbol] {
	fileName := "app.c3"
	tree := getTree(source, fileName)
	symbolTable := BuildSymbolTable(tree, fileName)

	return FindSymbolAtPosition(pos, fileName, symbolTable, tree, source)
}

func TestFindsSymbol_Declaration_variable(t *testing.T) {
	t.Run("Find global variable declaration in same module", func(t *testing.T) {
		source := `int number = 0;
		fn void main(){number + 2;}`

		cursorPosition := lsp.Position{Line: 1, Column: 18}
		symbolOpt := findSymbol(source, cursorPosition)

		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "number", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(0, 0, 0, 15), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
	})

	t.Run("Find local variable declaration in the right scope", func(t *testing.T) {
		source := `int number = 0;
		fn void main(){
			float number = 2; 
			number + 2;
		}`

		cursorPosition := lsp.Position{Line: 3, Column: 4}
		symbolOpt := findSymbol(source, cursorPosition)
		symbol := symbolOpt.Get()

		assert.Equal(t, "number", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 20), symbol.NodeDecl.GetRange())
		assert.Equal(t, "float", symbol.TypeDef.Name)
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

		cursorPosition := lsp.Position{Line: 5, Column: 5}
		symbolOpt := findSymbol(source, cursorPosition)
		symbol := symbolOpt.Get()

		assert.Equal(t, "number", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(4, 4, 4, 20), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
	})

	t.Run("Find local variable declaration in function arguments", func(t *testing.T) {
		source := `
		char tick;
		fn void main(int tick){
			tick = tick + 3;
		}`

		cursorPosition := lsp.Position{Line: 3, Column: 4}
		symbolOpt := findSymbol(source, cursorPosition)
		symbol := symbolOpt.Get()

		assert.Equal(t, "tick", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(2, 15, 2, 23), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
	})

	t.Run("Find global exported variable declaration", func(t *testing.T) {
		source := `
		module foo_alt;
		char tick;
		
		module foo;
		char tick;
		
		module foo2;
		import foo;
		char fps = tick * 60;
		`

		cursorPosition := lsp.Position{Line: 9, Column: 14} // Cursor at char fps = t|ick * 60;
		symbolOpt := findSymbol(source, cursorPosition)
		symbol := symbolOpt.Get()

		assert.Equal(t, "tick", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(5, 2, 5, 12), symbol.NodeDecl.GetRange())
		assert.Equal(t, "char", symbol.TypeDef.Name)
		assert.Equal(t, NewModuleName("foo"), symbol.Module)
	})

	t.Run("Find global implicitly imported variable declaration", func(t *testing.T) {
		source := `
		module foo::bar;
		char tick;
		
		module foo;
		char fps = tick * 60;
		`

		cursorPosition := lsp.Position{Line: 5, Column: 14} // Cursor at char fps = t|ick * 60;
		symbolOpt := findSymbol(source, cursorPosition)
		symbol := symbolOpt.Get()

		assert.Equal(t, "tick", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(2, 2, 2, 12), symbol.NodeDecl.GetRange())
		assert.Equal(t, "char", symbol.TypeDef.Name)
		assert.Equal(t, NewModuleName("foo::bar"), symbol.Module)
	})

	t.Run("Find explicitly exported variable declared in different module, same file", func(t *testing.T) {
		source := `
		module foo_alt;
		char tick;
		
		module foo;
		char tick;
		
		module app;
		import foo;
		import foo_alt;		// Importing module with also a 'tick' defined in it after correct module will confuse the algorithm.
		
		char fps = foo::tick * 60;
		`

		cursorPosition := lsp.Position{Line: 11, Column: 19} // Cursor at char fps = foo::t|ick * 60;
		symbolOpt := findSymbol(source, cursorPosition)
		symbol := symbolOpt.Get()

		assert.Equal(t, "tick", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(5, 2, 5, 12), symbol.NodeDecl.GetRange())
		assert.Equal(t, "char", symbol.TypeDef.Name)
		assert.Equal(t, NewModuleName("foo"), symbol.Module)
	})

	t.Run("Find explicitly exported variable declared in different module, different file", func(t *testing.T) {
		source := `
		module foo_alt;
		char tick;
		
		module app;
		import foo;
		char fps = foo::tick * 60;
		`

		symbolTable := NewSymbolTable()

		fileName := "app.c3"
		tree := getTree(source, fileName)
		UpdateSymbolTable(symbolTable, tree, fileName)

		secondFileName := "foo.c3"
		source2 := `module foo;
		char tick;`
		tree2 := getTree(source2, secondFileName)
		UpdateSymbolTable(symbolTable, tree2, secondFileName)

		cursorPosition := lsp.Position{Line: 6, Column: 19} // Cursor at char fps = foo::t|ick * 60;
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		symbol := symbolOpt.Get()

		assert.Equal(t, "tick", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(1, 2, 1, 12), symbol.NodeDecl.GetRange())
		assert.Equal(t, "char", symbol.TypeDef.Name)
		assert.Equal(t, NewModuleName("foo"), symbol.Module)
	})
}

func TestFindsSymbol_Declaration_enum(t *testing.T) {
	t.Run("Find enum declaration in same module", func(t *testing.T) {
		source := `enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
		fn void main(){WindowStatus status;}`

		cursorPosition := lsp.Position{Line: 1, Column: 18}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "WindowStatus", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(0, 0, 0, 49), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.ENUM), symbol.Kind)
	})

	t.Run("Find enum member in same module selecting the right inferred type", func(t *testing.T) {
		// This is testing the ability of understanding that OPEN in `WindowStatus status = OPEN`
		// should be of type WindowStatus and should not suggest
		t.Skip("Not yet implemented")
		source := `
		const OPEN = 0; 		// To confuse algorithm
		const BACKGROUND = 1; 	// To confuse algorithm
		enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
		fn void main(){
			WindowStatus status = OPEN;
			WindowStatus back = WindowStatus.BACKGROUND;
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 5, Column: 26}
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "OPEN", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(3, 22, 3, 26), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)

		cursorPosition = lsp.Position{Line: 6, Column: 37}
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "BACKGROUND", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(3, 28, 3, 38), symbol.NodeDecl.GetRange())
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

		cursorPosition := lsp.Position{Line: 5, Column: 11}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "foo", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(2, 2, 2, 31), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})

	t.Run("Find enum declaration in explicitly imported module", func(t *testing.T) {
		source := `
		module foo;
		enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }

		module app;
		import foo;
		fn void main(){foo::WindowStatus status;}`

		cursorPosition := lsp.Position{Line: 6, Column: 23} // Cursor at foo::W|indowStatus status;
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "WindowStatus", symbol.Identifier)
		assert.Equal(t, "foo", symbol.Module.String())
		assert.Equal(t, lsp.NewRange(2, 2, 2, 51), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.ENUM), symbol.Kind)
	})

	t.Run("Find enum constant in explicitly imported module", func(t *testing.T) {
		source := `
		module foo;
		enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }

		module app;
		import foo;
		fn void main(){foo::WindowStatus status = foo::WindowStatus.BACKGROUND;}`

		cursorPosition := lsp.Position{Line: 6, Column: 62} // Cursor at foo::WindowStatus.B|ACKGROUND;
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "BACKGROUND", symbol.Identifier)
		assert.Equal(t, "foo", symbol.Module.String())
		assert.Equal(t, lsp.NewRange(2, 28, 2, 38), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.ENUM_VALUE), symbol.Kind)
	})

	t.Run("Should not find enumerator on enumerator", func(t *testing.T) {
		// TODO Because now we work with the ast, this is invalid and generates an invalid AST, meaning, from the AST we cannot determine MINIMIZED belongs to an ast.SelectorExpr and thus, searching algorithm does not goes the chaining branch.
		source := `
			enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status;
				status = WindowStatus.BACKGROUND.MINIMIZED;
			}`

		cursorPosition := lsp.Position{Line: 4, Column: 38} // Cursor is at `status = WindowStatus.BACKGROUND.M|INIMIZED`
		symbolOpt := findSymbol(source, cursorPosition)
		assert.False(t, symbolOpt.IsSome())
		/*
			position := buildPosition(4, 38) // Cursor is at `status = WindowStatus.BACKGROUND.M|INIMIZED`
			symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

			assert.True(t, symbolOption.IsNone(), "Element found")*/
	})

	t.Run("Should not find enumerator on enumerator variable", func(t *testing.T) {
		source := `
			enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status = WindowStatus.BACKGROUND;
				status = status.MINIMIZE|||D;
			}`

		source, position := parseBodyWithCursor(source)
		symbolOpt := findSymbol(source, position)
		assert.False(t, symbolOpt.IsSome())
	})

	t.Run("Should find local enumerator definition associated value without custom backing type", func(t *testing.T) {
		source := `enum WindowStatus : (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			fn void main() {
				int status = WindowStatus.BACKGROUND.c|||ounter;
			}`

		source, position := parseBodyWithCursor(source)
		symbolOpt := findSymbol(source, position)

		assert.True(t, symbolOpt.IsSome(), "Element not found")
		variable := symbolOpt.Get()
		assert.Equal(t, "counter", variable.Identifier)
		assert.Equal(t, "int", variable.TypeDef.Name)
	})

	t.Run("Should find associated value on enum instance variable", func(t *testing.T) {
		source := `enum WindowStatus : int (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			fn void main() {
				WindowStatus status = WindowStatus.BACKGROUND;
				int value = status.c|||ounter;
			}`

		source, position := parseBodyWithCursor(source)
		symbolOpt := findSymbol(source, position)

		assert.True(t, symbolOpt.IsSome(), "Element not found")
		variable := symbolOpt.Get()
		assert.Equal(t, "counter", variable.Identifier)
		assert.Equal(t, "int", variable.TypeDef.Name)
	})

	t.Run("Should find associated value on enum instance struct member", func(t *testing.T) {
		source := `enum WindowStatus : int (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			struct MyStruct { WindowStatus stat; }
			fn void main() {
				MyStruct wrapper = { WindowStatus.BACKGROUND };
				int value = wrapper.stat.c|||ounter;
			}`

		source, position := parseBodyWithCursor(source)
		symbolOpt := findSymbol(source, position)

		assert.True(t, symbolOpt.IsSome(), "Element not found")
		variable := symbolOpt.Get()
		assert.Equal(t, "counter", variable.Identifier)
		assert.Equal(t, "int", variable.TypeDef.Name)
	})

	t.Run("Should not find associated value on enum type", func(t *testing.T) {
		source := `enum WindowStatus : int (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			fn void main() {
				WindowStatus.c|||ounter;
			}`

		source, position := parseBodyWithCursor(source)
		symbolOpt := findSymbol(source, position)

		assert.False(t, symbolOpt.IsSome(), "Element should not be found")
	})
}

func TestFindsSymbol_Declaration_fault(t *testing.T) {
	t.Run("Find fault declaration in same module", func(t *testing.T) {
		source := `fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
		fn void main(){WindowError err;}`

		cursorPosition := lsp.Position{Line: 1, Column: 18}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "WindowError", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(0, 0, 0, 58), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FAULT), symbol.Kind)
	})

	t.Run("Find fault member in same module", func(t *testing.T) {
		source := `fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
		fn void main(){WindowError err = UNEXPECTED_ERROR;}`

		cursorPosition := lsp.Position{Line: 1, Column: 36}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "UNEXPECTED_ERROR", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(0, 20, 0, 36), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FAULT_CONSTANT), symbol.Kind)
	})

	t.Run("Find fault method in same module", func(t *testing.T) {
		source := `fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
		fn int foo() {} // To confuse algorithm
		fn void WindowError.foo() {}
		fn void main(){
			WindowError err;
			err.foo();
		}`

		cursorPosition := lsp.Position{Line: 5, Column: 8}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "foo", symbol.Identifier)
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

		cursorPosition := lsp.Position{Line: 4, Column: 4}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "Animal", symbol.Identifier)
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
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "life", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(1, 3, 1, 12), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)

		// -----------------------------------------------
		// Following is testing finding the symbol when cursor is in property that is not a builtin type
		// -----------------------------------------------
		cursorPosition = lsp.Position{Line: 7, Column: 8}
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "taxId", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 18), symbol.NodeDecl.GetRange())
		assert.Equal(t, "Taxonomy", symbol.TypeDef.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)

		//-----------------------------------------------
		// Following is testing finding the symbol when cursor is in Selector.Expr
		// -----------------------------------------------
		cursorPosition = lsp.Position{Line: 6, Column: 4} // Cursor at d|og.life
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)

		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "dog", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(5, 3, 5, 14), symbol.NodeDecl.GetRange())
		assert.Equal(t, "Animal", symbol.TypeDef.Name)
		assert.Equal(t, ast.Token(ast.VAR), symbol.Kind)
	})

	t.Run("Find struct method definition in same module", func(t *testing.T) {
		source := `struct Animal {int life;}
		fn void Animal.bark() {}
		fn void main(){
			Animal dog;
			dog.bark();
		}`

		cursorPosition := lsp.Position{Line: 4, Column: 8}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "bark", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
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

		cursorPosition := lsp.Position{Line: 10, Column: 14} // cursor at dog.being.l|ife = 3
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "life", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 12), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)

		// -----------------------------------------------
		// Following is testing finding the symbol when cursor is in Selector.Expr
		// -----------------------------------------------
		cursorPosition = lsp.Position{Line: 10, Column: 8} // Cursor is at dog.b|eing.life
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "being", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(6, 3, 6, 15), symbol.NodeDecl.GetRange())
		assert.Equal(t, "Being", symbol.TypeDef.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})

	t.Run("Find indirect struct field definition, located in an explicit different module", func(t *testing.T) {
		source := `
		module foo2;
		struct Alien {int life;}

		module foo;
		struct Alien {int life;}

		module app;
		import foo;
		import foo2;
		struct Animal { 
			foo::Alien xeno; // Alien is referenced by specifying module
		}
		fn void main(){
			Animal dog;
			dog.xeno.life = 10;
		}`

		cursorPosition := lsp.Position{Line: 15, Column: 13} // Cursor at `dog.xeno.l|ife = 10;`
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "life", symbol.Identifier)
		assert.Equal(t, NewModuleName("foo"), symbol.Module)
		assert.Equal(t, lsp.NewRange(5, 16, 5, 25), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
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

		cursorPosition := lsp.Position{Line: 11, Column: 15}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "length", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 14), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})

	t.Run("Find indirect struct field in a method chain in different explicit module", func(t *testing.T) {
		source := `
		module foo;
		struct Sound {
			int length;
		}

		module app;
		import foo;
		struct Animal { 
			String name;
			Being being;
		}
		fn foo::Sound Animal.bark() {}
		fn void main(){
			Animal dog;
			dog.bark().length = 3;
		}`

		cursorPosition := lsp.Position{Line: 15, Column: 15} // Cursor at dog.bark().l|ength = 3;
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "length", symbol.Identifier)
		assert.Equal(t, NewModuleName("foo"), symbol.Module)
		assert.Equal(t, lsp.NewRange(3, 3, 3, 14), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
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

		cursorPosition := lsp.Position{Line: 7, Column: 11}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "length", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 14), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})

	t.Run("Find indirect struct field in an explicitly imported function chain in same module", func(t *testing.T) {
		source := `
		module foo;
		struct Sound {
			int length;
		}
		fn Sound bark() {}

		module app;
		import foo;
		fn void main(){
			Animal dog;
			foo::bark().length = 3;
		}`

		cursorPosition := lsp.Position{Line: 11, Column: 16} // Cursor at `foo::bark().l|ength = 3;`
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "length", symbol.Identifier)
		assert.Equal(t, NewModuleName("foo"), symbol.Module)
		assert.Equal(t, lsp.NewRange(3, 3, 3, 14), symbol.NodeDecl.GetRange())
		assert.Equal(t, "int", symbol.TypeDef.Name)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})

	t.Run("Find struct method when referencing it with self", func(t *testing.T) {
		source := `
		struct Sound {
			int length;
		}
		fn void Sound.play(&self) {
			self.stop();
		}
		fn void Confusion.stop() {}
		fn void Sound.stop(){}`

		cursorPosition := lsp.Position{Line: 5, Column: 9}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "stop", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(8, 2, 8, 24), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})

	t.Run("Find struct property when referencing it with self", func(t *testing.T) {
		source := `
		module foo;
		struct Sound {
			bool playing;
		}

		module app;
		import foo;
		fn void foo::Sound.play(&self) {
			self.playing = true
		}`

		cursorPosition := lsp.Position{Line: 9, Column: 9} // Cursor at `self.p|laying = true;`
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "playing", symbol.Identifier)
		assert.Equal(t, NewModuleName("foo"), symbol.Module)
		assert.Equal(t, lsp.NewRange(3, 3, 3, 16), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
	})

	t.Run("Find struct member that is inlined", func(t *testing.T) {
		source := `
		struct Foo {
		  int a;
		  int b;
		}
		struct Bar {
			inline Foo sub;
		}
		fn void main() {
			Bar obj;
			obj.a = 3;
		}`

		cursorPosition := lsp.Position{Line: 10, Column: 7} // Cursor at obj.|a = 3;
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome(), "Symbol not found")
	})

	t.Run("Find explicitly imported struct member that is inlined", func(t *testing.T) {
		source := `
		module foo;
		struct Foo {
		  int a;
		  int b;
		}
		module app;
		import foo;
		struct Bar {
			inline foo::Foo sub;
		}
		fn void main() {
			Bar obj;
			obj.a = 3;
		}`

		cursorPosition := lsp.Position{Line: 13, Column: 7} // Cursor at obj.|a = 3;
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome(), "Symbol not found")
		assert.Equal(t, "foo", symbolOpt.Get().Module.String())
	})

	t.Run("Find struct member that is anonymous sub struct", func(t *testing.T) {
		source := `
		struct Bar {
			struct data {
			  int a;
			}
		}
		fn void main() {
			Bar obj;
			obj.data.a = 3;
		}`

		cursorPosition := lsp.Position{Line: 8, Column: 8} // Cursor at `obj.d|ata.a = 3`
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome(), "Symbol not found")
		assert.Equal(t, "data", symbolOpt.Get().Identifier)
	})

	t.Run("Find struct member that is inside anonymous sub struct", func(t *testing.T) {
		source := `
		struct Bar {
			struct data {
			  int a;
			}
		}
		fn void main() {
			Bar obj;
			obj.data.a = 3;
		}`

		cursorPosition := lsp.Position{Line: 8, Column: 12} // Cursor at `obj.data.|a = 3`
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome(), "Symbol not found")
		assert.Equal(t, "a", symbolOpt.Get().Identifier)
		assert.Equal(t, lsp.NewRange(3, 5, 3, 11), symbolOpt.Get().Range)
	})

	t.Run("Find struct method when it is inlined", func(t *testing.T) {
		source := `
		struct Foo {}
		fn void Foo.jump(&self) {}
		fn void Foo.overloaded(&self) {}
		struct Bar {
			inline Foo sub;
		}
		fn void Bar.overloaded(&self) {}
		fn void Bar.init(&self) {
    		self.sub.jump();
			self.jump();
			self.overloaded();
			self.sub.overloaded();
		}`

		fileName := "app.c3"
		tree := getTree(source, fileName)
		symbolTable := BuildSymbolTable(tree, fileName)

		cursorPosition := lsp.Position{Line: 9, Column: 16} // Cursor at self.sub.j|ump();
		symbolOpt := FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "jump", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 2, 2, 28), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)

		// Testing cursor at `self.j|ump();` should find `fn void Foo.jump(&self)`
		cursorPosition = lsp.Position{Line: 10, Column: 9} //
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "jump", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(2, 2, 2, 28), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)

		// Testing cursor at `self.o|verloaded();` should find `fn void Bar.overloaded(&self)`
		cursorPosition = lsp.Position{Line: 11, Column: 9} //
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "overloaded", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(7, 2, 7, 34), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)

		// Testing cursor at `self.sub.o|verloaded();` should find `fn void For.jump(&self)`
		cursorPosition = lsp.Position{Line: 12, Column: 13}
		symbolOpt = FindSymbolAtPosition(cursorPosition, fileName, symbolTable, tree, source)
		assert.True(t, symbolOpt.IsSome())
		symbol = symbolOpt.Get()
		assert.Equal(t, "overloaded", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(3, 2, 3, 34), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})

	t.Run("Find interface struct is implementing", func(t *testing.T) {
		source := `
		interface Animal{fn void run();}
		struct Dog (Animal) {
			bool a;
		}`

		cursorPosition := lsp.Position{Line: 2, Column: 15}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "Animal", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(1, 2, 1, 34), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.INTERFACE), symbol.Kind)
	})
}

func TestFindsSymbol_Declaration_def(t *testing.T) {
	t.Run("Find def declaration in same module", func(t *testing.T) {
		source := `def Kilo = int;
		Kilo value = 3;`

		cursorPosition := lsp.Position{Line: 1, Column: 3}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "Kilo", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(0, 0, 0, 15), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.DEF), symbol.Kind)
	})

	t.Run("Find follow chained on def with function", func(t *testing.T) {
		source := `
		struct MyStruct {
			float number;
		}
		fn MyStruct a() {}
		def func = a;

		fn void main(){ func.n|||umber; }
		`

		source, position := parseBodyWithCursor(source)
		symbolOpt := findSymbol(source, position)

		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "number", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(2, 3, 2, 16), symbol.Range)
		assert.Equal(t, ast.Token(ast.FIELD), symbol.Kind)
		assert.Equal(t, "float", symbol.TypeDef.Name)
	})
}

func TestFindsSymbol_Declaration_function(t *testing.T) {
	t.Run("Find local function definition", func(t *testing.T) {
		source := `
	fn void run(int tick) {}
	fn void main() {
		run(3);
	}`

		cursorPosition := lsp.Position{Line: 3, Column: 3}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "run", symbol.Identifier)
		assert.Equal(t, lsp.NewRange(1, 1, 1, 25), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})

	t.Run("Should find function definition without body", func(t *testing.T) {
		source := `
	fn void init_window(int width, int height, char* title) @extern("InitWindow");
	fn void main() {
		init_window(200, 200, "hello");
	}`

		cursorPosition := lsp.Position{Line: 3, Column: 3}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "init_window", symbol.Identifier)
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

		cursorPosition := lsp.Position{Line: 7, Column: 4}
		symbolOpt := findSymbol(source, cursorPosition)
		assert.True(t, symbolOpt.IsSome())
		symbol := symbolOpt.Get()
		assert.Equal(t, "bark", symbol.Identifier)
		assert.Equal(t, NewModuleName("app"), symbol.Module)
		assert.Equal(t, lsp.NewRange(4, 2, 4, 20), symbol.NodeDecl.GetRange())
		assert.Equal(t, ast.Token(ast.FUNCTION), symbol.Kind)
	})
}
