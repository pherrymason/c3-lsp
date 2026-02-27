package stdlib

import (
	"encoding/json"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/stretchr/testify/assert"
)

func TestRehydrateModule_RestoresLookupDataAfterJSONRoundTrip(t *testing.T) {
	docID := "_stdlib_0.7.10"
	original := symbols.NewModule("std::io", "/tmp/io.c3", symbols.NewRange(0, 0, 0, 0), symbols.NewRange(0, 0, 1, 0))
	docs := symbols.NewDocCommentBuilder("Print any value to stdout, appending a newline.").
		WithContract("@param", "x: The value to print").
		Build()
	fun := symbols.NewFunctionBuilder("printn", symbols.NewTypeFromString("void", "std::io"), "std::io", "/tmp/io.c3").
		WithDocs(docs).
		Build()
	original.AddFunction(fun)

	payload, err := json.Marshal(original)
	assert.NoError(t, err)

	loaded := &symbols.Module{}
	err = json.Unmarshal(payload, loaded)
	assert.NoError(t, err)

	assert.Equal(t, "", loaded.GetModule().GetName())
	assert.Len(t, loaded.NestedScopes(), 0)

	rehydrateModule(loaded)

	assert.Equal(t, "std::io", loaded.GetModule().GetName())
	assert.Len(t, loaded.NestedScopes(), 1)
	rehydratedPrintn := loaded.ChildrenFunctions[0]
	assert.NotNil(t, rehydratedPrintn.GetDocComment())
	assert.NotEmpty(t, rehydratedPrintn.GetDocComment().GetBody())
	assert.True(t, rehydratedPrintn.GetDocComment().HasContracts())

	modules := symbols_table.NewParsedModules(&docID)
	modules.RegisterModule(loaded)
	loadable := modules.GetLoadableModules(symbols.NewModulePathFromString("std::io"))
	assert.Len(t, loadable, 1)
}
