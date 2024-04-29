package language

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Token struct {
	token    string
	position protocol.Position
}

type SearchParams struct {
	selectedSymbol Token
	parentSymbols  []Token // Limit search to symbols that has are child from parentSymbol
	docId          string
	modulePath     indexables.ModulePath
	findMode       FindMode
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
	if !doc.HasPointInFrontSymbol(cursorPosition) && !doc.HasModuleSeparatorInFrontSymbol(cursorPosition) {
		return search, nil
	}

	positionStart, _ := doc.GetSymbolPositionAtPosition(cursorPosition)

	// Iterate backwards from the cursor position to find all parent symbols
	iterating_module_path := false

	for i := int(positionStart.Character - 1); i >= 0; i-- {
		positionStart = protocol.Position{Line: cursorPosition.Line, Character: protocol.UInteger(i)}
		parentSymbol, err := doc.SymbolInPosition(positionStart)
		if err != nil {
			// No symbol found, check was is in parentSymbol anyway
			if parentSymbol == "." {

			} else if parentSymbol == ":" {
				iterating_module_path = true
			} else if parentSymbol == " " {
				break
			}
			continue
		}

		if iterating_module_path {
			search.modulePath.AddPath(parentSymbol)
			positionStart, _ := doc.GetSymbolPositionAtPosition(positionStart)
			i = int(positionStart.Character)
		} else {
			positionStart, _ := doc.GetSymbolPositionAtPosition(positionStart)
			search.parentSymbols = append(search.parentSymbols, Token{token: parentSymbol, position: positionStart})

			if doc.HasPointInFrontSymbol(positionStart) {
				i = int(positionStart.Character) - 1
			} else {
				break
			}
		}
	}

	return search, nil
}

func (s *SearchParams) HasParentSymbol() bool {
	return len(s.parentSymbols) > 0
}

func (s SearchParams) HasModuleSpecified() bool {
	return s.modulePath.Has()
}
