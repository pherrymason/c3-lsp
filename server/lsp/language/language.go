package language

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Language will be the center of knowledge of everything parsed.
type Language struct {
	index                  IndexStore
	functionTreeByDocument map[protocol.DocumentUri]indexables.Function
	logger                 commonlog.Logger
}

func NewLanguage(logger commonlog.Logger) Language {
	return Language{
		index:                  NewIndexStore(),
		functionTreeByDocument: make(map[protocol.DocumentUri]indexables.Function),
		logger:                 logger,
	}
}

func (l *Language) RefreshDocumentIdentifiers(doc *document.Document, parser *parser.Parser) {
	l.functionTreeByDocument[doc.URI] = parser.ExtractSymbols(doc)
}

func (l *Language) BuildCompletionList(doc *document.Document, position protocol.Position) []protocol.CompletionItem {
	// 1 - TODO find scoped symbols starting with same letters
	// 2 - TODO if previous character is '.', find previous symbol and if a struct, complete only with struct methods
	// 3 - TODO if writing function call arguments, complete with argument names. ¿Feasible?

	// Find symbols in document
	function := l.functionTreeByDocument[doc.URI]
	scopeSymbols := l.findAllScopeSymbols(&function, position)

	var items []protocol.CompletionItem
	for _, storedIdentifier := range scopeSymbols {
		tempKind := storedIdentifier.GetKind()

		items = append(items, protocol.CompletionItem{
			Label: storedIdentifier.GetName(),
			Kind:  &tempKind,
		})
	}

	return items
}

const (
	AnyPosition FindMode = iota
	InScope
)

type FindMode int

func (l *Language) FindSymbolDeclarationInWorkspace(doc *document.Document, position protocol.Position) (indexables.Indexable, error) {
	searchParams, err := NewSearchParamsFromPosition(doc, position)
	if err != nil {
		return indexables.Variable{}, err
	}

	symbol := l.findClosestSymbolDeclaration(searchParams)

	return symbol, nil
}

func (l *Language) FindHoverInformation(doc *document.Document, params *protocol.HoverParams) (protocol.Hover, error) {
	search, err := NewSearchParamsFromPosition(doc, params.Position)
	if err != nil {
		return protocol.Hover{}, err
	}

	foundSymbol := l.findClosestSymbolDeclaration(search)
	if foundSymbol == nil {
		return protocol.Hover{}, nil
	}

	// expected behaviour:
	// hovering on variables: display variable type + any description
	// hovering on functions: display function signature
	// hovering on members: same as variable
	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: foundSymbol.GetHoverInfo(),
		},
	}

	return hover, nil
}
