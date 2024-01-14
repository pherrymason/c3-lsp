package indexables

import (
	"fmt"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Def struct {
	name       string
	resolvesTo string
	BaseIndexable
}

func NewDef(name string, resolvesTo string, docId string, idRange Range, docRange Range) Def {
	return Def{
		name:       name,
		resolvesTo: resolvesTo,
		BaseIndexable: BaseIndexable{
			documentURI:     docId,
			identifierRange: idRange,
			documentRange:   docRange,
			Kind:            protocol.CompletionItemKindTypeParameter,
		},
	}
}

func (d Def) GetName() string {
	return d.name
}

func (d Def) GetKind() protocol.CompletionItemKind {
	return d.Kind
}

func (d Def) GetDocumentURI() string {
	return d.documentURI
}

func (d Def) GetDeclarationRange() Range {
	return d.identifierRange
}

func (d Def) GetDocumentRange() Range {
	return d.documentRange
}

func (d Def) GetHoverInfo() string {
	return fmt.Sprintf("def %s = %s", d.name, d.resolvesTo)
}
