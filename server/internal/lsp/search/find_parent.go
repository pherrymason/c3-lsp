package search

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

func (s *Search) findInParentSymbols(searchParams search_params.SearchParams, projState *project_state.ProjectState, debugger FindDebugger) SearchResult {
	accessPath := searchParams.GetFullAccessPath()
	state := NewFindParentState(accessPath)
	trackedModules := searchParams.TrackTraversedModules()
	searchResult := NewSearchResult(trackedModules)
	symbolsHierarchy := []symbols.Indexable{}

	docId := searchParams.DocId()
	iterSearch := search_params.NewSearchParamsBuilder().
		WithSymbolWord(accessPath[0]).
		WithDocId(docId.Get()).
		WithContextModuleName(searchParams.ModuleInCursor()).
		WithScopeMode(search_params.InScope).
		Build()

	result := s.findClosestSymbolDeclaration(iterSearch, projState, debugger.goIn())
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
				elm = s.resolve(elm, docId.Get(), searchParams.ModuleInCursor(), projState, symbolsHierarchy, debugger)
				if elm == nil {
					return NewSearchResultEmptyWithTraversedModules(result.traversedModules)
				}
				symbolsHierarchy = append(symbolsHierarchy, elm)
			} else {
				break
			}
		}

		if state.IsEnd() {
			break
		}

		// Here we can look inside elm
		switch elm.(type) {
		case *symbols.Enumerator:
			enumerator := elm.(*symbols.Enumerator)
			assocValues := enumerator.GetAssociatedValues()
			searchingSymbol := state.GetNextSymbol()
			for i := 0; i < len(assocValues); i++ {
				if assocValues[i].GetName() == searchingSymbol.Text() {
					elm = &assocValues[i]
					symbolsHierarchy = append(symbolsHierarchy, elm)
					state.Advance()
					break
				}
			}

		case *symbols.Enum:
			_enum := elm.(*symbols.Enum)
			enumerators := _enum.GetEnumerators()
			searchingSymbol := state.GetNextSymbol()
			foundMember := false
			for i := 0; i < len(enumerators); i++ {
				if enumerators[i].GetName() == searchingSymbol.Text() {
					elm = enumerators[i]
					symbolsHierarchy = append(symbolsHierarchy, elm)
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
				result := s.findClosestSymbolDeclaration(iterSearch, projState, debugger.goIn())
				if result.IsNone() {
					return NewSearchResultEmpty(trackedModules)
				}

				elm = result.Get()
				symbolsHierarchy = append(symbolsHierarchy, elm)
				state.Advance()
			}

		case *symbols.Fault:
			_enum := elm.(*symbols.Fault)
			constants := _enum.GetConstants()
			searchingSymbol := state.GetNextSymbol()
			for i := 0; i < len(constants); i++ {
				if constants[i].GetName() == searchingSymbol.Text() {
					elm = constants[i]
					symbolsHierarchy = append(symbolsHierarchy, elm)
					state.Advance()
					break
				}
			}
		case *symbols.Struct:
			strukt, _ := elm.(*symbols.Struct)
			members := strukt.GetMembers()
			searchingSymbol := state.GetNextSymbol()
			foundMember := false
			for i := 0; i < len(members); i++ {
				if members[i].GetName() == searchingSymbol.Text() {
					elm = members[i]
					symbolsHierarchy = append(symbolsHierarchy, elm)
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
				result := s.findClosestSymbolDeclaration(iterSearch, projState, debugger.goIn())
				if result.IsNone() {
					return NewSearchResultEmpty(trackedModules)
				}

				elm = result.Get()
				symbolsHierarchy = append(symbolsHierarchy, elm)
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

func isInspectable(elm symbols.Indexable) bool {
	isInspectable := true
	switch elm.(type) {
	case *symbols.Variable:
		isInspectable = false
	case *symbols.Function:
		isInspectable = false
	case *symbols.StructMember:
		isInspectable = false
	case *symbols.Def:
		isInspectable = false
	}

	return isInspectable
}

func (l *Search) resolve(elm symbols.Indexable, docId string, moduleName string, projState *project_state.ProjectState, symbolsHierarchy []symbols.Indexable, debugger FindDebugger) symbols.Indexable {
	var symbol sourcecode.Word
	switch elm.(type) {
	case *symbols.Variable:
		variable, _ := elm.(*symbols.Variable)
		symbol = sourcecode.NewWord(variable.GetType().GetName(), variable.GetIdRange())
	case *symbols.StructMember:
		sm, _ := elm.(*symbols.StructMember)
		symbol := projState.SearchByFQN(sm.GetType().GetFullQualifiedName())
		if len(symbol) > 0 {
			return symbol[0]
		} else {
			panic(fmt.Sprintf("Could not resolve structmember symbol: %s, with query: %s", elm.GetName(), sm.GetType().GetFullQualifiedName()))
		}

	case *symbols.Function:
		fun, _ := elm.(*symbols.Function)

		returnType := fun.GetReturnType()
		_type := l.resolveType(*returnType, symbolsHierarchy, projState)

		symbol = sourcecode.NewWord(_type.GetName(), fun.GetIdRange())

	case *symbols.Def:
		// Translate to the real symbol
		def := elm.(*symbols.Def)
		var query string
		if def.ResolvesToType() {
			query = def.ResolvedType().GetFullQualifiedName()
		} else {
			// ??? This was first version of this search
			query = def.GetModuleString() + "::" + def.GetResolvesTo()
		}

		symbols := projState.SearchByFQN(query)
		if len(symbols) > 0 {
			return symbols[0]
			// Do not advance state, we need to look inside
		}
	}

	iterSearch := search_params.NewSearchParamsBuilder().
		WithSymbolWord(symbol).
		WithDocId(docId).
		WithContextModuleName(moduleName).
		WithScopeMode(search_params.InModuleRoot).
		Build()

	found := l.findClosestSymbolDeclaration(iterSearch, projState, debugger.goIn())

	if found.IsNone() {
		return nil
		//panic(fmt.Sprintf("Could not resolve symbol: %s", elm.GetName()))
	}
	return found.Get()
}

func (l *Search) resolveType(_type symbols.Type, hierarchySymbols []symbols.Indexable, projState *project_state.ProjectState) symbols.Type {
	if !_type.IsGenericArgument() {
		return _type
	}

	// This type is refering to a Generic Argument of the current module.
	// We cannot use _type.GetName() because it does not contain the real type name.
	// We need to seach up in the hierarchySymbols for the actual type "injected"

	// V1: Naive implementation: iterate hierarchySymbols in reverse, search first item with a genericArgument field and take that
	var parentType *symbols.Type
	escape := false
	for i := len(hierarchySymbols) - 1; i >= 0 && !escape; i-- {
		elm := hierarchySymbols[i]

		switch elm.(type) {
		case *symbols.StructMember:
			sm := elm.(*symbols.StructMember)
			if sm.GetType().HasGenericArguments() {
				parentType = sm.GetType()
				escape = true
			}

		case *symbols.Function:
		}
	}

	if parentType != nil {
		_type = parentType.GetGenericArgument(0)

		return _type
	}

	panic("Generic type not found")
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
