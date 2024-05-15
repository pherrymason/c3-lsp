package symbols

import "github.com/pherrymason/c3-lsp/option"

type StructBuilder struct {
	strukt Struct
}

func NewStructBuilder(name string, module string, docId string) *StructBuilder {
	return &StructBuilder{
		strukt: NewStruct(name, []string{}, []StructMember{}, module, docId, Range{}, Range{}),
	}
}

func (b *StructBuilder) WithStructMember(name string, baseType string, idRange Range, module string, docId string) *StructBuilder {
	b.strukt.members = append(b.strukt.members, NewStructMember(name, baseType, option.None[[2]uint](), module, docId, idRange))
	return b
}

func (b *StructBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *StructBuilder {
	b.strukt.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return b
}

func (b *StructBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *StructBuilder {
	b.strukt.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return b
}

func (b *StructBuilder) ImplementsInterface(interfaceName string) *StructBuilder {
	b.strukt.implements = append(b.strukt.implements, interfaceName)
	return b
}

func (b *StructBuilder) Build() Struct {
	return b.strukt
}