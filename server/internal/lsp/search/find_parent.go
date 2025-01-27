package search

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

// If `true`, this indexable is a type, and so one can access its parent type's (itself)
// associated members, as well as methods.
// If `false`, this indexable is a variable or similar, so its parent type is distinct
// from the indexable itself, therefore only methods can be accessed.
func canReadMembersOf(s symbols.Indexable) bool {
	switch s.(type) {

	// Theoretically, a 'distinct' cannot have members, and in fact any type obtained
	// from a distinct cannot have its members accessed, but, for consistency, we
	// specify 'true' here to indicate that we're handling a type, not some instance.
	case *symbols.Struct, *symbols.Enum, *symbols.Fault, *symbols.Distinct:
		return true
	case *symbols.Def:
		// If Def resolves to a type, it can receive its members.
		return s.(*symbols.Def).ResolvesToType()
	default:
		return false
	}
}

// Search for a method for 'parentTypeName' given a symbol to search.
//
// Returns updated search parameters to progress the search, as well as
// the search result.
func (s *Search) findMethod(
	parentTypeName string,
	searchingSymbol sourcecode.Word,
	docId option.Option[string],
	searchParams search_params.SearchParams,
	projState *project_state.ProjectState,
	debugger FindDebugger,
) (search_params.SearchParams, SearchResult) {
	// Search in methods
	methodSymbol := sourcecode.NewWord(parentTypeName+"."+searchingSymbol.Text(), searchingSymbol.TextRange())
	iterSearch := search_params.NewSearchParamsBuilder().
		WithSymbolWord(methodSymbol).
		WithDocId(docId.Get()).
		WithContextModuleName(searchParams.ModuleInCursor()).
		WithScopeMode(search_params.InModuleRoot).
		Build()

	return iterSearch, s.findClosestSymbolDeclaration(iterSearch, projState, debugger.goIn())
}

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
	membersReadable := true

	// Indicates whether the current iteration is inspecting an element which was
	// transformed from a distinct into its base type. When this happens, enum
	// and fault constants cannot be accessed; in addition, if the distinct is
	// not inline, methods also cannot be accessed.
	fromDistinct := NotFromDistinct

	for {
		if protection > 500 {
			return searchResult
		}
		protection++

		// Check for readable members before converting the element from a variable
		// to its parent type, so we can know whether we were originally searching
		// a variable, from which we cannot read members (enum values and fault
		// constants).
		membersReadable = canReadMembersOf(elm)

		// An element came from walking into a distinct only if a distinct was found
		// in the type resolution loop below.
		fromDistinct = NotFromDistinct

		// This is used when type resolution is enough to fully resolve the parent type
		// for the current access path segment. Then, we skip to the next iteration,
		// resetting temporary variables accordingly.
		skip := false

		// Resolve the element before inspecting it further.
		subprotection := 0
		for {
			if subprotection > 1000 {
				return searchResult
			}
			subprotection++

			distinct, isDistinct := elm.(*symbols.Distinct)

			if isDistinct {
				// Only check distinct methods if we don't come from another,
				// non-inline distinct, which would forbid method access.
				if fromDistinct != NonInlineDistinct {
					// Check if we could be about to access a distinct's
					// own method. If so, don't resolve it to its inner type
					// and break out of type resolution.
					searchingSymbol := state.GetNextSymbol()
					newIterSearch, methodResult := s.findMethod(
						distinct.GetName(),
						searchingSymbol,
						docId,
						searchParams,
						projState,
						debugger,
					)

					if methodResult.IsSome() {
						iterSearch = newIterSearch
						elm = methodResult.Get()
						symbolsHierarchy = append(symbolsHierarchy, elm)
						state.Advance()

						// Skip type resolution entirely, found a method.
						// Skip the iteration in order to reset iteration variables.
						skip = true
						break
					}
				}

				// Let's try to access something under its base type by resolving.
				// The base methods are only available if the distinct is inline,
				// so we record whether or not we transformed from an inline distinct
				// in a variable. Still, non-inline distincts can access associated values
				// of enums and struct members, so we must keep searching.

				// Indicate to the new element that it was transformed from
				// a distinct of a certain kind.
				if distinct.IsInline() {
					fromDistinct = InlineDistinct
				} else {
					fromDistinct = NonInlineDistinct
				}
			}

			if isDistinct || !isInspectable(elm) {
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

		if skip {
			continue
		}

		// Here we can look inside elm
		switch elm.(type) {
		case *symbols.Enumerator:
			enumerator := elm.(*symbols.Enumerator)
			assocValues := enumerator.AssociatedValues
			searchingSymbol := state.GetNextSymbol()
			foundAssoc := false
			for i := 0; i < len(assocValues); i++ {
				if assocValues[i].GetName() == searchingSymbol.Text() {
					elm = &assocValues[i]
					symbolsHierarchy = append(symbolsHierarchy, elm)
					state.Advance()
					foundAssoc = true
					break
				}
			}

			if !foundAssoc {
				if fromDistinct == NonInlineDistinct || enumerator.GetModuleString() == "" || enumerator.GetEnumName() == "" {
					// Methods inacessible from non-inline distincts that converted into this type
					// Also impossible to determine if we know nothing about the parent enum type
					return NewSearchResultEmpty(trackedModules)
				}

				// Search in methods
				// First get the enum
				enumSymbols := projState.SearchByFQN(enumerator.GetEnumFQN())
				if len(enumSymbols) > 0 {
					// Search the enum's methods
					newIterSearch, result := s.findMethod(
						enumSymbols[0].GetName(),
						searchingSymbol,
						docId,
						searchParams,
						projState,
						debugger,
					)
					if result.IsNone() {
						return NewSearchResultEmpty(trackedModules)
					}
					iterSearch = newIterSearch
					elm = result.Get()
					symbolsHierarchy = append(symbolsHierarchy, elm)
					state.Advance()
				}
			}

		case *symbols.FaultConstant:
			constant := elm.(*symbols.FaultConstant)

			if fromDistinct != NonInlineDistinct && constant.GetModuleString() != "" && constant.GetFaultName() != "" {
				// Search in methods
				// First get the fault
				faultSymbols := projState.SearchByFQN(constant.GetFaultFQN())
				if len(faultSymbols) > 0 {
					// Search the fault's methods
					searchingSymbol := state.GetNextSymbol()
					newIterSearch, result := s.findMethod(
						faultSymbols[0].GetName(),
						searchingSymbol,
						docId,
						searchParams,
						projState,
						debugger,
					)
					if result.IsNone() {
						return NewSearchResultEmpty(trackedModules)
					}
					iterSearch = newIterSearch
					elm = result.Get()
					symbolsHierarchy = append(symbolsHierarchy, elm)
					state.Advance()
				}
			} else {
				// Methods inaccessible.
				return NewSearchResultEmpty(trackedModules)
			}

		case *symbols.Enum:
			_enum := elm.(*symbols.Enum)
			foundMemberOrAssoc := false
			searchingSymbol := state.GetNextSymbol()

			// 'CoolEnum.VARIANT.VARIANT' is invalid (member not readable on member)
			// But 'CoolEnum.VARIANT' is ok, as well as 'AliasForEnum.VARIANT'.
			// However, cannot access 'DistinctEnum.VARIANT'.
			if membersReadable && fromDistinct == NotFromDistinct {
				enumerators := _enum.GetEnumerators()
				for i := 0; i < len(enumerators); i++ {
					if enumerators[i].GetName() == searchingSymbol.Text() {
						elm = enumerators[i]
						symbolsHierarchy = append(symbolsHierarchy, elm)
						state.Advance()
						foundMemberOrAssoc = true
						break
					}
				}
			} else if !membersReadable {
				// Members not readable => this is an instance, so we can read associated values.
				// This is always accessible, even when coming from distincts, so that is not checked.
				assocs := _enum.GetAssociatedValues()
				for i := 0; i < len(assocs); i++ {
					if assocs[i].GetName() == searchingSymbol.Text() {
						elm = &assocs[i]
						symbolsHierarchy = append(symbolsHierarchy, elm)
						state.Advance()
						foundMemberOrAssoc = true
						break
					}
				}
			}

			if !foundMemberOrAssoc {
				if fromDistinct == NonInlineDistinct {
					// Can't search further (methods inaccessible).
					return NewSearchResultEmpty(trackedModules)
				}

				// Search in methods
				newIterSearch, result := s.findMethod(
					_enum.GetName(),
					searchingSymbol,
					docId,
					searchParams,
					projState,
					debugger,
				)
				if result.IsNone() {
					return NewSearchResultEmpty(trackedModules)
				}
				iterSearch = newIterSearch
				elm = result.Get()
				symbolsHierarchy = append(symbolsHierarchy, elm)
				state.Advance()
			}

		case *symbols.Fault:
			fault := elm.(*symbols.Fault)
			searchingSymbol := state.GetNextSymbol()
			foundMember := false

			if membersReadable && fromDistinct == NotFromDistinct {
				constants := fault.GetConstants()
				for i := 0; i < len(constants); i++ {
					if constants[i].GetName() == searchingSymbol.Text() {
						elm = constants[i]
						symbolsHierarchy = append(symbolsHierarchy, elm)
						state.Advance()
						foundMember = true
						break
					}
				}
			}

			if !foundMember {
				if fromDistinct == NonInlineDistinct {
					// Can't search further (methods inaccessible).
					return NewSearchResultEmpty(trackedModules)
				}

				// Search in methods
				newIterSearch, result := s.findMethod(
					fault.GetName(),
					searchingSymbol,
					docId,
					searchParams,
					projState,
					debugger,
				)
				if result.IsNone() {
					return NewSearchResultEmpty(trackedModules)
				}
				iterSearch = newIterSearch
				elm = result.Get()
				symbolsHierarchy = append(symbolsHierarchy, elm)
				state.Advance()
			}

		case *symbols.Struct:
			strukt, _ := elm.(*symbols.Struct)
			members := strukt.GetMembers()
			searchingSymbol := state.GetNextSymbol()
			foundMember := false

			// Members are always readable when the parent type is struct
			// TODO: Maybe we should actually check for NOT membersReadable,
			// if anonymous substructs are found to not be usable anywhere
			// (Can't write methods for them, for example)
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
				if fromDistinct == NonInlineDistinct {
					// Can't search further (methods inaccessible).
					return NewSearchResultEmpty(trackedModules)
				}

				// Search in methods
				newIterSearch, result := s.findMethod(
					strukt.GetName(),
					searchingSymbol,
					docId,
					searchParams,
					projState,
					debugger,
				)
				if result.IsNone() {
					return NewSearchResultEmpty(trackedModules)
				}
				iterSearch = newIterSearch
				elm = result.Get()
				symbolsHierarchy = append(symbolsHierarchy, elm)
				state.Advance()
			}
		}

		if state.IsEnd() {
			break
		}
	}
	searchResult.SetMembersReadable(membersReadable)
	searchResult.SetFromDistinct(fromDistinct)
	searchResult.Set(elm)

	return searchResult
}

func isInspectable(elm symbols.Indexable) bool {
	isInspectable := true
	switch elm.(type) {
	case *symbols.Variable, *symbols.Function, *symbols.StructMember, *symbols.Def:
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
		if sm.IsStruct() {
			// This is an inline struct definition, just return it
			return sm.Substruct().Get()
		} else {
			symbol := projState.SearchByFQN(sm.GetType().GetFullQualifiedName())
			if len(symbol) > 0 {
				return symbol[0]
			} else {
				return nil
				//panic(fmt.Sprintf("Could not resolve structmember symbol: %s, with query: %s", elm.GetName(), sm.GetType().GetFullQualifiedName()))
			}
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

	case *symbols.Distinct:
		// Translate to the real symbol
		distinct := elm.(*symbols.Distinct)
		query := distinct.GetBaseType().GetFullQualifiedName()

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
