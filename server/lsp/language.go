package lsp

import "C"
import (
	"errors"
	"fmt"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Language will be the center of knowledge of everything parsed.
type Language struct {
	indexablesByDocument   map[protocol.DocumentUri]indexables.IndexableCollection
	functionTreeByDocument map[protocol.DocumentUri]indexables.Function
}

func NewLanguage() Language {
	return Language{
		indexablesByDocument:   make(map[protocol.DocumentUri]indexables.IndexableCollection),
		functionTreeByDocument: make(map[protocol.DocumentUri]indexables.Function),
	}
}

func (l *Language) RefreshDocumentIdentifiers(doc *Document) {
	l.functionTreeByDocument[doc.URI] = FindSymbols(doc)
}

func (l *Language) BuildCompletionList(text string, line protocol.UInteger, character protocol.UInteger) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for _, value := range l.indexablesByDocument {
		for _, storedIdentifier := range value {
			tempKind := storedIdentifier.GetKind()

			items = append(items, protocol.CompletionItem{
				Label: storedIdentifier.GetName(),
				Kind:  &tempKind,
			})
		}
	}

	return items
}

func (l *Language) registerIndexable(doc *Document, indexable indexables.Indexable) {
	l.indexablesByDocument[doc.URI] = append(l.indexablesByDocument[doc.URI], indexable)
}

const (
	AnyPosition FindMode = iota
	InPosition
)

type FindMode int

func (l *Language) FindSymbolDeclarationInWorkspace(docId protocol.DocumentUri, identifier string, position protocol.Position) (indexables.Indexable, error) {

	symbol := l.findClosestSymbolDeclaration(identifier, docId, position)

	return symbol, nil
}

func (l *Language) FindHoverInformation(doc *Document, params *protocol.HoverParams) (protocol.Hover, error) {
	word, err := doc.WordInPosition(params.Position)
	if err != nil {
		return protocol.Hover{}, err
	}

	identifier := l.findClosestSymbolDeclaration(word, params.TextDocument.URI, params.Position)

	// expected behaviour:
	// hovering on variables: display variable type + any description
	// hovering on functions: display function signature
	// hovering on members: same as variable
	var hover protocol.Hover
	switch v := identifier.(type) {
	case indexables.Variable:
		hover = protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: fmt.Sprintf("%s %s", v.GetType(), v.GetName()),
			},
		}

	case *indexables.Function:
		hover = protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: fmt.Sprintf("%s %s()", v.ReturnType, v.GetName()),
			},
		}
	case *indexables.Struct:
	default:
	}

	return hover, nil
}

// Finds the closest symbol based on current scope.
// If not present in current Scope:
// - Search in imported files (TODO)
// - Search in global symbols in workspace
func (l *Language) findClosestSymbolDeclaration(word string, docId protocol.DocumentUri, position protocol.Position) indexables.Indexable {
	identifier, _ := l.findSymbolDeclarationInDocPositionScope(word, docId, position)
	if identifier != nil {
		return identifier
	}

	// TODO search in imported files in docId
	// -----------

	// Not found yet, let's try search the symbol defined as global in other files
	for _, scope := range l.functionTreeByDocument {
		found, foundDepth := findDeepFirst(word, position, &scope, 0, AnyPosition)

		if found != nil && (foundDepth <= 1) {
			return found
		}
	}

	// Not found...
	return nil
}

// Search for symbol in docId
func (l *Language) findSymbolDeclarationInDocPositionScope(identifier string, docId protocol.DocumentUri, position protocol.Position) (indexables.Indexable, error) {
	scopedTree, ok := l.functionTreeByDocument[docId]
	if !ok {
		return nil, errors.New("Document is not indexed")
	}

	// Go through every element defined in scopedTree
	symbol, _ := findDeepFirst(identifier, position, &scopedTree, 0, InPosition)
	return symbol, nil
}

func findDeepFirst(identifier string, position protocol.Position, function *indexables.Function, depth uint, mode FindMode) (indexables.Indexable, uint) {
	if mode == InPosition && !isPositionInsideRange(position, function.GetDocumentRange()) {
		return nil, depth
	}

	if identifier == function.Name {
		return function, depth
	}

	variable, foundInThisScope := function.Variables[identifier]

	for _, child := range function.ChildrenFunctions {
		if result, resultDepth := findDeepFirst(identifier, position, child, depth+1, mode); result != nil {
			return result, resultDepth
		}
	}

	if foundInThisScope {
		return variable, depth
	}

	return nil, depth
}

func isPositionInsideRange(position protocol.Position, rango protocol.Range) bool {
	if position.Line >= rango.Start.Line && position.Line <= rango.End.Line {
		// Exactly same line
		if position.Line == rango.Start.Line && position.Line == rango.End.Line {
			// Must be inside character ranges
			if position.Character >= rango.Start.Character && position.Character <= rango.End.Character {
				return true
			}
		} else {
			return true
		}
	}

	return false
}
