package symbols_table

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/stretchr/testify/assert"
)

func TestSymbolsTable_should_expand_substructs(t *testing.T) {
	docId := "aDocId"
	mod := "xx"
	symbolsTable := NewSymbolsTable()

	um := NewParsedModules(docId)
	module := symbols.NewModuleBuilder(mod, docId).Build()
	// Add Struct to be inlined
	module.AddStruct(
		symbols.NewStructBuilder("ToInline", mod, docId).
			WithStructMember("a", "int", mod, docId).
			WithStructMember("b", "char", mod, docId).
			Build(),
	)
	module.AddStruct(
		symbols.NewStructBuilder("ToProcess", mod, docId).
			WithStructMember("c", "int", mod, docId).
			WithSubStructMember("x", "ToInline", mod, docId).
			Build(),
	)
	um.modules.Set("xx", module)

	pendingToResolve := NewPendingToResolve()
	pendingToResolve.AddStructSubtype2(module.Structs["ToProcess"])
	symbolsTable.Register(um, pendingToResolve)

	// After registering new unit modules, inlined structs should be expanded
	members := module.Structs["ToProcess"].GetMembers()
	assert.True(t, members[1].IsExpandedInline())

	assert.Equal(t, "a", members[2].GetName())
	assert.Equal(t, "int", members[2].GetType().GetName())

	assert.Equal(t, "b", members[3].GetName())
	assert.Equal(t, "char", members[3].GetType().GetName())

	assert.Equal(t, 0, len(symbolsTable.pendingToResolve.subtyptingToResolve))
}

func TestExtractSymbols_find_variables_flag_pending_to_resolve(t *testing.T) {
	/*
		t.Run("resolves basic type declaration should not flag type as pending to be resolved", func(t *testing.T) {
			docId := "aDocId"
			mod := "xx"
			symbolsTable := NewSymbolsTable()

			um := NewParsedModules(docId)
			module := symbols.NewModuleBuilder(mod, docId).Build()
			// Add Struct to be inlined
			module.AddVariable(
				symbols.NewVariableBuilder("value", "int", mod, docId).Build(),
			)
			um.modules.Set("xx", module)

			pendingToResolve := NewPendingToResolve()
			pendingToResolve.AddVariableType([]*symbols.Variable{module.Variables["value"]}, module)
			symbolsTable.Register(um, pendingToResolve)

			assert.Equal(t, 0, len(pendingToResolve.GetTypesByModule(mod)))
		})*/

	t.Run("user type declaration defined in same file & module should resolve", func(t *testing.T) {
		docId := "aDocId"
		mod := "xx"
		symbolsTable := NewSymbolsTable()

		um := NewParsedModules(docId)
		module := symbols.NewModuleBuilder(mod, docId).Build()
		module.AddVariable(
			symbols.NewVariableBuilder("value", "Ref", mod, docId).Build(),
		)
		module.AddDef(
			symbols.NewDefBuilder("Ref", mod, docId).Build(),
		)
		um.modules.Set("xx", module)

		pendingToResolve := NewPendingToResolve()
		pendingToResolve.AddVariableType([]*symbols.Variable{module.Variables["value"]}, module)
		symbolsTable.Register(um, pendingToResolve)

		assert.Equal(t, true, symbolsTable.pendingToResolve.GetTypesByModule(mod)[0].solved)
	})

	t.Run("user type declaration defined in different file & module should resolve", func(t *testing.T) {
		docId := "aDocId"
		mod := "xx"
		symbolsTable := NewSymbolsTable()

		um := NewParsedModules(docId)
		module := symbols.NewModuleBuilder(mod, docId).Build()
		module.AddVariable(
			symbols.NewVariableBuilder("value", "Ref", mod, docId).Build(),
		)
		module.AddImports([]string{"yy"})

		um.modules.Set(mod, module)
		pendingToResolve := NewPendingToResolve()
		pendingToResolve.AddVariableType([]*symbols.Variable{module.Variables["value"]}, module)
		symbolsTable.Register(um, pendingToResolve)

		docBId := "aDocBId"
		modB := "yy"
		umB := NewParsedModules(docBId)
		moduleB := symbols.NewModuleBuilder(modB, docBId).Build()
		moduleB.AddDef(
			symbols.NewDefBuilder("Ref", modB, docBId).Build(),
		)
		umB.modules.Set(mod, moduleB)
		symbolsTable.Register(umB, NewPendingToResolve())

		assert.Equal(t, true, symbolsTable.pendingToResolve.GetTypesByModule(mod)[0].solved)
	})

	t.Run("user type declaration defined in different file & module should be resolved after imported file is parsed", func(t *testing.T) {
		/*
			parser := createParser()

			// First file
			source := `
			module main;
			import external;

			struct Data {
				MyType copyValue;
			}`
			docId := "main.c3"
			doc := document.NewDocument(docId, source)
			symbols := parser.ParseSymbols(&doc)

			mainStruct := symbols.Get("main").Structs["Data"]
			assert.NotNil(t, mainStruct)

			assert.Equal(t, 1, len(parser.pendingToResolve.typesByModule["main"]), "Custom type should be flagged as pending to resolve.")

			// Second trap file
			source = `
			module trap;
			def MyType = char;`
			docId = "trap.c3"
			doc = document.NewDocument(docId, source)
			parser.ParseSymbols(&doc)
			assert.Equal(t, 1, len(parser.pendingToResolve.typesByModule["main"]), "Pending resolved with trap file.")

			// Second file
			source = `
			module external;
			def MyType = int;`
			docId = "external.c3"
			doc = document.NewDocument(docId, source)
			symbols = parser.ParseSymbols(&doc)

			found := symbols.Get("external").Defs["MyType"]
			assert.NotNil(t, found)

			assert.Equal(t, 0, len(parser.pendingToResolve.typesByModule["main"]), "Custom type should be flagged as pending to resolve.")
			assert.Equal(t, "external::MyType", mainStruct.GetMembers()[0].GetType().GetFullQualifiedName())
		*/
	})
}
