package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type Variable struct {
	Name string
	Type string
	BaseIndexable
}

func NewVariable(name string, variableType string, uri protocol.DocumentUri, identifierRangePosition protocol.Range, documentRangePosition protocol.Range, kind protocol.CompletionItemKind) Variable {
	return Variable{
		Name: name,
		Type: variableType,
		BaseIndexable: BaseIndexable{
			documentURI:     uri,
			identifierRange: identifierRangePosition,
			documentRange:   documentRangePosition,
			Kind:            kind,
		},
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
	return v.documentURI
}

func (v Variable) GetDeclarationRange() protocol.Range {
	return v.identifierRange
}
func (v Variable) GetDocumentRange() protocol.Range {
	return v.documentRange
}
