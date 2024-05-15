package search_params

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/option"
)

const (
	Nullo int = iota
	LockStatusReady
	LockStatusLocked
)

type FindMode int
type TrackedModules map[string]int

const (
	AnyPosition FindMode = iota
	InScope
)

// TODO: Still confusing difference between module and modulePath
// when each is used and what they really represent?
// symbolModulePath: if symbol has an implicit module path specified, this will be that. If symbol does not have any module path, this will be empty
// module: best guess of what module cursor is currently at. Currently is the last `module xxxx` found.
type SearchParams struct {
	symbol      string        // symbol to search
	symbolRange symbols.Range // symbol start and end positions
	//module      symbols.ModulePath // evaluated module of symbol
	moduleSpecified bool // Symbol has module specified explicitly

	docId         option.Option[string] // limit search at document
	excludedDocId option.Option[string]

	continueOnModules bool // Allow searching on module locate on other files

	scopeMode FindMode

	// __ vv Here collected info abot symbol vv __
	symbolModulePath symbols.ModulePath // Calculated module path where symbol is located
	parentAccessPath []document.Token   // if symbol belongs to a parent hierarchy call, here will lie parent symbols

	// Tracking values used by search functions
	trackedModules TrackedModules // Here we register what modules have been already inspected in this search. Helps avoiding infinite loops
}

func (s SearchParams) Symbol() string {
	return s.symbol
}

func (s SearchParams) SymbolPosition() symbols.Position {
	return s.symbolRange.Start
}

func (s SearchParams) Module() string {
	return s.ModulePath().GetName()
}

func (s SearchParams) ModulePath() symbols.ModulePath {
	return s.symbolModulePath
}

func (s SearchParams) DocId() option.Option[string] {
	return s.docId
}

func (s SearchParams) TrackedModules() TrackedModules {
	return s.trackedModules
}

func (s SearchParams) ShouldExcludeDocId(docId string) bool {
	if s.excludedDocId.IsNone() {
		return false
	}

	if s.excludedDocId.Get() == docId {
		return true
	}

	return false
}

func (s SearchParams) ContinueOnModules() bool {
	return s.continueOnModules
}

func (s SearchParams) HasAccessPath() bool {
	return len(s.parentAccessPath) > 0
}

func (s SearchParams) HasModuleSpecified() bool {
	return !s.symbolModulePath.IsEmpty()
}

func (s SearchParams) IsLimitSearchInScope() bool {
	return s.scopeMode == InScope
}

func (s SearchParams) GetFullAccessPath() []document.Token {
	tokens := append(
		s.parentAccessPath,
		document.NewToken(s.symbol, s.symbolRange),
	)

	return tokens
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

func (s SearchParams) TrackTraversedModules() map[string]int {
	return s.trackedModules
}

// Creates a SearchParam to search by symbol located at a given position in document.
// This calculates the module cursor is located.
func BuildSearchBySymbolUnderCursor(doc *document.Document, docParsedModules parser.ParsedModulesInterface, cursorPosition symbols.Position) SearchParams {
	symbolInPosition := doc.SymbolInPosition2(cursorPosition)
	if symbolInPosition.IsNone() {
		panic("Could not find symbol in cursor")
	}

	sp := SearchParams{
		symbol:           symbolInPosition.Get().Token,
		symbolRange:      symbolInPosition.Get().TokenRange,
		docId:            option.Some(doc.URI),
		symbolModulePath: symbols.NewModulePathFromString(docParsedModules.FindModuleInCursorPosition(cursorPosition)),

		continueOnModules: true,
		scopeMode:         InScope,
		trackedModules:    make(map[string]int),
	}

	// Check if selectedSymbol has '.' in front
	if !doc.HasPointInFrontSymbol(cursorPosition) && !doc.HasModuleSeparatorInFrontSymbol(cursorPosition) {
		return sp
	}

	symbolModulePath, parentAccessPath := findParentSymbols(doc, cursorPosition)
	if symbolModulePath.IsEmpty() == false {
		sp.moduleSpecified = true
		sp.symbolModulePath = symbolModulePath
	}
	sp.parentAccessPath = parentAccessPath

	// TODO if sp.modulePath.IsEmpty() === false, mean that sp.module should be sp.moduelPath.String()

	return sp
}

func findParentSymbols(doc *document.Document, cursorPosition symbols.Position) (symbols.ModulePath, []document.Token) {
	var modulePath symbols.ModulePath
	var parentAccessPath []document.Token
	positionStart, _ := doc.GetSymbolPositionAtPosition(cursorPosition)
	// Iterate backwards from the cursor position to find all parent symbols
	iterating_module_path := false

	for i := int(positionStart.Character - 1); i >= 0; i-- {
		positionStart = symbols.Position{
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
			modulePath.AddPath(parentSymbol.Token)
			positionStart, _ := doc.GetSymbolPositionAtPosition(positionStart)
			i = int(positionStart.Character)
		} else {
			positionStart, _ := doc.GetSymbolPositionAtPosition(positionStart)
			parentAccessPath = append([]document.Token{
				parentSymbol,
			}, parentAccessPath...)

			if doc.HasPointInFrontSymbol(positionStart) {
				i = int(positionStart.Character) - 1
			} else {
				break
			}
		}
	}

	return modulePath, parentAccessPath
}

func BuildSearchBySymbolAtModule(symbol string, symbolModule string) SearchParams {
	sp := SearchParams{
		symbol:           symbol,
		symbolModulePath: symbols.NewModulePathFromString(symbolModule),
		trackedModules:   make(TrackedModules),
	}

	return sp
}

func NewSearchParams(symbol string, symbolRange symbols.Range, symbolModule string, docId option.Option[string]) SearchParams {
	return SearchParams{
		symbol:           symbol,
		symbolRange:      symbolRange,
		symbolModulePath: symbols.NewModulePathFromString(symbolModule),
		docId:            docId,
		trackedModules:   make(TrackedModules),
	}
}
