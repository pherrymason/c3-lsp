package indexables

type StructBuilder struct {
	strukt Struct
}

func NewStructBuilder(name string, module string, docId string) *StructBuilder {
	return &StructBuilder{
		strukt: NewStruct(name, []StructMember{}, module, docId, Range{}),
	}
}

func (b *StructBuilder) WithStructMember(name string, baseType string, posRange Range) *StructBuilder {
	b.strukt.members = append(b.strukt.members, NewStructMember(name, baseType, posRange))
	return b
}

func (b *StructBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *StructBuilder {
	b.strukt.BaseIndexable.identifierRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return b
}

func (b *StructBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *StructBuilder {
	b.strukt.BaseIndexable.documentRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return b
}

func (b *StructBuilder) Build() Struct {
	return b.strukt
}
