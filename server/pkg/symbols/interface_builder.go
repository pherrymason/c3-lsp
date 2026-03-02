package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

type InterfaceBuilder struct {
	_interface Interface
}

func NewInterfaceBuilder(name string, module string, docId string) *InterfaceBuilder {
	f := &InterfaceBuilder{
		_interface: Interface{
			methods: make(map[string]*Function, 0),
			BaseIndexable: NewBaseIndexable(
				name,
				module,
				docId,
				NewRange(0, 0, 0, 0),
				NewRange(0, 0, 0, 0),
				protocol.CompletionItemKindInterface,
			),
		},
	}

	f.WithDocumentRange(0, 0, 0, 1000)

	return f
}

func (ib *InterfaceBuilder) WithoutSourceCode() *InterfaceBuilder {
	ib._interface.BaseIndexable.HasSourceCode_ = false
	return ib
}

func (ib *InterfaceBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *InterfaceBuilder {
	ib._interface.BaseIndexable.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return ib
}

func (ib *InterfaceBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *InterfaceBuilder {
	ib._interface.BaseIndexable.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return ib
}

func (ib *InterfaceBuilder) WithDocs(docs string) *InterfaceBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	ib._interface.BaseIndexable.DocComment = &docComment
	return ib
}

func (ib *InterfaceBuilder) Build() Interface {
	return ib._interface
}
