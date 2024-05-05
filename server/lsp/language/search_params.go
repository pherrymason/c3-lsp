package language

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const (
	Nullo int = iota
	Ready
	Lock
)

type SearchParams struct {
	selectedSymbol document.Token
	parentSymbols  []document.Token // Limit search to symbols that has are child from parentSymbol
	docId          string
	modulePath     indexables.ModulePath

	continueOnModules bool
	scopeMode         FindMode // TODO Rename this to boolean

	trackedModules map[string]int // Here we register what modules have been already inspected in this search. Helps avoiding infinite loops
}

/*
func NewSearchParams(selectedSymbol string, position protocol.Position, docId string) SearchParams {
	return SearchParams{
		selectedSymbol:    document.NewToken(selectedSymbol, position),
		docId:             docId,
		scopeMode:         InScope,
		continueOnModules: true,
		trackedModules:    make(map[string]int),
	}
}*/

func NewSearchParamsFromToken(selectedSymbol document.Token, docId string) SearchParams {
	return SearchParams{
		selectedSymbol:    selectedSymbol,
		docId:             docId,
		scopeMode:         InScope,
		continueOnModules: true,
		trackedModules:    make(map[string]int),
	}
}

func NewSearchParamsFromPosition(doc *document.Document, cursorPosition protocol.Position) (SearchParams, error) {
	symbolInPosition, err := doc.SymbolInPosition(cursorPosition)
	if err != nil {
		return SearchParams{}, err
	}

	search := NewSearchParamsFromToken(symbolInPosition, doc.URI)

	// Check if selectedSymbol has '.' in front
	if !doc.HasPointInFrontSymbol(cursorPosition) && !doc.HasModuleSeparatorInFrontSymbol(cursorPosition) {
		return search, nil
	}

	positionStart, _ := doc.GetSymbolPositionAtPosition(cursorPosition)

	// Iterate backwards from the cursor position to find all parent symbols
	iterating_module_path := false

	for i := int(positionStart.Character - 1); i >= 0; i-- {
		positionStart = indexables.Position{
			Line:      uint(cursorPosition.Line),
			Character: uint(i),
		}
		parentSymbol, err := doc.SymbolInPosition(positionStart.ToLSPPosition())
		if err != nil {
			// No symbol found, check was is in parentSymbol anyway
			if parentSymbol.Token == "." {

			} else if parentSymbol.Token == ":" {
				iterating_module_path = true
			} else if parentSymbol.Token == " " {
				break
			}
			continue
		}

		if iterating_module_path {
			search.modulePath.AddPath(parentSymbol.Token)
			positionStart, _ := doc.GetSymbolPositionAtPosition(positionStart.ToLSPPosition())
			i = int(positionStart.Character)
		} else {
			positionStart, _ := doc.GetSymbolPositionAtPosition(positionStart.ToLSPPosition())
			search.parentSymbols = append([]document.Token{
				parentSymbol,
			}, search.parentSymbols...)

			if doc.HasPointInFrontSymbol(positionStart.ToLSPPosition()) {
				i = int(positionStart.Character) - 1
			} else {
				break
			}
		}
	}

	return search, nil
}

func (s SearchParams) HasParentSymbol() bool {
	return len(s.parentSymbols) > 0
}

func (s SearchParams) HasModuleSpecified() bool {
	return s.modulePath.Has()
}

func (s *SearchParams) RegisterTraversedModule(module string) {
	//s.traversedModules = append(s.traversedModules, module)
}
