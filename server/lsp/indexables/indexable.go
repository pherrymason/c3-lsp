package indexables

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Indexable interface {
	GetName() string
	GetKind() protocol.CompletionItemKind
	GetDocumentURI() string
	GetIdRange() Range
	GetDocumentRange() Range
	GetModuleString() string
	GetModule() ModulePath
	IsSubModuleOf(parentModule ModulePath) bool

	GetHoverInfo() string
}

type IndexableCollection []Indexable

type BaseIndexable struct {
	moduleString string
	module       ModulePath
	documentURI  string
	idRange      Range
	docRange     Range
	Kind         protocol.CompletionItemKind
}

func NewBaseIndexable(module string, docId protocol.DocumentUri, idRange Range, docRange Range, kind protocol.CompletionItemKind) BaseIndexable {
	return BaseIndexable{
		module:       NewModulePathFromString(module),
		moduleString: module,
		documentURI:  docId,
		idRange:      idRange,
		docRange:     docRange,
		Kind:         kind,
	}
}
