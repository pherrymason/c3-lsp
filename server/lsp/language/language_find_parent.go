package language

import (
	"github.com/pherrymason/c3-lsp/lsp/document/sourcecode"
	"github.com/pherrymason/c3-lsp/lsp/search_params"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
)

func (l *Language) findInParentSymbols(searchParams search_params.SearchParams, debugger FindDebugger) SearchResult {
	accessPath := searchParams.GetFullAccessPath()
	state := NewFindParentState(accessPath)
	trackedModules := searchParams.TrackTraversedModules()
	searchResult := NewSearchResult(trackedModules)

	docId := searchParams.DocId()
	iterSearch := search_params.NewSearchParamsBuilder().
		WithSymbolWord(accessPath[0]).
		WithDocId(docId.Get()).
		WithContextModuleName(searchParams.ModuleInCursor()).
		WithScopeMode(search_params.InScope).
		Build()

	result := l.findClosestSymbolDeclaration(iterSearch, debugger.goIn())
	if result.IsNone() {
		return result
	}

	elm := result.Get()
	protection := 0

	for {
		if protection > 500 {
			return searchResult
		}
		protection++

		for {
			if !isInspectable(elm) {
				elm = l.resolve(elm, docId.Get(), searchParams.ModuleInCursor(), debugger)
			} else {
				break
			}
		}

		if state.IsEnd() {
			break
		}

		// Here we can look inside elm
		switch elm.(type) {
		case *idx.Def:
			// Translate to the real symbol
			def := elm.(*idx.Def)
			var query string
			if def.ResolvesToType() {
				query = def.ResolvedType().GetFullQualifiedName()
			} else {
				// ??? This was first version of this search
				query = def.GetModuleString() + "::" + def.GetResolvesTo()
			}

			symbols := l.indexByFQN.SearchByFQN(query)
			if len(symbols) > 0 {
				elm = symbols[0]
				// Do not advance state, we need to look inside
			} else {
				panic("Stumbled on an unresolvable def.")
			}

		case *idx.Enumerator:
			enumerator := elm.(*idx.Enumerator)
			assocValues := enumerator.GetAssociatedValues()
			searchingSymbol := state.GetNextSymbol()
			for i := 0; i < len(assocValues); i++ {
				if assocValues[i].GetName() == searchingSymbol.Text() {
					elm = &assocValues[i]
					state.Advance()
					break
				}
			}

		case *idx.Enum:
			_enum := elm.(*idx.Enum)
			enumerators := _enum.GetEnumerators()
			searchingSymbol := state.GetNextSymbol()
			foundMember := false
			for i := 0; i < len(enumerators); i++ {
				if enumerators[i].GetName() == searchingSymbol.Text() {
					elm = enumerators[i]
					state.Advance()
					foundMember = true
					break
				}
			}
			if !foundMember {
				// Search in methods
				methodSymbol := sourcecode.NewWord(_enum.GetName()+"."+searchingSymbol.Text(), searchingSymbol.TextRange())
				iterSearch = search_params.NewSearchParamsBuilder().
					WithSymbolWord(methodSymbol).
					WithDocId(docId.Get()).
					WithContextModuleName(searchParams.ModuleInCursor()).
					WithScopeMode(search_params.InModuleRoot).
					Build()
				result := l.findClosestSymbolDeclaration(iterSearch, debugger.goIn())
				if result.IsNone() {
					return NewSearchResultEmpty(trackedModules)
				}

				elm = result.Get()
				state.Advance()
			}

		case *idx.Fault:
			_enum := elm.(*idx.Fault)
			constants := _enum.GetConstants()
			searchingSymbol := state.GetNextSymbol()
			for i := 0; i < len(constants); i++ {
				if constants[i].GetName() == searchingSymbol.Text() {
					elm = constants[i]
					state.Advance()
					break
				}
			}
		case *idx.Struct:
			strukt, _ := elm.(*idx.Struct)
			members := strukt.GetMembers()
			searchingSymbol := state.GetNextSymbol()
			foundMember := false
			for i := 0; i < len(members); i++ {
				if members[i].GetName() == searchingSymbol.Text() {
					elm = members[i]
					state.Advance()
					foundMember = true
					break
				}
			}

			if !foundMember {
				// Search in methods
				methodSymbol := sourcecode.NewWord(strukt.GetName()+"."+searchingSymbol.Text(), searchingSymbol.TextRange())
				iterSearch = search_params.NewSearchParamsBuilder().
					WithSymbolWord(methodSymbol).
					WithDocId(docId.Get()).
					WithContextModuleName(searchParams.ModuleInCursor()).
					WithScopeMode(search_params.InModuleRoot).
					Build()
				result := l.findClosestSymbolDeclaration(iterSearch, debugger.goIn())
				if result.IsNone() {
					return NewSearchResultEmpty(trackedModules)
				}

				elm = result.Get()
				state.Advance()
			}
		}

		if state.IsEnd() {
			break
		}
	}
	searchResult.Set(elm)

	return searchResult
}

func isInspectable(elm idx.Indexable) bool {
	isInspectable := true
	switch elm.(type) {
	case *idx.Variable:
		isInspectable = false
	case *idx.Function:
		isInspectable = false
	case *idx.StructMember:
		isInspectable = false
	}

	return isInspectable
}

func (l *Language) resolve(elm idx.Indexable, docId string, moduleName string, debugger FindDebugger) idx.Indexable {
	var symbol sourcecode.Word
	switch elm.(type) {
	case *idx.Variable:
		variable, _ := elm.(*idx.Variable)
		symbol = sourcecode.NewWord(variable.GetType().GetName(), variable.GetIdRange())
	case *idx.StructMember:
		sm, _ := elm.(*idx.StructMember)
		symbol := l.indexByFQN.SearchByFQN(sm.GetType().GetFullQualifiedName())
		if len(symbol) > 0 {
			return symbol[0]
		}

	case *idx.Function:
		fun, _ := elm.(*idx.Function)
		symbol = sourcecode.NewWord(fun.GetReturnType(), fun.GetIdRange())
	}

	iterSearch := search_params.NewSearchParamsBuilder().
		WithSymbolWord(symbol).
		WithDocId(docId).
		WithContextModuleName(moduleName).
		WithScopeMode(search_params.InModuleRoot).
		Build()

	found := l.findClosestSymbolDeclaration(iterSearch, debugger.goIn())

	return found.Get()
}

type FindParentState struct {
	currentStep int
	accessPath  []sourcecode.Word

	needsSearch bool
	nextSearch  search_params.SearchParams
}

func NewFindParentState(accessPath []sourcecode.Word) FindParentState {
	return FindParentState{
		currentStep: 0,
		accessPath:  accessPath,
		needsSearch: false,
	}
}

func (s FindParentState) GetNextSymbol() sourcecode.Word {
	return s.accessPath[s.currentStep+1]
}

func (s FindParentState) CurrentStep() int {
	return s.currentStep
}

func (s *FindParentState) Advance() {
	if s.currentStep < (len(s.accessPath) - 1) {
		s.currentStep++
	}
}

func (s FindParentState) IsEnd() bool {
	return s.currentStep >= (len(s.accessPath) - 1)
}
