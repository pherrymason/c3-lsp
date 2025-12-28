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
	GetFQN() string // Get Full Qualified Name
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
	Name           string                      `json:"name"`
	ModuleString   string                      `json:"module"`
	Module         ModulePath                  `json:"-"` // Skip - will be reconstructed from moduleString
	DocumentURI    string                      `json:"documentURI"`
	HasSourceCode_ bool                        `json:"hasSourceCode"`
	IdRange        Range                       `json:"idRange"`
	DocRange       Range                       `json:"docRange"`
	Kind           protocol.CompletionItemKind `json:"kind"`
	DocComment     *DocComment                 `json:"docComment,omitempty"`
	Attributes     []string                    `json:"attributes,omitempty"`

	Children_      []Indexable `json:"-"` // Skip - reconstructed on load
	ChildrenNames_ []string    `json:"-"` // Skip - reconstructed on load
	NestedScopes_  []Indexable `json:"-"` // Skip - reconstructed on load
}

func (b *BaseIndexable) GetName() string {
	return b.Name
}

func (b *BaseIndexable) GetFQN() string {
	return fmt.Sprintf("%s::%s", b.Module.GetName(), b.GetName())
}

func (b *BaseIndexable) GetKind() protocol.CompletionItemKind {
	return b.Kind
}

func (b *BaseIndexable) GetModuleString() string {
	return b.ModuleString
}

func (b *BaseIndexable) GetModule() ModulePath {
	return b.Module
}

func (b *BaseIndexable) IsSubModuleOf(module ModulePath) bool {
	if module.IsEmpty() {
		return false
	}

	return b.Module.IsSubModuleOf(module)
}

func (b *BaseIndexable) GetDocumentURI() string {
	return b.DocumentURI
}

func (b *BaseIndexable) GetDocumentRange() Range {
	return b.DocRange
}

func (b *BaseIndexable) GetIdRange() Range {
	return b.IdRange
}

func (b *BaseIndexable) HasSourceCode() bool {
	return b.HasSourceCode_
}

func (b *BaseIndexable) IsPrivate() bool {
	for _, attr := range b.Attributes {
		if attr == "@private" {
			return true
		}
	}
	return false
}

func (b *BaseIndexable) SetDocumentURI(docId string) {
	b.DocumentURI = docId
}

func (b *BaseIndexable) GetDocComment() *DocComment {
	return b.DocComment
}

func (b *BaseIndexable) GetAttributes() []string {
	return b.Attributes
}

func (b *BaseIndexable) SetAttributes(attributes []string) {
	b.Attributes = attributes
}

func (b *BaseIndexable) Children() []Indexable {
	return b.Children_
}

func (b *BaseIndexable) ChildrenNames() []string {
	return b.ChildrenNames_
}

func (b *BaseIndexable) NestedScopes() []Indexable {
	return b.NestedScopes_
}

func (b *BaseIndexable) ChildrenWithoutScopes() []Indexable {
	return b.Children_
}

func (b *BaseIndexable) Insert(child Indexable) {
	b.Children_ = append(b.Children_, child)
	b.ChildrenNames_ = append(b.ChildrenNames_, child.GetName())
}

func (b *BaseIndexable) InsertNestedScope(symbol Indexable) {
	b.NestedScopes_ = append(b.NestedScopes_, symbol)
}

func (b *BaseIndexable) SetDocComment(docComment *DocComment) {
	b.DocComment = docComment
}

func (b *BaseIndexable) formatSource(source string) string {
	return fmt.Sprintf("```c3\n%s```", source)
}

func NewBaseIndexable(name string, module string, docId protocol.DocumentUri, idRange Range, docRange Range, kind protocol.CompletionItemKind) BaseIndexable {
	return BaseIndexable{
		Name:           name,
		Module:         NewModulePathFromString(module),
		ModuleString:   module,
		DocumentURI:    docId,
		IdRange:        idRange,
		DocRange:       docRange,
		Kind:           kind,
		HasSourceCode_: true,
		DocComment:     nil,
		Attributes:     []string{},
	}
}
