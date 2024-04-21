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

func NewDef(name string, resolvesTo string, module string, docId string, idRange Range, docRange Range) Def {
	return Def{
		name:       name,
		resolvesTo: resolvesTo,
		BaseIndexable: BaseIndexable{
			module:      module,
			documentURI: docId,
			idRange:     idRange,
			docRange:    docRange,
			Kind:        protocol.CompletionItemKindTypeParameter,
		},
	}
}

func (d Def) GetName() string {
	return d.name
}

func (d Def) GetModule() string {
	return d.module
}

func (d Def) GetKind() protocol.CompletionItemKind {
	return d.Kind
}

func (d Def) GetDocumentURI() string {
	return d.documentURI
}

func (d Def) GetIdRange() Range {
	return d.idRange
}

func (d Def) GetDocumentRange() Range {
	return d.docRange
}

func (d Def) GetHoverInfo() string {
	return fmt.Sprintf("def %s = %s", d.name, d.resolvesTo)
}
