package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type Struct struct {
	Name            string
	Members         []string
	DocumentURI     string
	identifierRange Range
	documentRange   Range
	Kind            protocol.CompletionItemKind
}

func (s Struct) GetName() string {
	return s.Name
}

func (s Struct) GetKind() protocol.CompletionItemKind {
	return s.Kind
}

func (s Struct) GetDocumentURI() string {
	return s.DocumentURI
}

func (s Struct) GetDeclarationRange() Range {
	return s.identifierRange
}
func (s Struct) GetDocumentRange() Range {
	return s.documentRange
}
