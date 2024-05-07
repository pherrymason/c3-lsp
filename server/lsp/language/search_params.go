package language

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
)

const (
	Nullo int = iota
	LockStatusReady
	LockStatusLocked
)

type LockStatus int

const (
	AnyPosition FindMode = iota
	InScope
)

type FindMode int

type SearchParams struct {
	// Values setable by developer
	docId             string // Document from where we start searching
	continueOnModules bool
	scopeMode         FindMode // TODO Rename this to boolean

	// Values automatically calculated from cursor position
	selectedToken document.Token   // token (textual symbol) currently under the cursor
	parentSymbols []document.Token // Limit search to symbols that has are child from parentSymbol
	modulePath    indexables.ModulePath

	// Tracking values used by search functions
	trackedModules map[string]int // Here we register what modules have been already inspected in this search. Helps avoiding infinite loops
}

func NewSearchParamsFromToken(selectedToken document.Token, docId string) SearchParams {
	return SearchParams{
		selectedToken:     selectedToken,
		docId:             docId,
		scopeMode:         InScope,
		continueOnModules: true,
		trackedModules:    make(map[string]int),
	}
}

func NewSearchParamsFromPosition(doc *document.Document, cursorPosition indexables.Position) (SearchParams, error) {
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
		parentSymbol, err := doc.SymbolInPosition(positionStart)
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
			positionStart, _ := doc.GetSymbolPositionAtPosition(positionStart)
			i = int(positionStart.Character)
		} else {
			positionStart, _ := doc.GetSymbolPositionAtPosition(positionStart)
			search.parentSymbols = append([]document.Token{
				parentSymbol,
			}, search.parentSymbols...)

			if doc.HasPointInFrontSymbol(positionStart) {
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

func (s *SearchParams) TrackTraversedModule(module string) bool {
	mt, ok := s.trackedModules[module]
	trackValue := LockStatusReady
	if ok && mt == LockStatusLocked {
		return false
	} else if mt == LockStatusReady {
		trackValue = LockStatusLocked
	}
	s.trackedModules[module] = trackValue

	return true
}
