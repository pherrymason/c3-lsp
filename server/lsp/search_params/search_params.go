package search_params

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/document/sourcecode"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/lsp/symbols_table"
	"github.com/pherrymason/c3-lsp/option"
)

const (
	Nullo int = iota
	LockStatusReady
	LockStatusLocked
)

type ScopeMode int
type TrackedModules map[string]int

const (
	AnyPosition ScopeMode = iota
	InScope
	InModuleRoot
)

// TODO: Still confusing difference between module and modulePath
// when each is used and what they really represent?
// symbolModulePath: if symbol has an implicit module path specified, this will be that. If symbol does not have any module path, this will be empty
// module: best guess of what module cursor is currently at. Currently is the last `module xxxx` found.
type SearchParams struct {
	word sourcecode.Word
	//symbol      string        // symbol to search. @DEPRECATED, use word.Text()
	//symbolRange symbols.Range // symbol start and end positions. @DEPRECATED, use word.TextRange()

	limitToDocId  option.Option[string] // limit search at document
	excludedDocId option.Option[string]

	limitToModule option.Option[symbols.ModulePath] // Limit search to module

	scopeMode ScopeMode

	moduleInCursor symbols.ModulePath // Calculated module path where symbol is located

	// Tracking values used by search functions
	trackedModules TrackedModules // Here we register what modules have been already inspected in this search. Helps avoiding infinite loops
}

func (s SearchParams) Symbol() string {
	return s.word.Text()
}

func (s SearchParams) SymbolW() sourcecode.Word {
	return s.word
}

func (s SearchParams) SymbolPosition() symbols.Position {
	return s.word.TextRange().Start
}

func (s SearchParams) ModuleInCursor() string {
	return s.ModulePathInCursor().GetName()
}

func (s SearchParams) ModulePathInCursor() symbols.ModulePath {
	return s.moduleInCursor
}

func (s SearchParams) DocId() option.Option[string] {
	return s.limitToDocId
}

func (s SearchParams) LimitSearchToDoc() bool {
	return s.limitToDocId.IsSome()
}

func (s SearchParams) LimitsSearchInModule() bool {
	return s.limitToModule.IsSome()
}

func (s SearchParams) LimitToModule() option.Option[symbols.ModulePath] {
	return s.limitToModule
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

func (s SearchParams) HasAccessPath() bool {
	return s.word.HasAccessPath()
}

func (s SearchParams) HasModuleSpecified() bool {
	return s.word.HasModulePath()
}

func (s SearchParams) IsLimitSearchInScope() bool {
	return s.scopeMode == InScope
}

func (s SearchParams) ScopeMode() ScopeMode {
	return s.scopeMode
}

func (s SearchParams) GetFullAccessPath() []sourcecode.Word {
	return s.word.GetFullAccessPath()
}

func (s SearchParams) GetFullQualifiedName() string {
	return s.word.GetFullQualifiedName()
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
func BuildSearchBySymbolUnderCursor(doc *document.Document, docParsedModules symbols_table.UnitModules, cursorPosition symbols.Position) SearchParams {
	symbolInPosition := doc.SourceCode.SymbolInPosition(cursorPosition, &docParsedModules)
	/*if symbolInPosition.IsNone() {
		panic("Could not find symbol in cursor")
	}*/

	sp := SearchParams{
		word: symbolInPosition,
		//symbol: symbolInPosition.Text(), // Deprecated
		//symbolRange: symbolInPosition.TextRange(), // Deprecated

		limitToDocId:   option.Some(doc.URI),
		moduleInCursor: symbols.NewModulePathFromString(docParsedModules.FindContextModuleInCursorPosition(cursorPosition)),

		scopeMode:      InScope,
		trackedModules: make(map[string]int),
	}

	// Check if selectedSymbol has '.' in front
	if !doc.HasPointInFrontSymbol(cursorPosition) && !doc.HasModuleSeparatorInFrontSymbol(cursorPosition) {
		return sp
	}

	if symbolInPosition.HasModulePath() || len(symbolInPosition.ResolvedModulePath()) > 0 {
		if len(symbolInPosition.ResolvedModulePath()) > 0 {
			sp.limitToModule = option.Some(symbols.NewModulePath(symbolInPosition.ResolvedModulePath()))
		} else {
			ps := []string{}
			for _, p := range symbolInPosition.ModulePath() {
				ps = append(ps, p.Text())
			}

			sp.limitToModule = option.Some(symbols.NewModulePath(ps))
		}
	}

	// TODO if sp.modulePath.IsEmpty() === false, mean that sp.module should be sp.moduelPath.String()

	return sp
}

func BuildSearchBySymbolAtModule(symbol string, symbolModule string) SearchParams {
	sp := SearchParams{
		word: sourcecode.NewWord(symbol, symbols.NewRange(0, 0, 0, 0)),
		//symbol:            symbol,
		moduleInCursor: symbols.NewModulePathFromString(symbolModule),
		trackedModules: make(TrackedModules),
	}

	return sp
}

func NewSearchParams(symbol string, symbolRange symbols.Range, symbolModule string, docId option.Option[string]) SearchParams {
	return SearchParams{
		//symbol:            symbol,
		//symbolRange:       symbolRange,
		word:           sourcecode.NewWord(symbol, symbolRange),
		moduleInCursor: symbols.NewModulePathFromString(symbolModule),
		limitToDocId:   docId,
		trackedModules: make(TrackedModules),
	}
}
