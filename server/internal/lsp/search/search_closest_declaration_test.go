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

func SearchUnderCursor_ClosestDecl(body string, optionalState ...TestState) option.Option[idx.Indexable] {
	state := NewTestState()
	search := NewSearchWithoutLog()

	if len(optionalState) > 0 {
		state = optionalState[0]
	}

	cursorlessBody, position := parseBodyWithCursor(body)
	state.registerDoc(
		"app.c3",
		cursorlessBody,
	)

	return search.FindSymbolDeclarationInWorkspace("app.c3", position, &state.state)
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
	t.Run("Find global variable definition, with cursor in usage", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`int number = 0;
			fn void newNumber(){
				int result = n|||umber + 10;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "number", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})

	t.Run("Find local variable definition, with cursor in same declaration", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fn void newNumber(){
				int n|||umber;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "number", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})

	t.Run("Find local variable definition from usage", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fn Emu newEmulator(){
				Emu emulator;
				e|||mulator = 2;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "emulator", symbol.GetName())
		assert.Equal(t, "Emu", variable.GetType().String())
	})

	t.Run("Should find the right element when there is a different element with the same name up in the scope", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`char ambiguousVariable = 'C';
			fn void main() {
				int a|||mbiguousVariable = 3;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "ambiguousVariable", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})

	t.Run("Find local variable definition in function arguments", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fn void run(int tick) {
				t|||ick = tick + 3;
			}`,
		)

		assert.True(t, symbolOption.IsSome(), "Element not found")

		variable := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, "tick", symbolOption.Get().GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})
}

// Tests related to structs:
func TestLanguage_findClosestSymbolDeclaration_structs(t *testing.T) {
	t.Run("Should find struct declaration in variable declaration", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`struct Emu {
				bool a;
			}
			fn void main() {
				E|||mu emulator;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		_struct := symbol.(*idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Should find struct declaration in function return type", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`struct Emu {
				bool a;
			}
			fn E|||mu main() {
				Emu emulator;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		_struct := symbol.(*idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Should find interface struct is implementing", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`interface EmulatorConsole
			{
				fn void run();
			}
			struct Emu (|||EmulatorConsole) {
				bool a;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_interface, ok := symbolOption.Get().(*idx.Interface)
		assert.True(t, ok, "Element found should be an Interface")
		assert.Equal(t, "EmulatorConsole", _interface.GetName())
	})

	// TODO test finding interface method
}

func TestLanguage_findClosestSymbolDeclaration_enums(t *testing.T) {
	t.Run("Find local enum variable definition when cursor is in enum declaration", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus st|||atus;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		variable := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, "status", symbolOption.Get().GetName())
		assert.Equal(t, "WindowStatus", variable.GetType().String())
	})

	t.Run("Should find enum definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				W|||indowStatus status;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		enum := symbolOption.Get().(*idx.Enum)
		assert.Equal(t, "WindowStatus", enum.GetName())
	})

	t.Run("Should find local explicit enumerator definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status;
				status = WindowStatus.B|||ACKGROUND;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.Enumerator)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an enumerator, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "BACKGROUND", symbolOption.Get().GetName())
	})

	t.Run("Should not find enumerator on enumerator", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status;
				status = WindowStatus.BACKGROUND.M|||INIMIZED;
			}`,
		)

		assert.True(t, symbolOption.IsNone(), "Element found")
	})

	t.Run("Should not find enumerator on enumerator variable", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status = WindowStatus.BACKGROUND;
				status = status.M|||INIMIZED;
			}`,
		)

		assert.True(t, symbolOption.IsNone(), "Element found")
	})

	t.Run("Should find local enumerator definition associated value", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus : int (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			fn void main() {
				int status = WindowStatus.BACKGROUND.c|||ounter;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		variable, ok := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an associated value, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "counter", variable.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should find local enumerator definition associated value without custom backing type", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus : (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			fn void main() {
				int status = WindowStatus.BACKGROUND.c|||ounter;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		variable, ok := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an associated value, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "counter", variable.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should find associated value on enum instance variable", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus : int (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			fn void main() {
				WindowStatus status = WindowStatus.BACKGROUND;
				int value = status.c|||ounter;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		variable, ok := symbolOption.Get().(*idx.Variable)
		assert.True(t, ok, fmt.Sprintf("The symbol is not an associated value, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "counter", variable.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should find associated value on enum instance struct member", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus : int (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			struct MyStruct { WindowStatus stat; }
			fn void main() {
				MyStruct wrapper = { WindowStatus.BACKGROUND };
				int value = wrapper.stat.c|||ounter;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		variable, ok := symbolOption.Get().(*idx.Variable)
		assert.True(t, ok, fmt.Sprintf("The symbol is not an associated value, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "counter", variable.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should not find associated value on enum type", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus : int (int counter) {
				OPEN = 1,
				BACKGROUND = 2,
				MINIMIZED = 3
			}
			fn void main() {
				WindowStatus.c|||ounter;
			}`,
		)

		assert.True(t, symbolOption.IsNone(), "Element was found")
	})

	t.Run("Should find local implicit enumerator definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn void main() {
				WindowStatus status;
				status = |||BACKGROUND;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.Enumerator)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an enumerator, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "BACKGROUND", symbolOption.Get().GetName())
	})

	t.Run("Should find enum method definition on instance variable", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn bool WindowStatus.isOpen(){}

			fn void main() {
				WindowStatus val = OPEN;
				val.is|||Open();
			}
			`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.Function)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not a method, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "WindowStatus.isOpen", symbolOption.Get().GetName())
	})

	t.Run("Should find enum method definition on explicit enumerator", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn bool WindowStatus.isOpen(){}

			fn void main() {
				WindowStatus.OPEN.isO|||pen();
			}
			`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.Function)
		assert.True(t, ok, fmt.Sprintf("The symbol is not a method, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "WindowStatus.isOpen", symbolOption.Get().GetName())
	})
}

func TestLanguage_findClosestSymbolDeclaration_faults(t *testing.T) {
	t.Run("Find local fault definition in type declaration", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				W|||indowError error = WindowError.SOMETHING_HAPPENED;
				error = UNEXPECTED_ERROR;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Fault not found")

		fault := symbolOption.Get().(*idx.Fault)
		assert.Equal(t, "WindowError", fault.GetName())
	})

	t.Run("Find local fault variable definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				WindowError error = WindowError.SOMETHING_HAPPENED;
				e|||rror = UNEXPECTED_ERROR;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Fault not found")

		fault := symbolOption.Get().(*idx.Variable)
		assert.Equal(t, "error", fault.GetName())
	})

	t.Run("Should find implicit fault constant definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				WindowError error = WindowError.SOMETHING_HAPPENED;
				error = U|||NEXPECTED_ERROR;
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.FaultConstant)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an fault constant, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "UNEXPECTED_ERROR", symbolOption.Get().GetName())
	})

	t.Run("Should not find fault constant on fault constant", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				WindowError.SOMETHING_HAPPENED.U|||NEXPECTED_ERROR;
			}`,
		)

		assert.True(t, symbolOption.IsNone(), "Element found")
	})

	t.Run("Should not find fault constant on fault instance", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn void main() {
				WindowError error = WindowError.SOMETHING_HAPPENED;
				error.U|||NEXPECTED_ERROR;
			}`,
		)

		assert.True(t, symbolOption.IsNone(), "Element found")
	})

	t.Run("Should find fault method definition on instance variable", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn bool WindowError.isBad(){}

			fn void main() {
				WindowError val = UNEXPECTED_ERROR;
				val.is|||Bad();
			}
			`,
		)

		assert.False(t, symbolOption.IsNone(), "Method not found")
		_, ok := symbolOption.Get().(*idx.Function)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not a method, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "WindowError.isBad", symbolOption.Get().GetName())
	})

	t.Run("Should find fault method definition on explicit fault constant", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn bool WindowError.isBad(){}

			fn void main() {
				WindowError.UNEXPECTED_ERROR.isB|||ad();
			}
			`,
		)

		assert.False(t, symbolOption.IsNone(), "Method not found")
		_, ok := symbolOption.Get().(*idx.Function)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not a method, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "WindowError.isBad", symbolOption.Get().GetName())
	})
}

func TestLanguage_findClosestSymbolDeclaration_def(t *testing.T) {
	t.Run("Find local definition definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`def Kilo = int;
			K|||ilo value = 3;`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		assert.Equal(t, "Kilo", symbolOption.Get().GetName())
	})
}

func TestLanguage_findClosestSymbolDeclaration_functions(t *testing.T) {
	t.Run("Find local function definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fn void run(int tick) {
			}
			fn void main() {
				r|||un(3);
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "run", fun.GetName())
		assert.Equal(t, "void", fun.GetReturnType().GetName())
	})

	t.Run("Should not confuse function with virtual root scope function", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fn void main() {
				run(3);
			}
			fn void call(){ m|||ain(); }`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "main", fun.GetName())
		assert.Equal(t, idx.FunctionType(idx.UserDefined), fun.FunctionType())
	})

	t.Run("Should find function definition without body", func(t *testing.T) {
		symbolOption := SearchUnderCursor_ClosestDecl(
			`fn void init_window(int width, int height, char* title) @extern("InitWindow");

			i|||nit_window(200, 200, "hello");
			`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "init_window", fun.GetName())
		assert.Equal(t, idx.FunctionType(idx.UserDefined), fun.FunctionType())
	})
}
