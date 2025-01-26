package search

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func TestProjectState_findClosestSymbolDeclaration_access_path(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Should find method from std collection", func(t *testing.T) {
		state := NewTestStateWithStdLibVersion("0.5.5")
		state.registerDoc(
			"def.c3",
			`module core::actions;
			import std::collections::map;

			def ActionListMap = HashMap(<char*, ActionList>);
			struct ActionListManager{
				ActionListMap actionLists;
			}
			fn void ActionListManager.addActionList(&self, ActionList actionList) {
				self.actionLists.set(actionList.getName(), actionList);
			}`,
		)
		position := buildPosition(9, 22) // Cursor at `self.actionLists.s|et(actionList.getName(), actionList);`
		doc := state.GetDoc("def.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(
			&doc,
			*state.state.GetUnitModulesByDoc(doc.URI),
			position,
		)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "HashMap.set", fun.GetName())
	})

	t.Run("Should find fault constant definition with path definition", func(t *testing.T) {
		state.registerDoc(
			"faults.c3",
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			WindowError error = WindowError.SOMETHING_HAPPENED;`,
		)
		position := buildPosition(2, 36) // Cursor at `WindowError error = WindowError.S|OMETHING_HAPPENED;`
		doc := state.GetDoc("faults.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.FaultConstant)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an fault constant, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "SOMETHING_HAPPENED", symbolOption.Get().GetName())
	})

	t.Run("Should find local struct member variable definition", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			Emu emulator;
			emulator.on = true;`,
		)
		position := buildPosition(7, 13) // Cursor at `emulator.o|n = true`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.StructMember)
		assert.Equal(t, "on", symbol.GetName())
		assert.Equal(t, "bool", variable.GetType().GetName())
	})

	t.Run("Should find local struct member variable definition when struct is a pointer", func(t *testing.T) {
		state.clearDocs()
		state.registerDoc(
			"structs.c3",
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn void Emu.run(Emu* emu) {
				emu.on = true;
				emu.tick();
			}`,
		)
		position := buildPosition(7, 9) // Cursor at emulator.o|n = true
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.StructMember)
		assert.Equal(t, "on", symbol.GetName())
		assert.Equal(t, "bool", variable.GetType().GetName())
	})

	// This test maybe works better in language_find_closes_declaration_test.go
	t.Run("Should find same struct member declaration, when cursor is already in member declaration", func(t *testing.T) {
		t.Skip() // Do not understand this test.
		state.registerDoc(
			"structs.c3",
			`Cpu cpu; // Trap for finding struct member when cursor is on declaration member.
			struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }`,
		)
		position := buildPosition(12, 8) // Cursor at `Cpu c|pu;`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.StructMember)
		assert.Equal(t, "cpu", symbol.GetName())
		assert.Equal(t, "Cpu", variable.GetType())
	})

	t.Run("Should find same struct member declaration, when struct is behind a def and cursor is already in member declaration", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
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
			view.camera.target = 3;
			`,
		)
		position := buildPosition(13, 16) // Cursor at `view.camera.t|arget = 3;`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.StructMember)
		assert.Equal(t, "target", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})

	t.Run("Should find struct method", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn void Emu.init(Emu* emu) {}
			fn void main() {
				Emu emulator;
				emulator.init();
			}`,
		)
		// Cursor at `emulator.i|nit();`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), buildPosition(9, 14))

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)
		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "Emu.init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct method on alternative callable", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn void Emu.init(Emu* emu) {}
			fn void main() {
				Emu emulator;
				Emu.init(&emulator);
			}`,
		)
		// Cursor at `Emu.i|nit(&emulator);`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), buildPosition(9, 9))

		resolvedSymbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)
		fun := resolvedSymbolOption.Get().(*idx.Function)
		assert.Equal(t, "Emu.init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct method when cursor is already in method declaration", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			  }
			fn void Emu.init(Emu* emu) {}`,
		)
		// Cursor at `Emu.i|nit();`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), buildPosition(6, 16))

		resolvedSymbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)
		fun := resolvedSymbolOption.Get().(*idx.Function)
		assert.Equal(t, "Emu.init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct member when cursor is on chained returned from function", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
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
				newEmu().on = false;
			}`,
		)
		// Cursor at `newEmu().o|n = false;`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), buildPosition(11, 14))

		resolvedSymbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)
		variable := resolvedSymbolOption.Get().(*idx.StructMember)
		assert.Equal(t, "on", variable.GetName())
		assert.Equal(t, "bool", variable.GetType().GetName())
	})

	t.Run("Should find struct method when cursor is on chained returned from function", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
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
				newEmu().init();
			}`,
		)
		// Cursor at `newEmu().i|nit();`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), buildPosition(12, 14))

		resolvedSymbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)
		fun := resolvedSymbolOption.Get().(*idx.Function)
		assert.Equal(t, "Emu.init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find local struct method when there are N nested structs", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			}
			fn void Emu.init(Emu* emu) {
				emu.audio.init();
			}
			struct Audio {
				int frequency;
			}
			fn void Audio.init() {}`,
		)
		position := buildPosition(7, 15) // Cursor at `emu.audio.i|nit();``
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		resolvedSymbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, resolvedSymbolOption.IsNone(), "Struct method not found")

		fun, ok := resolvedSymbolOption.Get().(*idx.Function)
		assert.True(t, ok, "Struct method not found")
		assert.Equal(t, "Audio.init", fun.GetName())
		assert.Equal(t, "Audio.init", fun.GetFullName())
	})

	t.Run("Should find struct method on alternative callable when there are N nested structs", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			}
			fn void Emu.init(Emu* emu) {
				Audio.init(&emu.audio);
			}
			struct Audio {
				int frequency;
			}
			fn void Audio.init() {}`,
		)
		// Cursor at `Audio.i|nit(&emu.audio);`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), buildPosition(7, 11))

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)
		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "Audio.init", fun.GetName())
		assert.Equal(t, "Audio.init", fun.GetFullName())
	})

	t.Run("Should not find local struct method definition", func(t *testing.T) {
		state.registerDoc(
			"structs.c3",
			`struct Emu {
				Cpu cpu;
				Audio audio;
				bool on;
			}
			fn void Emu.init(Emu* emu) {
				emu.audio.unknown();
			}
			struct Audio {
				int frequency;
			}
			fn void Audio.init() {}`,
		)
		doc := state.GetDoc("structs.c3")
		position := buildPosition(7, 15) // Cursor is at emu.audio.u|nknown
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

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
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Should find enumerator with path definition", func(t *testing.T) {
		state.registerDoc(
			"enums.c3",
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			WindowStatus stat = WindowStatus.OPEN;`,
		)
		position := buildPosition(2, 37) // Cursor at `WindowStatus stat = WindowStatus.O|PEN;`
		doc := state.GetDoc("enums.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(
			&doc,
			*state.state.GetUnitModulesByDoc(doc.URI),
			position,
		)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.Enumerator)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an enumerator, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "OPEN", symbolOption.Get().GetName())
	})

	t.Run("Should find enum method", func(t *testing.T) {
		state.registerDoc(
			"enums.c3",
			`enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
			fn bool WindowStatus.isOpen() {}
			fn void main() {
				WindowStatus val = WindowStatus.OPEN;
				val.isOpen();
			}`,
		)
		// Cursor at `val.is|Open();`
		doc := state.GetDoc("enums.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), buildPosition(5, 10))

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)
		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "WindowStatus.isOpen", fun.GetName())
		assert.Equal(t, "WindowStatus.isOpen", fun.GetFullName())
	})
}

func TestProjectState_findClosestSymbolDeclaration_access_path_faults(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Should find fault constant with path definition", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			WindowError err = WindowError.UNEXPECTED_ERROR;`,
		)
		position := buildPosition(2, 34) // Cursor at `WindowStatus stat = WindowStatus.O|PEN;`
		doc := state.GetDoc("app.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(
			&doc,
			*state.state.GetUnitModulesByDoc(doc.URI),
			position,
		)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(*idx.FaultConstant)
		assert.True(t, ok, fmt.Sprintf("The symbol is not a fault constant, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "UNEXPECTED_ERROR", symbolOption.Get().GetName())
	})

	t.Run("Should find fault method", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }
			fn bool WindowError.isBad() {}
			fn void main() {
				WindowError val = WindowError.UNEXPECTED_ERROR;
				val.isBad();
			}`,
		)
		// Cursor at `val.is|Bad();`
		doc := state.GetDoc("app.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), buildPosition(5, 10))

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)
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
