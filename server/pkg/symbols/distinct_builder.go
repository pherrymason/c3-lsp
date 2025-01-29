package symbols

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type DistinctBuilder struct {
	distinct Distinct
}

func NewDistinctBuilder(name string, module string, docId string) *DistinctBuilder {
	return &DistinctBuilder{
		distinct: Distinct{
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

func (d *DistinctBuilder) WithName(name string) *DistinctBuilder {
	d.distinct.name = name
	return d
}

func (d *DistinctBuilder) WithBaseType(baseType Type) *DistinctBuilder {
	d.distinct.baseType = &baseType
	return d
}

func (d *DistinctBuilder) WithInline(inline bool) *DistinctBuilder {
	d.distinct.inline = inline
	return d
}

func (d *DistinctBuilder) WithoutSourceCode() *DistinctBuilder {
	d.distinct.BaseIndexable.hasSourceCode = false
	return d
}

func (d *DistinctBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *DistinctBuilder {
	d.distinct.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return d
}

func (d *DistinctBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *DistinctBuilder {
	d.distinct.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return d
}

func (d *DistinctBuilder) WithDocs(docs string) *DistinctBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	d.distinct.BaseIndexable.docComment = &docComment
	return d
}

func (d *DistinctBuilder) Build() *Distinct {
	return &d.distinct
}
