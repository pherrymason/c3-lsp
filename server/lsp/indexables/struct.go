package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type Struct struct {
	Name                  string
	Members               []string
	DocumentURI           protocol.URI
	DocumentPositionRange protocol.Position
	Kind                  protocol.CompletionItemKind
}

func (s Struct) GetName() string {
	return s.Name
}

func (s Struct) GetKind() protocol.CompletionItemKind {
	return s.Kind
}

func (s Struct) GetDocumentURI() protocol.DocumentUri {
	return s.DocumentURI
}

func (s Struct) GetDeclarationPosition() protocol.Position {
	return s.DocumentPositionRange
}
