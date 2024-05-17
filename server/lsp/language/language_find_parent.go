package language

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/search_params"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/option"
)

func (l *Language) findInParentSymbols(searchParams search_params.SearchParams, debugger FindDebugger) option.Option[idx.Indexable] {
	accessPath := searchParams.GetFullAccessPath()
	state := NewFindParentState(accessPath)

	docId := searchParams.DocId()
	iterSearch := search_params.NewSearchParamsBuilder().
		WithSymbol(accessPath[0].Token).
		WithSymbolRange(accessPath[0].TokenRange).
		WithDocId(docId.Get()).
		WithModule(searchParams.Module()).
		Build()

	result := l.findClosestSymbolDeclaration(iterSearch, debugger.goIn())
	if result.IsNone() {
		return result
	}

	elm := result.Get()

	for {
		for {
			if !isInspectable(elm) {
				elm = l.resolve(elm, docId.Get(), searchParams.Module(), debugger)
			} else {
				break
			}
		}

		if state.IsEnd() {
			break
		}

		// Here we can look inside elm
		switch elm.(type) {
		case *idx.Enum:
			_enum := elm.(*idx.Enum)
			enumerators := _enum.GetEnumerators()
			searchingSymbol := state.GetNextSymbol()
			for i := 0; i < len(enumerators); i++ {
				if enumerators[i].GetName() == searchingSymbol.Token {
					elm = enumerators[i]
					state.Advance()
					break
				}
			}
		case *idx.Fault:
			_enum := elm.(*idx.Fault)
			constants := _enum.GetConstants()
			searchingSymbol := state.GetNextSymbol()
			for i := 0; i < len(constants); i++ {
				if constants[i].GetName() == searchingSymbol.Token {
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
				if members[i].GetName() == searchingSymbol.Token {
					elm = members[i]
					state.Advance()
					foundMember = true
					break
				}
			}

			if !foundMember {
				// Search in methods
				methodSymbol := document.NewToken(strukt.GetName()+"."+searchingSymbol.Token, searchingSymbol.TokenRange)
				iterSearch = search_params.NewSearchParamsBuilder().
					WithSymbol(methodSymbol.Token).
					WithSymbolRange(methodSymbol.TokenRange).
					WithDocId(docId.Get()).
					WithModule(searchParams.Module()).
					Build()
				result := l.findClosestSymbolDeclaration(iterSearch, debugger.goIn())
				if result.IsNone() {
					return option.None[idx.Indexable]()
				}

				elm = result.Get()
				state.Advance()
			}
		}

		if state.IsEnd() {
			break
		}
	}

	return option.Some[idx.Indexable](elm)
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
	var symbol document.Token
	switch elm.(type) {
	case *idx.Variable:
		variable, _ := elm.(*idx.Variable)
		symbol = document.NewToken(variable.GetType().GetName(), variable.GetIdRange())
	case *idx.StructMember:
		sm, _ := elm.(*idx.StructMember)
		symbol = document.NewToken(sm.GetType().GetName(), sm.GetIdRange())
	case *idx.Function:
		fun, _ := elm.(*idx.Function)
		symbol = document.NewToken(fun.GetReturnType(), fun.GetIdRange())
	}

	iterSearch := search_params.NewSearchParamsBuilder().
		WithSymbol(symbol.Token).
		WithSymbolRange(symbol.TokenRange).
		WithDocId(docId).
		WithModule(moduleName).
		Build()

	found := l.findClosestSymbolDeclaration(iterSearch, debugger.goIn())

	return found.Get()
}

type FindParentState struct {
	currentStep int
	accessPath  []document.Token

	needsSearch bool
	nextSearch  search_params.SearchParams
}

func NewFindParentState(accessPath []document.Token) FindParentState {
	return FindParentState{
		currentStep: 0,
		accessPath:  accessPath,
		needsSearch: false,
	}
}

func (s FindParentState) GetNextSymbol() document.Token {
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
