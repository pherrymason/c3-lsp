package symbols

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type TypeDefBuilder struct {
	distinct TypeDef
}

func NewTypeDefBuilder(name string, module string, docId string) *TypeDefBuilder {
	return &TypeDefBuilder{
		distinct: TypeDef{
			baseType: nil,
			inline:   false,
			BaseIndexable: NewBaseIndexable(
				name,
				module,
				docId,
				NewRange(0, 0, 0, 0),
				NewRange(0, 0, 0, 0),
				protocol.CompletionItemKindTypeParameter,
			),
		},
	}
}

func (d *TypeDefBuilder) WithName(name string) *TypeDefBuilder {
	d.distinct.Name = name
	return d
}

func (d *TypeDefBuilder) WithBaseType(baseType Type) *TypeDefBuilder {
	d.distinct.baseType = &baseType
	return d
}

func (d *TypeDefBuilder) WithInline(inline bool) *TypeDefBuilder {
	d.distinct.inline = inline
	return d
}

func (d *TypeDefBuilder) WithoutSourceCode() *TypeDefBuilder {
	d.distinct.HasSourceCode_ = false
	return d
}

func (d *TypeDefBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *TypeDefBuilder {
	d.distinct.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return d
}

func (d *TypeDefBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *TypeDefBuilder {
	d.distinct.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return d
}

func (d *TypeDefBuilder) WithDocs(docs string) *TypeDefBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	d.distinct.DocComment = &docComment
	return d
}

func (d *TypeDefBuilder) Build() *TypeDef {
	return &d.distinct
}
