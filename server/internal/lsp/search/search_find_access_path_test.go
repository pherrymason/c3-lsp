package search

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func SearchUnderCursor_AccessPath(body string, optionalState ...TestState) SearchResult {
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

	doc := state.GetDoc("app.c3")
	searchParams := search_params.BuildSearchBySymbolUnderCursor(
		&doc,
		*state.state.GetUnitModulesByDoc(doc.URI),
		position,
	)

	return search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)
}

func TestProjectState_findClosestSymbolDeclaration_access_path(t *testing.T) {
	t.Run("Should find method from std collection", func(t *testing.T) {
		state := NewTestStateWithStdLibVersion("0.5.5")
		symbolOption := SearchUnderCursor_AccessPath(
			`module core::actions;
			import std::collections::map;

			def ActionListMap = HashMap(<char*, ActionList>);
			struct ActionListManager{
				ActionListMap actionLists;
			}
			fn void ActionListManager.addActionList(&self, ActionList actionList) {
				self.actionLists.s|||et(actionList.getName(), actionList);
			}`,
			state,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "HashMap.set", fun.GetName())
	})

	t.Run("Should find fault constant definition with path definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			WindowError error = WindowError.S|||OMETHING_HAPPENED;`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.FaultConstant)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an fault constant, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "SOMETHING_HAPPENED", symbolOption.Get().GetName())
	})

	t.Run("Should find local struct member variable definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			Emu emulator;
			emulator.o|||n = true;`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.StructMember)
		assert.Equal(t, "on", symbol.GetName())
		assert.Equal(t, "bool", variable.GetType().GetName())
	})

	t.Run("Should find local struct member variable definition when struct is a pointer", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn void Emu.run(Emu* emu) {
				emu.o|||n = true;
				emu.tick();
			}`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.StructMember)
		assert.Equal(t, "on", symbol.GetName())
		assert.Equal(t, "bool", variable.GetType().GetName())
	})

	// This test maybe works better in language_find_closes_declaration_test.go
	t.Run("Should find same struct member declaration, when cursor is already in member declaration", func(t *testing.T) {
		t.Skip() // Do not understand this test.
		symbolOption := SearchUnderCursor_AccessPath(
			`Cpu cpu; // Trap for finding struct member when cursor is on declaration member.
			struct Emu {
				Cpu c|||pu;
				Audio audio;
				bool on;
			  }`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.StructMember)
		assert.Equal(t, "cpu", symbol.GetName())
		assert.Equal(t, "Cpu", variable.GetType())
	})

	t.Run("Should find same struct member declaration, when struct is behind a def and cursor is already in member declaration", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`
			struct Camera3D {
				int target;
			}
			def Camera = Camera3D;

			struct Widget {
				int count;
				Camera camera;
			}

			Widget view = {};
			view.camera.t|||arget = 3;
			`,
		)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.StructMember)
		assert.Equal(t, "target", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should find struct method", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn void Emu.init(Emu* emu) {}
			fn void main() {
				Emu emulator;
				emulator.i|||nit();
			}`,
		)

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "Emu.init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct method on alternative callable", func(t *testing.T) {
		resolvedSymbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn void Emu.init(Emu* emu) {}
			fn void main() {
				Emu emulator;
				Emu.i|||nit(&emulator);
			}`,
		)

		fun := resolvedSymbolOption.Get().(*idx.Function)
		assert.Equal(t, "Emu.init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct method when cursor is already in method declaration", func(t *testing.T) {
		resolvedSymbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn void Emu.i|||nit(Emu* emu) {}`,
		)

		fun := resolvedSymbolOption.Get().(*idx.Function)
		assert.Equal(t, "Emu.init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct member when cursor is on chained returned from function", func(t *testing.T) {
		resolvedSymbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn Emu newEmu() {
				Emu emulator;
				return emulator;
			}
			fn void main() {
				newEmu().o|||n = false;
			}`,
		)

		variable := resolvedSymbolOption.Get().(*idx.StructMember)
		assert.Equal(t, "on", variable.GetName())
		assert.Equal(t, "bool", variable.GetType().GetName())
	})

	t.Run("Should find struct method when cursor is on chained returned from function", func(t *testing.T) {
		resolvedSymbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn Emu newEmu() {
				Emu emulator;
				return emulator;
			}
			fn void Emu.init(){}
			fn void main() {
				newEmu().i|||nit();
			}`,
		)

		fun := resolvedSymbolOption.Get().(*idx.Function)
		assert.Equal(t, "Emu.init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find local struct method when there are N nested structs", func(t *testing.T) {
		resolvedSymbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			}
			fn void Emu.init(Emu* emu) {
				emu.audio.i|||nit();
			}
			struct Audio {
				int frequency;
			}
			fn void Audio.init() {}`,
		)

		assert.False(t, resolvedSymbolOption.IsNone(), "Struct method not found")

		fun, ok := resolvedSymbolOption.Get().(*idx.Function)
		assert.True(t, ok, "Struct method not found")
		assert.Equal(t, "Audio.init", fun.GetName())
		assert.Equal(t, "Audio.init", fun.GetFullName())
	})

	t.Run("Should find struct method on alternative callable when there are N nested structs", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			}
			fn void Emu.init(Emu* emu) {
				Audio.i|||nit(&emu.audio);
			}
			struct Audio {
				int frequency;
			}
			fn void Audio.init() {}`,
		)

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "Audio.init", fun.GetName())
		assert.Equal(t, "Audio.init", fun.GetFullName())
	})

	t.Run("Should not find local struct method definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			}
			fn void Emu.init(Emu* emu) {
				emu.audio.u|||nknown();
			}
			struct Audio {
				int frequency;
			}
			fn void Audio.init() {}`,
		)

		assert.True(t, symbolOption.IsNone(), "Struct method should not be found")
	})

	t.Run("Asking the selectedSymbol information in the very same declaration, should resolve to the correct selectedSymbol. Even if there is another selectedSymbol with same name in a different file.", func(t *testing.T) {
		t.Skip()
		// Should only resolve in very same module, unless module B is imported.
		// ---------------------
		// module A has int out;
		// module B has int out;
		// asking info about B::out should resolve to B::out, and not A::out.

		// Other cases:
		// module A;
		// struct MyStruct{}
		// fn void MyStruct.search(&self) {}
		// fn void search() {}
		//
		// module B;
		// MyStruct object;
		// object.search();
	})
}

func TestProjectState_findClosestSymbolDeclaration_access_path_enums(t *testing.T) {
	t.Run("Should find enumerator with path definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			WindowStatus stat = WindowStatus.O|||PEN;`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.Enumerator)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an enumerator, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "OPEN", symbolOption.Get().GetName())
	})

	t.Run("Should not find enumerator after explicit enumerator path", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			WindowStatus stat = WindowStatus.OPEN.B|||ACKGROUND;`,
		)

		assert.True(t, symbolOption.IsNone(), "Element was found")
	})

	t.Run("Should not find enumerator after instance variable", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			WindowStatus stat = WindowStatus.OPEN;
			WindoWStatus bad = stat.B|||ACKGROUND;`,
		)

		assert.True(t, symbolOption.IsNone(), "Element was found")
	})

	t.Run("Should find enum method", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn bool WindowStatus.isOpen() {}
			fn void main() {
				WindowStatus val = WindowStatus.OPEN;
				val.is|||Open();
			}`,
		)

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "WindowStatus.isOpen", fun.GetName())
		assert.Equal(t, "WindowStatus.isOpen", fun.GetFullName())
	})

	t.Run("Should find enum method on explicit enumerator", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn bool WindowStatus.isOpen() {}
			fn void main() {
				WindowStatus.OPEN.i|||sOpen();
			}`,
		)

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "WindowStatus.isOpen", fun.GetName())
		assert.Equal(t, "WindowStatus.isOpen", fun.GetFullName())
	})

	t.Run("Should find associated value on explicit enumerator", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`enum WindowStatus : int (int assoc) {
				OPEN = 5,
				BACKGROUND = 6,
				MINIMIZED = 7
			}
			int stat = WindowStatus.OPEN.a|||ssoc;`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		variable, ok := symbolOption.Get().(*idx.Variable)
		assert.True(t, ok, fmt.Sprintf("The symbol is not a variable, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "assoc", variable.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should find associated value on explicit enumerator without custom backing type", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`enum WindowStatus : (int assoc) {
				OPEN = 5,
				BACKGROUND = 6,
				MINIMIZED = 7
			}
			int stat = WindowStatus.OPEN.a|||ssoc;`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		variable, ok := symbolOption.Get().(*idx.Variable)
		assert.True(t, ok, fmt.Sprintf("The symbol is not a variable, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "assoc", variable.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should find associated value on enum instance variable", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`enum WindowStatus : (int assoc) {
				OPEN = 5,
				BACKGROUND = 6,
				MINIMIZED = 7
			}
			WindowStatus stat = WindowStatus.OPEN;
			int val = stat.a|||ssoc;`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		variable, ok := symbolOption.Get().(*idx.Variable)
		assert.True(t, ok, fmt.Sprintf("The symbol is not a variable, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "assoc", variable.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})
}

func TestProjectState_findClosestSymbolDeclaration_access_path_faults(t *testing.T) {
	t.Run("Should find fault constant with path definition", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			WindowError err = WindowError.U|||NEXPECTED_ERROR;`,
		)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.FaultConstant)
		assert.True(t, ok, fmt.Sprintf("The symbol is not a fault constant, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "UNEXPECTED_ERROR", symbolOption.Get().GetName())
	})

	t.Run("Should not find fault constant after explicit instance", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			WindowError err = WindowError.UNEXPECTED_ERROR.S|||OMETHING_HAPPENED;`,
		)

		assert.True(t, symbolOption.IsNone(), "Constant was wrongly found on instance")
	})

	t.Run("Should not find fault constant after instance variable", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			WindowError err = WindowError.UNEXPECTED_ERROR;
			WindowError bad = err.S|||OMETHING_HAPPENED;`,
		)

		assert.True(t, symbolOption.IsNone(), "Constant was wrongly found on instance variable")
	})

	t.Run("Should not find fault constant after instance variable in struct member", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			struct MyStruct { WindowError f; }
			MyStruct st = { WindowError.UNEXPECTED_ERROR };
			WindowError bad = st.f.S|||OMETHING_HAPPENED;`,
		)

		assert.True(t, symbolOption.IsNone(), "Constant was wrongly found on instance variable")
	})

	t.Run("Should find fault method on instance variable", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn bool WindowError.isBad() {}
			fn void main() {
				WindowError val = WindowError.UNEXPECTED_ERROR;
				val.is|||Bad();
			}`,
		)

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "WindowError.isBad", fun.GetName())
		assert.Equal(t, "WindowError.isBad", fun.GetFullName())
	})

	t.Run("Should find fault method after instance variable in struct member", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn bool WindowError.isBad() {}
			struct MyStruct { WindowError f; }
			MyStruct st = { WindowError.UNEXPECTED_ERROR };
			WindowError bad = st.f.i|||sBad();`,
		)

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "WindowError.isBad", fun.GetName())
		assert.Equal(t, "WindowError.isBad", fun.GetFullName())
	})

	t.Run("Should find fault method on explicit constant", func(t *testing.T) {
		symbolOption := SearchUnderCursor_AccessPath(
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn bool WindowError.isBad() {}
			fn void main() {
				WindowError.UNEXPECTED_ERROR.is|||Bad();
			}`,
		)

		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "WindowError.isBad", fun.GetName())
		assert.Equal(t, "WindowError.isBad", fun.GetFullName())
	})
}

func TestProjectState_findClosestSymbolDeclaration_access_path_with_generics(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Should xxxxxx", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`module app;
			import list;

			struct Home {
				List(<Room>) rooms;
			}
			struct Room {
				String name;
			}
			fn void Room.paint() {}

			fn void main() {
				Home home;
				home.rooms.get(0).paint();
			}`,
		)

		state.registerDoc(
			"list.c3",
			`module list(<Type>);
			struct List (Printable)
			{
				usz size;
				usz capacity;
				Allocator allocator;
				Type *entries;
			}
			fn Type List.get(usz index) {}`,
		)
		doc := state.GetDoc("app.c3")
		position := buildPosition(14, 23) // Cursor is at home.rooms.p|aint()
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.True(t, symbolOption.IsSome(), "Struct method not found")
		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "Room.paint", fun.GetName())
		assert.Equal(t, "Room.paint", fun.GetFullName())
	})
}
