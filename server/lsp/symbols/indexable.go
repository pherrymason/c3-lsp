package symbols

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
	name         string
	moduleString string
	module       ModulePath
	documentURI  string
	idRange      Range
	docRange     Range
	Kind         protocol.CompletionItemKind
}

func (b BaseIndexable) GetName() string {
	return b.name
}

func (b BaseIndexable) GetKind() protocol.CompletionItemKind {
	return b.Kind
}

func (b BaseIndexable) GetModuleString() string {
	return b.moduleString
}

func (b BaseIndexable) GetModule() ModulePath {
	return b.module
}

func (b BaseIndexable) IsSubModuleOf(module ModulePath) bool {
	if module.IsEmpty() {
		return false
	}

	return b.module.IsSubModuleOf(module)
}

func (b BaseIndexable) GetDocumentURI() string {
	return b.documentURI
}

func (b BaseIndexable) GetDocumentRange() Range {
	return b.docRange
}

func (b BaseIndexable) GetIdRange() Range {
	return b.idRange
}

func NewBaseIndexable(name string, module string, docId protocol.DocumentUri, idRange Range, docRange Range, kind protocol.CompletionItemKind) BaseIndexable {
	return BaseIndexable{
		name:         name,
		module:       NewModulePathFromString(module),
		moduleString: module,
		documentURI:  docId,
		idRange:      idRange,
		docRange:     docRange,
		Kind:         kind,
	}
}
