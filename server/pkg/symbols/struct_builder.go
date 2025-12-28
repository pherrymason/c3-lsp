package symbols

import "github.com/pherrymason/c3-lsp/pkg/option"

type StructBuilder struct {
	strukt Struct
}

func NewStructBuilder(name string, module string, docId string) *StructBuilder {
	return &StructBuilder{
		strukt: NewStruct(name, []string{}, []*StructMember{}, module, docId, Range{}, Range{}),
	}
}

func (sb *StructBuilder) WithoutSourceCode() *StructBuilder {
	sb.strukt.BaseIndexable.HasSourceCode_ = false
	return sb
}

func (sb *StructBuilder) WithDocs(docs string) *StructBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	sb.strukt.BaseIndexable.DocComment = &docComment
	return sb
}

func (b *StructBuilder) WithStructMember(name string, baseType Type, module string, docId string) *StructBuilder {
	member := NewStructMember(
		name,
		baseType,
		option.None[[2]uint](),
		module,
		docId,
		NewRange(0, 0, 0, 0),
	)
	b.strukt.members = append(b.strukt.members, &member)
	return b
}

func (b *StructBuilder) WithSubStructMember(name string, baseType Type, module string, docId string) *StructBuilder {
	member := NewStructMember(
		name,
		baseType,
		option.None[[2]uint](),
		module,
		docId,
		NewRange(0, 0, 0, 0),
	)
	member.inlinePendingResolve = true
	b.strukt.members = append(b.strukt.members, &member)
	return b
}

func (b *StructBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *StructBuilder {
	b.strukt.BaseIndexable.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return b
}

func (b *StructBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *StructBuilder {
	b.strukt.BaseIndexable.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return b
}

func (b *StructBuilder) ImplementsInterface(interfaceName string) *StructBuilder {
	b.strukt.implements = append(b.strukt.implements, interfaceName)
	return b
}

func (b *StructBuilder) Build() *Struct {
	return &b.strukt
}
