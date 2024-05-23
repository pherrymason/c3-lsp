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

func (eb *EnumBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *EnumBuilder {
	eb.enum.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *EnumBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *EnumBuilder {
	eb.enum.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *EnumBuilder) WithEnumerator(enumerator *Enumerator) *EnumBuilder {
	eb.enum.enumerators = append(eb.enum.enumerators, enumerator)

	return eb
}

func (eb *EnumBuilder) Build() Enum {
	return eb.enum
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

func (eb *EnumeratorBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *EnumeratorBuilder {
	eb.enumerator.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *EnumeratorBuilder) Build() *Enumerator {
	return &eb.enumerator
}
