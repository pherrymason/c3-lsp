package symbols

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Struct struct {
	members    []*StructMember
	isUnion    bool
	implements []string
	BaseIndexable
}

func NewStruct(name string, interfaces []string, members []*StructMember, module string, docId *string, idRange Range, docRange Range) Struct {
	strukt := Struct{
		members:    members,
		isUnion:    false,
		implements: interfaces,
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
		strukt.Insert(member)
	}

	return strukt
}

func NewUnion(name string, members []*StructMember, module string, docId *string, idRange Range, docRange Range) Struct {
	union := Struct{
		members: members,
		isUnion: true,
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
		union.Insert(member)
	}

	return union
}

func (s Struct) GetMembers() []*StructMember {
	return s.members
}

func (s Struct) GetInterfaces() []string {
	return s.implements
}

func (s Struct) IsUnion() bool {
	return s.isUnion
}

func (s Struct) GetHoverInfo() string {
	return fmt.Sprintf("%s", s.name)
}

func (s *Struct) InheritMembersFrom(inlinedMemberName string, otherStruct *Struct) {
	for _, member := range s.GetMembers() {
		if member.GetType().GetName() == inlinedMemberName {
			member.inlinePendingResolve = false
			member.expandedInline = true
		}
	}

	for _, member := range otherStruct.GetMembers() {
		s.members = append(s.members, member)
	}
}

type StructMember struct {
	baseType             Type
	bitRange             option.Option[[2]uint]
	inlinePendingResolve bool
	expandedInline       bool
	BaseIndexable
}

func (m StructMember) IsInlinePendingToResolve() bool {
	return m.inlinePendingResolve
}

func (m StructMember) IsExpandedInline() bool {
	return m.expandedInline
}

func (m *StructMember) GetType() *Type {
	return &m.baseType
}

func (m StructMember) GetBitRange() [2]uint {
	return m.bitRange.Get()
}

func (s StructMember) GetHoverInfo() string {
	return fmt.Sprintf("%s %s", s.baseType, s.name)
}

func NewStructMember(name string, fieldType Type, bitRanges option.Option[[2]uint], module string, docId *string, idRange Range) StructMember {
	return StructMember{
		baseType: fieldType,
		bitRange: bitRanges,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			NewRange(0, 0, 0, 0),
			protocol.CompletionItemKindField,
		),
	}
}

func NewInlineSubtype(name string, fieldType Type, module string, docId *string, idRange Range) StructMember {
	return StructMember{
		baseType:             fieldType,
		bitRange:             option.None[[2]uint](),
		inlinePendingResolve: true,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			NewRange(0, 0, 0, 0),
			protocol.CompletionItemKindField,
		),
	}
}
