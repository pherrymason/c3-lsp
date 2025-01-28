package symbols

import "github.com/pherrymason/c3-lsp/pkg/option"

type BitstructBuilder struct {
	bitstruct Bitstruct
}

func NewBitstructBuilder(name string, backingType Type, module string, docId string) *BitstructBuilder {
	return &BitstructBuilder{
		bitstruct: NewBitstruct(name, backingType, []string{}, []*StructMember{}, module, docId, Range{}, Range{}),
	}
}

func (sb *BitstructBuilder) WithoutSourceCode() *BitstructBuilder {
	sb.bitstruct.BaseIndexable.hasSourceCode = false
	return sb
}

func (b *BitstructBuilder) WithStructMember(name string, baseType Type, module string, docId string) *BitstructBuilder {
	member := NewStructMember(name, baseType, option.None[[2]uint](), module, docId, NewRange(0, 0, 0, 0))
	b.bitstruct.members = append(b.bitstruct.members, &member)
	return b
}

func (b *BitstructBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *BitstructBuilder {
	b.bitstruct.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return b
}

func (b *BitstructBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *BitstructBuilder {
	b.bitstruct.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return b
}

func (b *BitstructBuilder) ImplementsInterface(interfaceName string) *BitstructBuilder {
	b.bitstruct.implements = append(b.bitstruct.implements, interfaceName)

	return b
}

func (b *BitstructBuilder) Build() *Bitstruct {
	return &b.bitstruct
}
