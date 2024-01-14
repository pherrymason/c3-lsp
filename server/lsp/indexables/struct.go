package indexables

import (
	"fmt"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Struct struct {
	name    string
	members []StructMember
	BaseIndexable
}

func NewStruct(name string, members []StructMember, docId string, idRange Range) Struct {
	return Struct{
		name:    name,
		members: members,
		BaseIndexable: BaseIndexable{
			identifierRange: idRange,
			documentURI:     docId,
		},
	}
}

func (s Struct) GetName() string {
	return s.name
}

func (s Struct) GetKind() protocol.CompletionItemKind {
	return s.Kind
}

func (s Struct) GetDocumentURI() string {
	return s.documentURI
}

func (s Struct) GetDeclarationRange() Range {
	return s.identifierRange
}
func (s Struct) GetDocumentRange() Range {
	return s.documentRange
}

type StructMember struct {
	name     string
	baseType string
	posRange Range
}

func NewStructMember(name string, baseType string, posRange Range) StructMember {
	return StructMember{
		name:     name,
		baseType: baseType,
		posRange: posRange,
	}
}

func (s Struct) GetHoverInfo() string {
	return fmt.Sprintf("%s", s.name)
}
