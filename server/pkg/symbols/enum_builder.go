package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

type EnumBuilder struct {
	enum Enum
}

func NewEnumBuilder(name string, baseType string, module string, docId string) *EnumBuilder {
	return &EnumBuilder{
		enum: Enum{
			baseType: baseType,
			BaseIndexable: NewBaseIndexable(
				name,
				module,
				docId,
				NewRange(0, 0, 0, 0),
				NewRange(0, 0, 0, 0),
				protocol.CompletionItemKindEnum,
			),
		},
	}
}

func (d *EnumBuilder) WithoutSourceCode() *EnumBuilder {
	d.enum.BaseIndexable.HasSourceCode_ = false
	return d
}

func (eb *EnumBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *EnumBuilder {
	eb.enum.BaseIndexable.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *EnumBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *EnumBuilder {
	eb.enum.BaseIndexable.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *EnumBuilder) WithDocs(docs string) *EnumBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	eb.enum.BaseIndexable.DocComment = &docComment
	return eb
}

func (eb *EnumBuilder) WithEnumerator(enumerator *Enumerator) *EnumBuilder {
	eb.enum.enumerators = append(eb.enum.enumerators, enumerator)

	return eb
}

func (eb *EnumBuilder) Build() *Enum {
	return &eb.enum
}

// EnumeratorBuilder
type EnumeratorBuilder struct {
	enumerator Enumerator
}

func NewEnumeratorBuilder(name string, docId string) *EnumeratorBuilder {
	return &EnumeratorBuilder{
		enumerator: Enumerator{
			value: "",
			BaseIndexable: NewBaseIndexable(
				name,
				"",
				docId,
				NewRange(0, 0, 0, 0),
				NewRange(0, 0, 0, 0),
				protocol.CompletionItemKindEnumMember,
			),
		},
	}
}

func (eb *EnumeratorBuilder) WithoutSourceCode() *EnumeratorBuilder {
	eb.enumerator.BaseIndexable.HasSourceCode_ = false
	return eb
}

func (eb *EnumeratorBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *EnumeratorBuilder {
	eb.enumerator.BaseIndexable.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *EnumeratorBuilder) WithAssociativeValues(associativeValues []Variable) *EnumeratorBuilder {
	eb.enumerator.AssociatedValues = associativeValues

	return eb
}

func (eb *EnumeratorBuilder) WithEnumName(name string) *EnumeratorBuilder {
	eb.enumerator.EnumName = name

	return eb
}

func (eb *EnumeratorBuilder) Build() *Enumerator {
	return &eb.enumerator
}
