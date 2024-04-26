package language

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type SearchParams struct {
	selectedSymbol string
	parentSymbol   string // Limit search to symbols that has are child from parentSymbol
	position       protocol.Position
	docId          string
	findMode       FindMode
}

func NewSearchParams(selectedSymbol string, position protocol.Position, docId string) SearchParams {
	return SearchParams{
		selectedSymbol: selectedSymbol,
		position:       position,
		docId:          docId,
		findMode:       InPosition,
	}
}

func NewSearchParamsFromPosition(doc *document.Document, position protocol.Position) (SearchParams, error) {
	symbolInPosition, err := doc.SymbolInPosition(position)
	if err != nil {
		return SearchParams{}, err
	}
	search := NewSearchParams(symbolInPosition, position, doc.URI)

	// Check if selectedSymbol has '.' in front
	if doc.HasPointInFrontSymbol(position) {
		parentSymbol, err := doc.ParentSymbolInPosition(position)
		if err == nil {
			// We have some context information
			search.parentSymbol = parentSymbol
		}
	}

	return search, nil
}

func (s *SearchParams) HasParentSymbol() bool {
	return s.parentSymbol != ""
}
