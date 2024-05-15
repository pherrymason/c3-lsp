package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/stretchr/testify/assert"
)

func TestParserModules_should_get_scopes_of_given_module(t *testing.T) {
	pm := NewParsedModules("a-doc")
	fun := symbols.NewFunctionBuilder("xxx", "void", "foo", "a-doc").Build()
	pm.fnByModules["foo"] = &fun

	assert.Equal(t, &fun, pm.Get("foo"))
}

func TestParserModules_GetLoadableModules_should_get_scopes_that_are_children_of_given_module(t *testing.T) {
	pm := NewParsedModules("a-doc")
	loadableFunc := symbols.NewFunctionBuilder("xxx", "void", "foo::bar", "a-doc").Build()
	pm.fnByModules["foo::bar"] = &loadableFunc
	loadableFunc2 := symbols.NewFunctionBuilder("xxx", "void", "foo", "a-doc").Build()
	pm.fnByModules["foo"] = &loadableFunc2
	notLoadableFunc := symbols.NewFunctionBuilder("xxx", "void", "yyy", "a-doc").Build()
	pm.fnByModules["yyy"] = &notLoadableFunc

	funcs := pm.GetLoadableModules(symbols.NewModulePathFromString("foo"))

	assert.Equal(t, &loadableFunc, funcs[0])
	assert.Equal(t, &loadableFunc2, funcs[1])
	assert.Equal(t, 2, len(funcs))
}

func TestParserModules_GetLoadableModules_should_get_scopes_that_are_parent_of_given_module(t *testing.T) {
	pm := NewParsedModules("a-doc")
	loadableFunc := symbols.NewFunctionBuilder("xxx", "void", "foo::bar", "a-doc").Build()
	pm.fnByModules["foo::bar"] = &loadableFunc
	loadableFunc2 := symbols.NewFunctionBuilder("xxx", "void", "foo", "a-doc").Build()
	pm.fnByModules["foo"] = &loadableFunc2
	notLoadableFunc := symbols.NewFunctionBuilder("xxx", "void", "yyy", "a-doc").Build()
	pm.fnByModules["yyy"] = &notLoadableFunc
	notLoadableFunc2 := symbols.NewFunctionBuilder("xxx", "void", "foo::circle", "a-doc").Build()
	pm.fnByModules["foo::circle"] = &notLoadableFunc2

	funcs := pm.GetLoadableModules(symbols.NewModulePathFromString("foo::bar::line"))

	assert.Equal(t, &loadableFunc, funcs[0])
	assert.Equal(t, &loadableFunc2, funcs[1])
	assert.Equal(t, 2, len(funcs))
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
			pm := NewParsedModules("a-doc")
			loadableFunc := symbols.NewFunctionBuilder("xxx", "void", tt.existingModule, "a-doc").Build()
			pm.fnByModules[tt.existingModule] = &loadableFunc

			assert.False(t, pm.HasImplicitLoadableModules(module))
		})
	}
}

func TestParserModules_HasImplicitLoadableModules_should_return_true_when_there_is_same_module(t *testing.T) {
	module := symbols.NewModulePathFromString("foo")

	pm := NewParsedModules("a-doc")
	loadableFunc := symbols.NewFunctionBuilder("xxx", "void", "foo", "a-doc").Build()
	pm.fnByModules["foo"] = &loadableFunc

	assert.True(t, pm.HasImplicitLoadableModules(module))
}

func TestParserModules_HasImplicitLoadableModules_should_return_true_when_there_is_a_child_module(t *testing.T) {
	module := symbols.NewModulePathFromString("foo")

	pm := NewParsedModules("a-doc")
	loadableFunc := symbols.NewFunctionBuilder("xxx", "void", "foo::bar", "a-doc").Build()
	pm.fnByModules["foo::bar"] = &loadableFunc

	assert.True(t, pm.HasImplicitLoadableModules(module))
}

func TestParserModules_HasImplicitLoadableModules_should_return_true_when_there_is_a_parent_module(t *testing.T) {
	module := symbols.NewModulePathFromString("foo::bar")

	pm := NewParsedModules("a-doc")
	loadableFunc := symbols.NewFunctionBuilder("xxx", "void", "foo", "a-doc").Build()
	pm.fnByModules["foo"] = &loadableFunc

	assert.True(t, pm.HasImplicitLoadableModules(module))
}
