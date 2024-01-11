package indexables

import "github.com/tliron/glsp/protocol_3_16"

type Indexable interface {
	GetName() string
	GetKind() protocol.CompletionItemKind
	GetDocumentURI() protocol.DocumentUri
	GetDeclarationRange() protocol.Range
	GetDocumentRange() protocol.Range
}

type IndexableCollection []Indexable

type BaseIndexable struct {
	documentURI     protocol.DocumentUri
	identifierRange protocol.Range
	documentRange   protocol.Range
	Kind            protocol.CompletionItemKind
}

func NewBaseIndexable(docId protocol.DocumentUri, idRange protocol.Range, docRange protocol.Range, kind protocol.CompletionItemKind) BaseIndexable {
	return BaseIndexable{
		documentURI:     docId,
		identifierRange: idRange,
		documentRange:   docRange,
		Kind:            kind,
	}
}
