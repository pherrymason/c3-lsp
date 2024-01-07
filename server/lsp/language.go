package lsp

import "C"
import (
	"errors"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Indexable interface {
	GetName() string
	GetKind() protocol.CompletionItemKind
	GetDocumentURI() protocol.DocumentUri
	GetDeclarationPosition() protocol.Position
}

type IndexableCollection []Indexable

// Language will be the center of knowledge of everything parsed.
type Language struct {
	indexablesByDocument map[protocol.DocumentUri]IndexableCollection
}

func NewLanguage() Language {
	return Language{
		indexablesByDocument: make(map[protocol.DocumentUri]IndexableCollection),
	}
}

func (l *Language) RefreshDocumentIdentifiers(doc *Document) {
	// Reparse Document and find deprecatedIdentifiers
	identifiers := FindIdentifiers(doc)
	for _, id := range identifiers {
		l.registerIndexable(doc, id)
	}
}

func (l *Language) BuildCompletionList(text string, line protocol.UInteger, character protocol.UInteger) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for _, value := range l.indexablesByDocument {
		for _, stored_identifier := range value {
			tempKind := stored_identifier.GetKind()

			items = append(items, protocol.CompletionItem{
				Label: stored_identifier.GetName(),
				Kind:  &tempKind,
			})
		}
	}

	return items
}

func (l *Language) FindIdentifierDeclaration(identifier string) (Indexable, error) {
	for docIdx, value := range l.indexablesByDocument {
		docIdx = docIdx
		for _, stored_identifier := range value {
			if stored_identifier.GetName() == identifier {
				return stored_identifier, nil
			}
		}
	}

	return nil, errors.New("no se encontró el string en el array")
}

func (l *Language) registerIndexable(doc *Document, indexable Indexable) {
	l.indexablesByDocument[doc.URI] = append(l.indexablesByDocument[doc.URI], indexable)
}

func (l *Language) FindHoverInformation(identifier string) string {
	// expected behaviour:
	// hovering on variables: display variable type + any description
	// hovering on functions: display function signature
	// hovering on members: same as variable

	return "not **found**"
}
