package search

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"

	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func readC3File(filePath string) string {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("Error reading file: %v\n", err))
	}

	// Convierte el slice de bytes a un string
	return string(contentBytes)
}

/*
func initTestEnv() (*project_state.ProjectState, map[string]document.Document) {
	parser := createParser()
	language := project_state.NewProjectState(commonlog.MockLogger{}, option.Some("dummy"), false)

	documents := installDocuments(&language, &parser)

	return &language, documents
}*/

func buildPosition(line uint, character uint) idx.Position {
	return idx.Position{Line: line - 1, Character: character}
}

var debugger = NewFindDebugger(true)

func TestLanguage_findClosestSymbolDeclaration_ignores_keywords(t *testing.T) {
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
	parser := createParser()
	logger := &MockLogger{
		tracker: make(map[string][]string),
	}
	search := NewSearch(logger, true)
	state := project_state.NewProjectState(logger, option.Some("dummy"), true)

	doc := document.NewDocument("x", "module foo;")
	state.RefreshDocumentIdentifiers(&doc, &parser)

	doc = document.NewDocument("z", "module bar;import foo;")
	state.RefreshDocumentIdentifiers(&doc, &parser)

	for _, tt := range cases {
		t.Run(tt.source, func(t *testing.T) {
			logger.tracker = make(map[string][]string)
			doc := document.NewDocument("y", "module foo;"+tt.source)
			state.RefreshDocumentIdentifiers(&doc, &parser)
			position := buildPosition(1, 12) // Cursor at BA|R_WEIGHT
			symbol := search.FindSymbolDeclarationInWorkspace(doc.URI, position, &state)

			assert.True(t, symbol.IsNone(), fmt.Sprintf("\"%s\" Symbol should not be found", tt.source))
			assert.Equal(t, 1, len(logger.tracker["debug"]))
			assert.Equal(t, "| Ignore because C3 keyword", logger.tracker["debug"][0])
		})
	}
}

func TestLanguage_findClosestSymbolDeclaration_variables(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Find global variable definition, with cursor in usage", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`int number = 0;
			fn void newNumber(){
				int result = number + 10;
			}`,
		)

		position := buildPosition(3, 18) // Cursor at `n|umber`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "number", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})

	t.Run("Find local variable definition, with cursor in same declaration", func(t *testing.T) {
		state.registerDoc(
			"number.c3",
			`fn void newNumber(){
				int number;
			}`,
		)

		position := buildPosition(2, 9) // Cursor at `n|umber`

		symbolOption := search.FindSymbolDeclarationInWorkspace("number.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "number", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})

	t.Run("Find local variable definition from usage", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fn Emu newEmulator(){
				Emu emulator;
				emulator = 2;
			}`,
		)
		position := buildPosition(3, 5) // Cursor at `e|mulator`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "emulator", symbol.GetName())
		assert.Equal(t, "Emu", variable.GetType().String())
	})

	t.Run("Should find the right element when there is a different element with the same name up in the scope", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`char ambiguousVariable = 'C';
			fn void main() {
				int ambiguousVariable = 3;
			}`,
		)
		position := buildPosition(3, 9) // Cursor a|mbiguousVariable

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "ambiguousVariable", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})

	t.Run("Find local variable definition in function arguments", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fn void run(int tick) {
				tick = tick + 3;
			}`,
		)
		position := buildPosition(2, 5) // Cursor at `t|ick = tick + 3;`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.True(t, symbolOption.IsSome(), "Element not found")

		variable := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, "tick", symbolOption.Get().GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})
}

// Tests related to structs:
func TestLanguage_findClosestSymbolDeclaration_structs(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Should find struct declaration in variable declaration", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`struct Emu {
				bool a;
			}
			fn void main() {
				Emu emulator;
			}`,
		)

		position := buildPosition(5, 5) // Cursor at `E|mu emulator`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		_struct := symbol.(*idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Should find struct declaration in function return type", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`struct Emu {
				bool a;
			}
			fn Emu main() {
				Emu emulator;
			}`,
		)
		position := buildPosition(4, 7) // Cursor at `fn E|mu main() {`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		_struct := symbol.(*idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Should find interface struct is implementing", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`interface EmulatorConsole
			{
				fn void run();
			}
			struct Emu (EmulatorConsole) {
				bool a;
			}`,
		)
		position := buildPosition(5, 15) // Cursor is at struct Emu (E|mulatorConsole) {

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_interface, ok := symbolOption.Get().(*idx.Interface)
		assert.True(t, ok, "Element found should be an Interface")
		assert.Equal(t, "EmulatorConsole", _interface.GetName())
	})

	// TODO test finding interface method
}

func TestLanguage_findClosestSymbolDeclaration_enums(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Find local enum variable definition when cursor is in enum declaration", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status;
			}`,
		)
		position := buildPosition(3, 19)

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		variable := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, "status", symbolOption.Get().GetName())
		assert.Equal(t, "WindowStatus", variable.GetType().String())
	})

	t.Run("Should find enum definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status;
			}`,
		)
		position := buildPosition(3, 5) // Cursor is at `W|indowStatus status;`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		enum := symbolOption.Get().(*idx.Enum)
		assert.Equal(t, "WindowStatus", enum.GetName())
	})

	t.Run("Should find local explicit enumerator definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status;
				status = WindowStatus.BACKGROUND;
			}`,
		)
		position := buildPosition(4, 27) // Cursor is at `status = WindowStatus.B|ACKGROUND`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.Enumerator)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an enumerator, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "BACKGROUND", symbolOption.Get().GetName())
	})

	t.Run("Should not find enumerator on enumerator", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status;
				status = WindowStatus.BACKGROUND.MINIMIZED;
			}`,
		)
		position := buildPosition(4, 38) // Cursor is at `status = WindowStatus.BACKGROUND.M|INIMIZED`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.True(t, symbolOption.IsNone(), "Element found")
	})

	t.Run("Should not find enumerator on enumerator variable", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status = WindowStatus.BACKGROUND;
				status = status.MINIMIZED;
			}`,
		)
		position := buildPosition(4, 21) // Cursor is at `status = status.M|INIMIZED`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.True(t, symbolOption.IsNone(), "Element found")
	})

	t.Run("Should find local enumerator definition associated value", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`enum WindowStatus : int (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			fn void main() {
				int status = WindowStatus.BACKGROUND.counter;
			}`,
		)
		position := buildPosition(7, 42) // Cursor is at `status = WindowStatus.BACKGROUND.c|ounter`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		variable, ok := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an associated value, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "counter", variable.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should find local enumerator definition associated value without custom backing type", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`enum WindowStatus : (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			fn void main() {
				int status = WindowStatus.BACKGROUND.counter;
			}`,
		)
		position := buildPosition(7, 42) // Cursor is at `status = WindowStatus.BACKGROUND.c|ounter`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		variable, ok := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an associated value, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "counter", variable.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should find local implicit enumerator definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status;
				status = BACKGROUND;
			}`,
		)
		position := buildPosition(4, 13) // Cursor is at `status = B|ACKGROUND`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.Enumerator)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an enumerator, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "BACKGROUND", symbolOption.Get().GetName())
	})

	t.Run("Should find enum method definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn bool WindowStatus.isOpen(){}

			fn void main() {
				WindowStatus val = OPEN;
				val.isOpen();
			}
			`,
		)
		position := buildPosition(6, 10) // Cursor is at `e.is|Open()`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.Function)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not a method, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "WindowStatus.isOpen", symbolOption.Get().GetName())
	})
}

func TestLanguage_findClosestSymbolDeclaration_faults(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Find local fault definition in type declaration", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				WindowError error = WindowError.SOMETHING_HAPPENED;
				error = UNEXPECTED_ERROR;
			}`,
		)
		position := buildPosition(3, 5) // Cursor at `W|indowError error =`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Fault not found")

		fault := symbolOption.Get().(*idx.Fault)
		assert.Equal(t, "WindowError", fault.GetName())
	})

	t.Run("Find local fault variable definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				WindowError error = WindowError.SOMETHING_HAPPENED;
				error = UNEXPECTED_ERROR;
			}`,
		)
		position := buildPosition(4, 5) // Cursor at `e|rror = UNEXPECTED_ERROR``

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Fault not found")

		fault := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, "error", fault.GetName())
	})

	t.Run("Should find implicit fault constant definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				WindowError error = WindowError.SOMETHING_HAPPENED;
				error = UNEXPECTED_ERROR;
			}`,
		)
		position := buildPosition(4, 13) // Cursor at `error = U|NEXPECTED_ERROR;`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.FaultConstant)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an fault constant, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "UNEXPECTED_ERROR", symbolOption.Get().GetName())
	})

	t.Run("Should not find fault constant on fault constant", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				WindowError.SOMETHING_HAPPENED.UNEXPECTED_ERROR;
			}`,
		)
		position := buildPosition(3, 36) // Cursor at `WindowError.SOMETHING_HAPPENED.U|NEXPECTED_ERROR;`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.True(t, symbolOption.IsNone(), "Element found")
	})

	t.Run("Should not find fault constant on fault instance", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				WindowError error = WindowError.SOMETHING_HAPPENED;
				error.UNEXPECTED_ERROR;
			}`,
		)
		position := buildPosition(4, 11) // Cursor at `error.U|NEXPECTED_ERROR;`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.True(t, symbolOption.IsNone(), "Element found")
	})

	t.Run("Should find fault method definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn bool WindowError.isBad(){}

			fn void main() {
				WindowError val = UNEXPECTED_ERROR;
				val.isBad();
			}
			`,
		)
		position := buildPosition(6, 10) // Cursor is at `e.is|Bad()`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Method not found")
		_, ok := symbolOption.Get().(*idx.Function)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not a method, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "WindowError.isBad", symbolOption.Get().GetName())
	})
}

func TestLanguage_findClosestSymbolDeclaration_def(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Find local definition definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`def Kilo = int;
			Kilo value = 3;`,
		)
		position := buildPosition(2, 4) // Cursor at `K|ilo value = 3`

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		assert.Equal(t, "Kilo", symbolOption.Get().GetName())
	})
}

func TestLanguage_findClosestSymbolDeclaration_functions(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Find local function definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fn void run(int tick) {
			}
			fn void main() {
				run(3);
			}`,
		)
		position := buildPosition(4, 5) // Cursor at r|un(3);

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "run", fun.GetName())
		assert.Equal(t, "void", fun.GetReturnType().GetName())
	})

	t.Run("Should not confuse function with virtual root scope function", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fn void main() {
				run(3);
			}
			fn void call(){ main(); }`,
		)
		position := buildPosition(4, 20) // Cursor at m|ain();

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "main", fun.GetName())
		assert.Equal(t, idx.FunctionType(idx.UserDefined), fun.FunctionType())
	})

	t.Run("Should find function definition without body", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fn void init_window(int width, int height, char* title) @extern("InitWindow");

			init_window(200, 200, "hello");
			`,
		)

		position := buildPosition(3, 4) // Cursor at i|nit_window(200, 200, "hello")

		symbolOption := search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "init_window", fun.GetName())
		assert.Equal(t, idx.FunctionType(idx.UserDefined), fun.FunctionType())
	})
}
