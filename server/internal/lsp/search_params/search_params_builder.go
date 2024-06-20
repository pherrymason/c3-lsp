package search_params

import (
	"github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
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
	return b
}

func (b *SearchParamsBuilder) WithText(text string, textRange symbols.Range) *SearchParamsBuilder {
	b.params.word = sourcecode.NewWord(text, textRange)
	return b
}

func (b *SearchParamsBuilder) WithContextModuleName(moduleName string) *SearchParamsBuilder {
	b.params.moduleInCursor = symbols.NewModulePathFromString(moduleName)
	return b
}

func (b *SearchParamsBuilder) WithContextModule(modulePath symbols.ModulePath) *SearchParamsBuilder {
	b.params.moduleInCursor = modulePath
	return b
}

func (b *SearchParamsBuilder) WithTrackedModules(trackedModules TrackedModules) *SearchParamsBuilder {
	b.params.trackedModules = trackedModules
	return b
}

func (b *SearchParamsBuilder) WithDocId(docId string) *SearchParamsBuilder {
	b.params.limitToDocId = option.Some(docId)

	return b
}

func (b *SearchParamsBuilder) WithoutDoc() *SearchParamsBuilder {
	b.params.limitToDocId = option.None[string]()
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

func (b *SearchParamsBuilder) LimitedToModule(modulePath []string) *SearchParamsBuilder {
	b.params.limitToModule = option.Some(symbols.NewModulePath(modulePath))

	return b
}

func (b *SearchParamsBuilder) LimitedToModulePath(modulePath symbols.ModulePath) *SearchParamsBuilder {
	b.params.limitToModule = option.Some(modulePath)

	return b
}

func (b *SearchParamsBuilder) Build() SearchParams {

	if b.params.trackedModules == nil {
		b.params.trackedModules = make(TrackedModules)
	}

	return *b.params
}
