package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

type Bitstruct struct {
	backingType Type
	members     []StructMember
	implements  []string
	BaseIndexable
}

func NewBitstruct(name string, backingType string, interfaces []string, members []StructMember, module string, docId string, idRange Range, docRange Range) Bitstruct {
	return Bitstruct{
		backingType: NewTypeFromString(backingType),
		members:     members,
		implements:  interfaces,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindStruct,
		),
	}
}

func (b Bitstruct) Type() Type {
	return b.backingType
}

func (b Bitstruct) Members() []StructMember {
	return b.members
}
