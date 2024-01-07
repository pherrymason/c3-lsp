package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type Function struct {
	Name                string
	ReturnType          string
	DocumentURI         protocol.DocumentUri
	DeclarationPosition protocol.Position
	Kind                protocol.CompletionItemKind
}

func NewFunction(name string, uri protocol.DocumentUri, position protocol.Position, kind protocol.CompletionItemKind) Function {
	return Function{
		Name:                name,
		ReturnType:          "??",
		DocumentURI:         uri,
		DeclarationPosition: position,
		Kind:                kind,
	}
}

func (f Function) GetName() string {
	return f.Name
}

func (f Function) GetKind() protocol.CompletionItemKind {
	return f.Kind
}

func (f Function) GetDocumentURI() protocol.DocumentUri {
	return f.DocumentURI
}

func (f Function) GetDeclarationPosition() protocol.Position {
	return f.DeclarationPosition
}
