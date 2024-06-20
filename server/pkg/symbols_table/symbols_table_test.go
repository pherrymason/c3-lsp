package symbols_table

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func TestSymbolsTable_should_expand_substructs(t *testing.T) {
	docId := "aDocId"
	mod := "xx"
	symbolsTable := NewSymbolsTable()

	um := NewParsedModules(&docId)
	module := symbols.NewModuleBuilder(mod, &docId).Build()
	// Add Struct to be inlined
	module.AddStruct(
		symbols.NewStructBuilder("ToInline", mod, &docId).
			WithStructMember("a", "int", mod, &docId).
			WithStructMember("b", "char", mod, &docId).
			Build(),
	)
	module.AddStruct(
		symbols.NewStructBuilder("ToProcess", mod, &docId).
			WithStructMember("c", "int", mod, &docId).
			WithSubStructMember("x", "ToInline", mod, &docId).
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
	t.Run("resolves variable type defined in same file & module should resolve", func(t *testing.T) {
		docId := "aDocId"
		mod := "xx"
		symbolsTable := NewSymbolsTable()

		um := NewParsedModules(&docId)
		module := symbols.NewModuleBuilder(mod, &docId).Build()
		module.AddVariable(
			symbols.NewVariableBuilder("value", "Ref", mod, &docId).Build(),
		)
		module.AddDef(
			symbols.NewDefBuilder("Ref", mod, &docId).Build(),
		)
		um.modules.Set("xx", module)

		pendingToResolve := NewPendingToResolve()
		pendingToResolve.AddVariableType([]*symbols.Variable{module.Variables["value"]}, module)
		symbolsTable.Register(um, pendingToResolve)

		assert.Equal(t, true, symbolsTable.pendingToResolve.GetTypesByModule(mod)[0].solved)
	})

	t.Run("resolves variable type declaration defined in different file & module should resolve", func(t *testing.T) {
		docId := "aDocId"
		mod := "xx"
		symbolsTable := NewSymbolsTable()

		um := NewParsedModules(&docId)
		module := symbols.NewModuleBuilder(mod, &docId).Build()
		module.AddVariable(
			symbols.NewVariableBuilder("value", "Ref", mod, &docId).Build(),
		)
		module.AddImports([]string{"yy"})

		um.modules.Set(mod, module)
		pendingToResolve := NewPendingToResolve()
		pendingToResolve.AddVariableType([]*symbols.Variable{module.Variables["value"]}, module)
		symbolsTable.Register(um, pendingToResolve)

		docBId := "aDocBId"
		modB := "yy"
		umB := NewParsedModules(&docBId)
		moduleB := symbols.NewModuleBuilder(modB, &docBId).Build()
		moduleB.AddDef(
			symbols.NewDefBuilder("Ref", modB, &docBId).Build(),
		)
		umB.modules.Set(mod, moduleB)
		symbolsTable.Register(umB, NewPendingToResolve())

		assert.Equal(t, true, symbolsTable.pendingToResolve.GetTypesByModule(mod)[0].solved)
	})

	t.Run("resolves struct member type declaration defined in different file & module should resolve", func(t *testing.T) {
		docId := "aDocId"
		mod := "xx"
		symbolsTable := NewSymbolsTable()

		um := NewParsedModules(&docId)
		module := symbols.NewModuleBuilder(mod, &docId).Build()
		module.AddStruct(
			symbols.NewStructBuilder("CustomStruct", mod, &docId).
				WithStructMember("a", "Ref", mod, &docId).
				WithStructMember("b", "char", mod, &docId).
				Build(),
		)
		module.AddImports([]string{"yy"})

		um.modules.Set(mod, module)
		pendingToResolve := NewPendingToResolve()
		pendingToResolve.AddStructMemberTypes(module.Structs["CustomStruct"], module)
		symbolsTable.Register(um, pendingToResolve)

		docBId := "aDocBId"
		modB := "yy"
		umB := NewParsedModules(&docBId)
		moduleB := symbols.NewModuleBuilder(modB, &docBId).Build()
		moduleB.AddDef(
			symbols.NewDefBuilder("Ref", modB, &docBId).Build(),
		)
		umB.modules.Set(mod, moduleB)
		symbolsTable.Register(umB, NewPendingToResolve())

		assert.Equal(t, true, symbolsTable.pendingToResolve.GetTypesByModule(mod)[0].solved)
		assert.Equal(t, "yy::Ref", module.Structs["CustomStruct"].GetMembers()[0].GetType().GetFullQualifiedName())
	})

	t.Run("resolves function return and argument types defined in different file & module should resolve", func(t *testing.T) {
		docId := "aDocId"
		mod := "xx"
		symbolsTable := NewSymbolsTable()

		um := NewParsedModules(&docId)
		module := symbols.NewModuleBuilder(mod, &docId).Build()
		module.AddFunction(
			symbols.NewFunctionBuilder("foo", symbols.NewTypeFromString("Ref", mod), mod, &docId).
				WithArgument(
					symbols.NewVariableBuilder("zoo", "Ref", mod, &docId).Build(),
				).
				Build(),
		)
		module.AddImports([]string{"yy"})

		um.modules.Set(mod, module)
		pendingToResolve := NewPendingToResolve()
		pendingToResolve.AddFunctionTypes(module.ChildrenFunctions[0], module)
		symbolsTable.Register(um, pendingToResolve)

		docBId := "aDocBId"
		modB := "yy"
		umB := NewParsedModules(&docBId)
		moduleB := symbols.NewModuleBuilder(modB, &docBId).Build()
		moduleB.AddDef(
			symbols.NewDefBuilder("Ref", modB, &docBId).Build(),
		)
		umB.modules.Set(mod, moduleB)
		symbolsTable.Register(umB, NewPendingToResolve())

		assert.Equal(t, true, symbolsTable.pendingToResolve.GetTypesByModule(mod)[0].solved)
		assert.Equal(t, "yy::Ref", module.ChildrenFunctions[0].GetReturnType().GetFullQualifiedName())
		assert.Equal(t, "yy::Ref", module.ChildrenFunctions[0].GetArguments()[0].GetType().GetFullQualifiedName())
	})

	t.Run("resolves definition defined in different file & module should resolve", func(t *testing.T) {
		docId := "aDocId"
		mod := "xx"
		symbolsTable := NewSymbolsTable()

		um := NewParsedModules(&docId)
		module := symbols.NewModuleBuilder(mod, &docId).Build()
		module.AddDef(
			symbols.NewDefBuilder("foo", mod, &docId).
				WithResolvesToType(symbols.NewTypeFromString("HashMap", mod)).
				Build(),
		)
		module.AddImports([]string{"std::collections::map"})

		um.modules.Set(mod, module)
		pendingToResolve := NewPendingToResolve()
		pendingToResolve.AddDefType(module.Defs["foo"], module)
		symbolsTable.Register(um, pendingToResolve)

		docBId := "aDocBId"
		modB := "std::collections::map"
		umB := NewParsedModules(&docBId)
		moduleB := symbols.NewModuleBuilder(modB, &docBId).Build()
		moduleB.AddStruct(
			symbols.NewStructBuilder("HashMap", modB, &docBId).Build(),
		)
		umB.modules.Set(mod, moduleB)
		symbolsTable.Register(umB, NewPendingToResolve())

		assert.Equal(t, true, symbolsTable.pendingToResolve.GetTypesByModule(mod)[0].solved)
		assert.Equal(t, "std::collections::map::HashMap", module.Defs["foo"].ResolvedType().GetFullQualifiedName())
	})

}
