package indexables

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
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
			moduleString: module,
			module:       NewModulePathFromString(module),
			documentURI:  docId,
			idRange:      identifierPosition,
			Kind:         protocol.CompletionItemKindEnumMember,
		},
	}
}

func (e Enumerator) GetName() string {
	return e.name
}

func (e Enumerator) GetKind() protocol.CompletionItemKind {
	return e.Kind
}

func (e Enumerator) GetModuleString() string {
	return e.moduleString
}

func (e Enumerator) GetModule() ModulePath {
	return e.module
}

func (e Enumerator) IsSubModuleOf(module ModulePath) bool {
	return e.module.IsSubModuleOf(module)
}

func (e Enumerator) GetDocumentURI() string {
	return e.documentURI
}

func (e Enumerator) GetIdRange() Range {
	return e.idRange
}

func (e Enumerator) GetDocumentRange() Range {
	return e.docRange
}

func (e Enumerator) GetHoverInfo() string {
	return fmt.Sprintf("%s: %s", e.name, e.value)
}
