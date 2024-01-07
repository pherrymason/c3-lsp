package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type FunctionIndexable struct {
	Name                string
	ReturnType          string
	DocumentURI         protocol.DocumentUri
	DeclarationPosition protocol.Position
	Kind                protocol.CompletionItemKind
}

func NewFunctionIndexable(name string, uri protocol.DocumentUri, position protocol.Position, kind protocol.CompletionItemKind) FunctionIndexable {
	return FunctionIndexable{
		Name:                name,
		ReturnType:          "??",
		DocumentURI:         uri,
		DeclarationPosition: position,
		Kind:                kind,
	}
}

func (f FunctionIndexable) GetName() string {
	return f.Name
}

func (f FunctionIndexable) GetKind() protocol.CompletionItemKind {
	return f.Kind
}

func (f FunctionIndexable) GetDocumentURI() protocol.DocumentUri {
	return f.DocumentURI
}

func (f FunctionIndexable) GetDeclarationPosition() protocol.Position {
	return f.DeclarationPosition
}
