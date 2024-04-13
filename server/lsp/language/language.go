package language

import (
	"errors"
	"fmt"
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pherrymason/c3-lsp/lsp/parser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Language will be the center of knowledge of everything parsed.
type Language struct {
	index                  IndexStore
	functionTreeByDocument map[protocol.DocumentUri]indexables.Function
}

func NewLanguage() Language {
	return Language{
		index:                  NewIndexStore(),
		functionTreeByDocument: make(map[protocol.DocumentUri]indexables.Function),
	}
}

func (l *Language) RefreshDocumentIdentifiers(doc *document.Document, parser *parser.Parser) {
	l.functionTreeByDocument[doc.URI] = parser.ExtractSymbols(doc)
}

func (l *Language) BuildCompletionList(doc *document.Document, position protocol.Position) []protocol.CompletionItem {
	// 1 - TODO find scoped symbols starting with same letters
	// 2 - TODO if previous character is '.', find previous symbol and if a struct, complete only with struct methods
	// 3 - TODO if writing function call arguments, complete with argument names. ¿Feasible?

	// Find symbols in document
	function := l.functionTreeByDocument[doc.URI]
	scopeSymbols := l.findAllScopeSymbols(&function, position)

	var items []protocol.CompletionItem
	for _, storedIdentifier := range scopeSymbols {
		tempKind := storedIdentifier.GetKind()

		items = append(items, protocol.CompletionItem{
			Label: storedIdentifier.GetName(),
			Kind:  &tempKind,
		})
	}

	return items
}

func (l *Language) findAllScopeSymbols(scopeFunction *indexables.Function, position protocol.Position) []indexables.Indexable {
	var symbols []indexables.Indexable

	for _, variable := range scopeFunction.Variables {
		symbols = append(symbols, variable)
	}
	for _, enum := range scopeFunction.Enums {
		symbols = append(symbols, enum)
	}
	for _, strukt := range scopeFunction.Structs {
		symbols = append(symbols, strukt)
	}
	for _, def := range scopeFunction.Defs {
		symbols = append(symbols, def)
	}

	for _, function := range scopeFunction.ChildrenFunctions {
		if function.GetDocumentRange().HasPosition(position) {
			symbols = append(symbols, function)
		}
	}

	return symbols
}

const (
	AnyPosition FindMode = iota
	InPosition
)

type FindMode int

func buildSearchParams(doc *document.Document, position protocol.Position) (SearchParams, error) {
	symbolInPosition, err := doc.SymbolInPosition(position)
	if err != nil {
		return SearchParams{}, err
	}
	search := NewSearchParams(symbolInPosition, doc.URI)

	// Check if selectedSymbol has '.' in front
	if doc.HasPointInFrontSymbol(position) {
		parentSymbol, err := doc.ParentSymbolInPosition(position)
		if err == nil {
			// We have some context information
			search.parentSymbol = parentSymbol
		}
	}

	return search, nil
}

func (l *Language) FindSymbolDeclarationInWorkspace(doc *document.Document, position protocol.Position) (indexables.Indexable, error) {
	searchParams, err := buildSearchParams(doc, position)
	if err != nil {
		return indexables.Variable{}, err
	}

	symbol := l.findClosestSymbolDeclaration(searchParams, position)

	return symbol, nil
}

func (l *Language) FindHoverInformation(doc *document.Document, params *protocol.HoverParams) (protocol.Hover, error) {
	search, err := buildSearchParams(doc, params.Position)
	if err != nil {
		return protocol.Hover{}, err
	}

	foundSymbol := l.findClosestSymbolDeclaration(search, params.Position)
	if foundSymbol == nil {
		return protocol.Hover{}, nil
	}

	// expected behaviour:
	// hovering on variables: display variable type + any description
	// hovering on functions: display function signature
	// hovering on members: same as variable
	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: foundSymbol.GetHoverInfo(),
		},
	}

	return hover, nil
}

// Finds the closest selectedSymbol based on current scope.
// If not present in current Scope:
// - SearchParams in imported files (TODO)
// - SearchParams in global symbols in workspace
func (l *Language) findClosestSymbolDeclaration(searchParams SearchParams, position protocol.Position) indexables.Indexable {

	var parentIdentifier indexables.Indexable
	// Check if there's parent contextual information in searchParams
	if searchParams.HasParentSymbol() {
		subSearchParams := NewSearchParams(searchParams.parentSymbol, searchParams.docId)
		parentIdentifier = l.findClosestSymbolDeclaration(subSearchParams, position)
	}

	if parentIdentifier != nil {
		//fmt.Printf("Parent found")
		// If parent is Variable -> Look for variable.Type
		switch parentIdentifier.(type) {
		case indexables.Variable:
			variable, ok := parentIdentifier.(indexables.Variable)
			if !ok {
				panic("Error")
			}
			sp := NewSearchParams(variable.Type, variable.GetDocumentURI())
			variableTypeSymbol := l.findClosestSymbolDeclaration(sp, position)
			//	fmt.Sprint(variableTypeSymbol.GetName())
			searchParams.selectedSymbol = variableTypeSymbol.GetName() + "." + searchParams.selectedSymbol
		}
	}

	identifier, _ := l.findSymbolDeclarationInDocPositionScope(searchParams, position)

	if identifier != nil {
		return identifier
	}

	// TODO search in imported files in docId
	// -----------

	// Not found yet, let's try search the selectedSymbol defined as global in other files
	// Note: Iterating a map is not guaranteed to be done always in the same order.
	for _, scope := range l.functionTreeByDocument {
		found, foundDepth := findDeepFirst(searchParams.selectedSymbol, position, &scope, 0, AnyPosition)

		if found != nil && (foundDepth <= 1) {
			return found
		}
	}

	// Not found...
	return nil
}

// SearchParams for selectedSymbol in docId
func (l *Language) findSymbolDeclarationInDocPositionScope(searchParams SearchParams, position protocol.Position) (indexables.Indexable, error) {
	scopedTree, found := l.functionTreeByDocument[searchParams.docId]
	if !found {
		return nil, errors.New(fmt.Sprint("Skipping as no symbols for ", searchParams.docId, " were indexed."))
	}

	// Go through every element defined in scopedTree
	symbol, _ := findDeepFirst(searchParams.selectedSymbol, position, &scopedTree, 0, InPosition)
	return symbol, nil
}

func findDeepFirst(identifier string, position protocol.Position, function *indexables.Function, depth uint, mode FindMode) (indexables.Indexable, uint) {
	if mode == InPosition &&
		!function.GetDocumentRange().HasPosition(position) {
		return nil, depth
	}

	if identifier == function.GetFullName() {
		return function, depth
	}

	for _, child := range function.ChildrenFunctions {
		if result, resultDepth := findDeepFirst(identifier, position, &child, depth+1, mode); result != nil {
			return result, resultDepth
		}
	}

	variable, foundVariableInThisScope := function.Variables[identifier]
	if foundVariableInThisScope {
		return variable, depth
	}

	enum, foundEnumInThisScope := function.Enums[identifier]
	if foundEnumInThisScope {
		return enum, depth
	}

	var enumerator indexables.Enumerator
	foundEnumeratorInThisScope := false
	for _, scopedEnums := range function.Enums {
		if scopedEnums.HasEnumerator(identifier) {
			enumerator = scopedEnums.GetEnumerator(identifier)
			foundEnumeratorInThisScope = true
		}
	}
	if foundEnumeratorInThisScope {
		return enumerator, depth
	}

	_struct, foundStructInThisScope := function.Structs[identifier]
	if foundStructInThisScope {
		return _struct, depth
	}

	def, foundDefInScope := function.Defs[identifier]
	if foundDefInScope {
		return def, depth
	}

	return nil, depth
}
