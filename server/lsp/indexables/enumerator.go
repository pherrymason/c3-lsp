package indexables

import (
	"fmt"
	"github.com/tliron/glsp/protocol_3_16"
)

type Enumerator struct {
	name  string
	value string
	BaseIndexable
}

func NewEnumerator(name string, value string, module string, identifierPosition Range, docId string) Enumerator {
	return Enumerator{
		name:  name,
		value: value,
		BaseIndexable: BaseIndexable{
			module:          module,
			documentURI:     docId,
			identifierRange: identifierPosition,
			Kind:            protocol.CompletionItemKindEnumMember,
		},
	}
}

func (e Enumerator) GetName() string {
	return e.name
}

func (e Enumerator) GetKind() protocol.CompletionItemKind {
	return e.Kind
}

func (e Enumerator) GetModule() string {
	return e.module
}

func (e Enumerator) GetDocumentURI() string {
	return e.documentURI
}

func (e Enumerator) GetDeclarationRange() Range {
	return e.identifierRange
}

func (e Enumerator) GetDocumentRange() Range {
	return e.documentRange
}

func (e Enumerator) GetHoverInfo() string {
	return fmt.Sprintf("%s: %s", e.name, e.value)
}
