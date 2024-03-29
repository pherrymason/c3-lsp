package indexables

import (
	"fmt"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Variable struct {
	name string
	Type string
	BaseIndexable
}

func NewVariable(name string, variableType string, module string, uri string, identifierRangePosition Range, documentRangePosition Range) Variable {
	return Variable{
		name: name,
		Type: variableType,
		BaseIndexable: BaseIndexable{
			module:          module,
			documentURI:     uri,
			identifierRange: identifierRangePosition,
			documentRange:   documentRangePosition,
			Kind:            protocol.CompletionItemKindVariable,
		},
	}

}

func (v Variable) GetType() string {
	return v.Type
}

func (v Variable) GetName() string {
	return v.name
}

func (v Variable) GetModule() string { return v.module }

func (v Variable) GetKind() protocol.CompletionItemKind {
	return v.Kind
}

func (v Variable) GetDocumentURI() string {
	return v.documentURI
}

func (v Variable) GetDeclarationRange() Range {
	return v.identifierRange
}
func (v Variable) GetDocumentRange() Range {
	return v.documentRange
}

func (v Variable) GetHoverInfo() string {
	return fmt.Sprintf("%s %s", v.GetType(), v.GetName())
}
