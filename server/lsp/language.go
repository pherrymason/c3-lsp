package lsp

import "C"
import (
	"errors"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type IdentifierCollection []Identifier

type Identifier struct {
	name string
	kind protocol.CompletionItemKind

	declarationPosition protocol.Position
	documentURI         protocol.DocumentUri
}

// Language will be the center of knowledge of everything parsed.
type Language struct {
	deprecatedIdentifiers []Identifier
	identifiersByDocument map[protocol.DocumentUri]IdentifierCollection
}

func NewLanguage() Language {
	return Language{
		identifiersByDocument: make(map[protocol.DocumentUri]IdentifierCollection),
	}
}

func (l *Language) RefreshDocumentIdentifiers(doc *Document) {
	// Reparse Document and find deprecatedIdentifiers
	identifiers := FindIdentifiers(doc)
	for _, id := range identifiers {
		l.registerIdentifier(doc, id)
	}
}

func (l *Language) BuildCompletionList(text string, line protocol.UInteger, character protocol.UInteger) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for _, value := range l.identifiersByDocument {
		for _, stored_identifier := range value {
			items = append(items, protocol.CompletionItem{
				Label: stored_identifier.name,
				Kind:  &stored_identifier.kind,
			})
		}
	}

	return items
}

func (l *Language) FindIdentifierDeclaration(identifier string) (Identifier, error) {
	for docIdx, value := range l.identifiersByDocument {
		docIdx = docIdx
		for _, stored_identifier := range value {
			if stored_identifier.name == identifier {
				return stored_identifier, nil
			}
		}
	}

	return Identifier{}, errors.New("no se encontró el string en el array")
}

func (l *Language) registerIdentifier(doc *Document, identifier Identifier) {
	l.identifiersByDocument[doc.URI] = append(l.identifiersByDocument[doc.URI], identifier)
}
