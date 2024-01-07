package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type Variable struct {
	Name                string
	Type                string
	DocumentURI         protocol.DocumentUri
	DeclarationPosition protocol.Position
	Kind                protocol.CompletionItemKind
}

func NewVariable(name string, variableType string, uri protocol.DocumentUri, position protocol.Position, kind protocol.CompletionItemKind) Variable {
	return Variable{
		Name:                name,
		Type:                variableType,
		DocumentURI:         uri,
		DeclarationPosition: position,
		Kind:                kind,
	}
}

func (v Variable) GetType() string {
	return v.Type
}

func (v Variable) GetName() string {
	return v.Name
}

func (v Variable) GetKind() protocol.CompletionItemKind {
	return v.Kind
}

func (v Variable) GetDocumentURI() protocol.DocumentUri {
	return v.DocumentURI
}

func (v Variable) GetDeclarationPosition() protocol.Position {
	return v.DeclarationPosition
}
