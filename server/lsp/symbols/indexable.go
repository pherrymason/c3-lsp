package symbols

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type SymbolType int

// Declarar constantes para los días de la semana utilizando iota
const (
	ModuleSymbolType SymbolType = iota
	FunctionSymbolType
	VariableSymbolType
	EnumSymbolType
	EnumeratorSymbolType
	FaultSymbolType
	FaultConstantType
	StructSymbolType
	StructMemberSymbolType
	BitstructSymbolType
)

type Indexable interface {
	GetName() string
	//GetType() SymbolType
	GetKind() protocol.CompletionItemKind
	GetDocumentURI() string
	GetIdRange() Range
	GetDocumentRange() Range
	GetModuleString() string
	GetModule() ModulePath
	IsSubModuleOf(parentModule ModulePath) bool

	GetHoverInfo() string

	Children() []Indexable
	NestedScopes() []Indexable
	ChildrenWithoutScopes() []Indexable
	Insert(symbol Indexable)
	InsertNestedScope(symbol Indexable)
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
	children     []Indexable
	nestedScopes []Indexable
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

func (b BaseIndexable) Children() []Indexable {
	return b.children
}

func (b BaseIndexable) NestedScopes() []Indexable {
	return b.nestedScopes
}

func (b BaseIndexable) ChildrenWithoutScopes() []Indexable {
	return b.children
}

func (b *BaseIndexable) Insert(child Indexable) {
	b.children = append(b.children, child)
}

func (b *BaseIndexable) InsertNestedScope(symbol Indexable) {
	b.nestedScopes = append(b.nestedScopes, symbol)
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
