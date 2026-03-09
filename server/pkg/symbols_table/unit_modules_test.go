package symbols_table

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func TestParserModules_should_get_scopes_of_given_module(t *testing.T) {
	docId := "a-doc"
	pm := NewParsedModules(&docId)
	module := symbols.NewModuleBuilder("xxx", docId).Build()
	pm.modules.Set("foo", module)

	assert.Equal(t, module, pm.Get("foo"))
}

func TestParserModules_GetLoadableModules_should_get_scopes_that_are_children_of_given_module(t *testing.T) {
	docId := "a-doc"
	pm := NewParsedModules(&docId)
	loadableModule := symbols.NewModuleBuilder("foo::bar", docId).Build()
	pm.modules.Set("foo::bar", loadableModule)
	loadableModule2 := symbols.NewModuleBuilder("foo", docId).Build()
	pm.modules.Set("foo", loadableModule2)
	notLoadableModule := symbols.NewModuleBuilder("yyy", docId).Build()
	pm.modules.Set("yyy", notLoadableModule)

	modules := pm.GetLoadableModules(symbols.NewModulePathFromString("foo"))

	assert.Equal(t, loadableModule, modules[0])
	assert.Equal(t, loadableModule2, modules[1])
	assert.Equal(t, 2, len(modules))
}

func TestParserModules_GetLoadableModules_should_get_scopes_that_are_parent_of_given_module(t *testing.T) {
	docId := "a-doc"
	pm := NewParsedModules(&docId)
	loadableModule := symbols.NewModuleBuilder("foo::bar", docId).Build()
	pm.modules.Set("foo::bar", loadableModule)
	loadableModule2 := symbols.NewModuleBuilder("foo", docId).Build()
	pm.modules.Set("foo", loadableModule2)
	notLoadableModule := symbols.NewModuleBuilder("yyy", docId).Build()
	pm.modules.Set("yyy", notLoadableModule)
	notLoadableModule2 := symbols.NewModuleBuilder("foo::circle", docId).Build()
	pm.modules.Set("foo::circle", notLoadableModule2)

	modules := pm.GetLoadableModules(symbols.NewModulePathFromString("foo::bar::line"))

	assert.Equal(t, loadableModule, modules[0])
	assert.Equal(t, loadableModule2, modules[1])
	assert.Equal(t, 2, len(modules))
}

func TestParserModules_HasImplicitLoadableModules_should_return_false_when_there_are_not_implicitly_loadable_modules(t *testing.T) {
	cases := []struct {
		desc            string
		searchingModule string
		existingModule  string
	}{
		{"no matching", "foo", "xxx"},
		{"cousing module", "foo::bar", "foo::circle"},
	}
	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			module := symbols.NewModulePathFromString(tt.searchingModule)
			docId := "a-doc"
			pm := NewParsedModules(&docId)
			loadableModule := symbols.NewModuleBuilder(tt.existingModule, docId).Build()
			pm.modules.Set(tt.existingModule, loadableModule)

			assert.False(t, pm.HasImplicitLoadableModules(module))
		})
	}
}

func TestParserModules_HasImplicitLoadableModules_should_return_true_when_there_is_same_module(t *testing.T) {
	module := symbols.NewModulePathFromString("foo")

	docId := "a-doc"
	pm := NewParsedModules(&docId)
	loadableModule := symbols.NewModuleBuilder("foo", docId).Build()
	pm.modules.Set("foo", loadableModule)

	assert.True(t, pm.HasImplicitLoadableModules(module))
}

func TestParserModules_HasImplicitLoadableModules_should_return_true_when_there_is_a_child_module(t *testing.T) {
	module := symbols.NewModulePathFromString("foo")
	docId := "a-doc"
	pm := NewParsedModules(&docId)
	loadableModule := symbols.NewModuleBuilder("foo::bar", docId).Build()
	pm.modules.Set("foo::bar", loadableModule)

	assert.True(t, pm.HasImplicitLoadableModules(module))
}

func TestParserModules_HasImplicitLoadableModules_should_return_true_when_there_is_a_parent_module(t *testing.T) {
	module := symbols.NewModulePathFromString("foo::bar")
	docId := "a-doc"
	pm := NewParsedModules(&docId)
	loadableModule := symbols.NewModuleBuilder("foo", docId).Build()
	pm.modules.Set("foo", loadableModule)

	assert.True(t, pm.HasImplicitLoadableModules(module))
}

func TestRegisterModule_UsesSymbolNameWhenModulePathMissing(t *testing.T) {
	docID := "doc"
	pm := NewParsedModules(&docID)
	mod := symbols.NewModuleBuilder("std::io", docID).Build()
	mod.Module = symbols.ModulePath{}

	pm.RegisterModule(mod)

	assert.NotNil(t, pm.Get("std::io"))
	assert.Equal(t, "std::io", pm.Get("std::io").GetModule().GetName())
}

func TestFindContextModuleInCursorPosition_prefers_nearest_previous_module(t *testing.T) {
	docID := "doc"
	pm := NewParsedModules(&docID)

	moduleA := symbols.NewModule(
		"bgimpl::data_methods",
		docID,
		symbols.NewRange(5, 7, 5, 22),
		symbols.NewRange(5, 0, 5, 22),
	)
	pm.modules.Set(moduleA.GetName(), moduleA)

	moduleB := symbols.NewModule(
		"std::core::array",
		docID,
		symbols.NewRange(204, 7, 204, 24),
		symbols.NewRange(204, 0, 204, 24),
	)
	pm.modules.Set(moduleB.GetName(), moduleB)

	contextBeforeSecond := pm.FindContextModuleInCursorPosition(symbols.NewPosition(40, 13))
	assert.Equal(t, "bgimpl::data_methods", contextBeforeSecond)

	contextAfterSecond := pm.FindContextModuleInCursorPosition(symbols.NewPosition(215, 4))
	assert.Equal(t, "std::core::array", contextAfterSecond)
}
