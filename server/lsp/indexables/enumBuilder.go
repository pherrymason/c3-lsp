package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type EnumBuilder struct {
	enum Enum
}

func NewEnumBuilder(name string, baseType string, module string, docId string) *EnumBuilder {
	return &EnumBuilder{
		enum: Enum{
			name:     name,
			baseType: baseType,
			BaseIndexable: BaseIndexable{
				module:      module,
				documentURI: docId,
				Kind:        protocol.CompletionItemKindEnum,
			},
		},
	}
}

func (eb *EnumBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *EnumBuilder {
	eb.enum.BaseIndexable.identifierRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *EnumBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *EnumBuilder {
	eb.enum.BaseIndexable.documentRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *EnumBuilder) WithEnumerator(enumerator Enumerator) *EnumBuilder {
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
			name:  name,
			value: "",
			BaseIndexable: BaseIndexable{
				documentURI: docId,
				Kind:        protocol.CompletionItemKindEnumMember,
			},
		},
	}
}

func (eb *EnumeratorBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *EnumeratorBuilder {
	eb.enumerator.BaseIndexable.identifierRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *EnumeratorBuilder) Build() Enumerator {
	return eb.enumerator
}
