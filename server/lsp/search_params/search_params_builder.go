package search_params

import (
	"github.com/pherrymason/c3-lsp/lsp/document/sourcecode"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/option"
)

type SearchParamsBuilder struct {
	params *SearchParams
}

func NewSearchParamsBuilder() *SearchParamsBuilder {
	return &SearchParamsBuilder{
		params: &SearchParams{},
	}
}

func (b *SearchParamsBuilder) WithSymbolWord(word sourcecode.Word) *SearchParamsBuilder {
	b.params.word = word
	b.params.symbol = word.Text()
	b.params.symbolRange = word.TextRange()
	return b
}

func (b *SearchParamsBuilder) WithText(text string, textRange symbols.Range) *SearchParamsBuilder {
	b.params.word = sourcecode.NewWord(text, textRange)
	b.params.symbol = text
	b.params.symbolRange = textRange
	return b
}

func (b *SearchParamsBuilder) WithContextModuleName(moduleName string) *SearchParamsBuilder {
	b.params.contextModulePath = symbols.NewModulePathFromString(moduleName)
	return b
}

func (b *SearchParamsBuilder) WithContextModule(modulePath symbols.ModulePath) *SearchParamsBuilder {
	b.params.contextModulePath = modulePath
	return b
}

func (b *SearchParamsBuilder) WithTrackedModules(trackedModules TrackedModules) *SearchParamsBuilder {
	b.params.trackedModules = trackedModules
	return b
}

func (b *SearchParamsBuilder) WithDocId(docId string) *SearchParamsBuilder {
	b.params.docId = option.Some(docId)

	return b
}

func (b *SearchParamsBuilder) WithoutDoc() *SearchParamsBuilder {
	b.params.docId = option.None[string]()
	return b
}

func (b *SearchParamsBuilder) WithExcludedDocs(excludedDocId option.Option[string]) *SearchParamsBuilder {
	b.params.excludedDocId = excludedDocId

	return b
}

func (b *SearchParamsBuilder) WithScopeMode(scopeMode ScopeMode) *SearchParamsBuilder {
	b.params.scopeMode = scopeMode

	return b
}

func (b *SearchParamsBuilder) WithoutContinueOnModules() *SearchParamsBuilder {
	b.params.continueOnModules = false
	return b
}

// Otros métodos withXXX según sea necesario

// Método para construir el objeto final
func (b *SearchParamsBuilder) Build() SearchParams {

	if b.params.trackedModules == nil {
		b.params.trackedModules = make(TrackedModules)
	}

	return *b.params
}
