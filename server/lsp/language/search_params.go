package language

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Token struct {
	token    string
	position protocol.Position
}

type SearchParams struct {
	selectedSymbol Token
	parentSymbols  []Token // Limit search to symbols that has are child from parentSymbol
	//position       protocol.Position
	docId    string
	findMode FindMode
}

func NewSearchParams(selectedSymbol string, position protocol.Position, docId string) SearchParams {
	return SearchParams{
		selectedSymbol: Token{token: selectedSymbol, position: position},
		docId:          docId,
		findMode:       InPosition,
	}
}

func NewSearchParamsFromPosition(doc *document.Document, cursorPosition protocol.Position) (SearchParams, error) {
	symbolInPosition, err := doc.SymbolInPosition(cursorPosition)
	if err != nil {
		return SearchParams{}, err
	}

	search := NewSearchParams(symbolInPosition, cursorPosition, doc.URI)

	// Check if selectedSymbol has '.' in front
	if doc.HasPointInFrontSymbol(cursorPosition) {
		positionStart, _ := doc.GetSymbolPositionAtPosition(cursorPosition)

		// Iterate backwards from the cursor position to find all parent symbols
		for i := positionStart.Character - 2; i >= 0; i-- {
			positionStart = protocol.Position{Line: cursorPosition.Line, Character: i}
			parentSymbol, err := doc.SymbolInPosition(positionStart)
			if err == nil && parentSymbol != "." {
				// If a non-dot symbol is found, add it to the parentSymbols list
				positionStart, _ := doc.GetSymbolPositionAtPosition(positionStart)
				search.parentSymbols = append(search.parentSymbols, Token{token: parentSymbol, position: positionStart})

				if doc.HasPointInFrontSymbol(positionStart) {
					i = positionStart.Character - 2
				} else {
					break
				}
			} else {
				// If a dot symbol or an error is encountered, stop iterating
				break
			}
		}
	}

	return search, nil
}

func (s *SearchParams) HasParentSymbol() bool {
	return len(s.parentSymbols) > 0
}
