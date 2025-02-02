package symbols

import (
	"fmt"

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
	GetFQN() string // Get Full Qualified URI
	GetKind() protocol.CompletionItemKind
	GetDocumentURI() string
	GetIdRange() Range
	GetDocumentRange() Range
	GetModuleString() string
	GetModule() ModulePath
	IsSubModuleOf(parentModule ModulePath) bool

	GetDocComment() *DocComment
	GetHoverInfo() string
	GetCompletionDetail() string
	HasSourceCode() bool // This will return false for that code that is not accesible either because it belongs to the stdlib, or inside a .c3lib library. This results in disabling "Go to definition" / "Go to declaration" on these symbols

	Children() []Indexable
	ChildrenNames() []string
	NestedScopes() []Indexable
	ChildrenWithoutScopes() []Indexable
	Insert(symbol Indexable)
	InsertNestedScope(symbol Indexable)
}

type Typeable interface {
	GetType() *Type
}

type IndexableCollection []Indexable

type BaseIndexable struct {
	name          string
	moduleString  string
	module        ModulePath
	documentURI   string
	hasSourceCode bool
	idRange       Range
	docRange      Range
	Kind          protocol.CompletionItemKind
	docComment    *DocComment
	attributes    []string

	children      []Indexable
	childrenNames []string
	nestedScopes  []Indexable
}

func (b *BaseIndexable) GetName() string {
	return b.name
}

func (b *BaseIndexable) GetFQN() string {
	return fmt.Sprintf("%s::%s", b.module.GetName(), b.GetName())
}

func (b *BaseIndexable) GetKind() protocol.CompletionItemKind {
	return b.Kind
}

func (b *BaseIndexable) GetModuleString() string {
	return b.moduleString
}

func (b *BaseIndexable) GetModule() ModulePath {
	return b.module
}

func (b *BaseIndexable) IsSubModuleOf(module ModulePath) bool {
	if module.IsEmpty() {
		return false
	}

	return b.module.IsSubModuleOf(module)
}

func (b *BaseIndexable) GetDocumentURI() string {
	return b.documentURI
}

func (b *BaseIndexable) GetDocumentRange() Range {
	return b.docRange
}

func (b *BaseIndexable) GetIdRange() Range {
	return b.idRange
}

func (b *BaseIndexable) HasSourceCode() bool {
	return b.hasSourceCode
}

func (b *BaseIndexable) IsPrivate() bool {
	for _, attr := range b.attributes {
		if attr == "@private" {
			return true
		}
	}
	return false
}

func (b *BaseIndexable) SetDocumentURI(docId string) {
	b.documentURI = docId
}

func (b *BaseIndexable) GetDocComment() *DocComment {
	return b.docComment
}

func (b *BaseIndexable) GetAttributes() []string {
	return b.attributes
}

func (b *BaseIndexable) SetAttributes(attributes []string) {
	b.attributes = attributes
}

func (b *BaseIndexable) Children() []Indexable {
	return b.children
}

func (b *BaseIndexable) ChildrenNames() []string {
	return b.childrenNames
}

func (b *BaseIndexable) NestedScopes() []Indexable {
	return b.nestedScopes
}

func (b *BaseIndexable) ChildrenWithoutScopes() []Indexable {
	return b.children
}

func (b *BaseIndexable) Insert(child Indexable) {
	b.children = append(b.children, child)
	b.childrenNames = append(b.childrenNames, child.GetName())
}

func (b *BaseIndexable) InsertNestedScope(symbol Indexable) {
	b.nestedScopes = append(b.nestedScopes, symbol)
}

func (b *BaseIndexable) SetDocComment(docComment *DocComment) {
	b.docComment = docComment
}

func (b *BaseIndexable) formatSource(source string) string {
	return fmt.Sprintf("```c3\n%s```", source)
}

func NewBaseIndexable(name string, module string, docId protocol.DocumentUri, idRange Range, docRange Range, kind protocol.CompletionItemKind) BaseIndexable {
	return BaseIndexable{
		name:          name,
		module:        NewModulePathFromString(module),
		moduleString:  module,
		documentURI:   docId,
		idRange:       idRange,
		docRange:      docRange,
		Kind:          kind,
		hasSourceCode: true,
		docComment:    nil,
		attributes:    []string{},
	}
}
