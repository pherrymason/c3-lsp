package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

type Bitstruct struct {
	backingType Type
	members     []*StructMember
	implements  []string
	BaseIndexable
}

func NewBitstruct(name string, backingType Type, interfaces []string, members []*StructMember, module string, docId string, idRange Range, docRange Range) Bitstruct {
	bitstruct := Bitstruct{
		backingType: backingType,
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

	for _, member := range members {
		bitstruct.Insert(member)
	}

	return bitstruct
}

func (b Bitstruct) Type() Type {
	return b.backingType
}

func (b Bitstruct) Members() []*StructMember {
	return b.members
}

func (b Bitstruct) GetHoverInfo() string {
	return b.Name
}

func (b Bitstruct) GetCompletionDetail() string {
	// Same rationale as for struct
	return "Type"
}
