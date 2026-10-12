package symbols

import (
	"encoding/json"

	"github.com/pherrymason/c3-lsp/pkg/option"
)

// JSON (de)serialization used by the stdlib disk cache.
//
// Most symbol types keep their data in unexported fields, which encoding/json
// silently drops. Each type below marshals through a DTO that exposes those
// fields, keeping them unexported in the public API. Derived indexes
// (Children_, NestedScopes_, Module) are not serialized; they are rebuilt by
// Module.RebuildAfterDeserialization.

type typeJSON struct {
	BaseTypeLanguage  bool               `json:"baseTypeLanguage,omitempty"`
	Name              string             `json:"name"`
	Pointer           int                `json:"pointer,omitempty"`
	Optional          bool               `json:"optional,omitempty"`
	GenericArguments  []Type             `json:"genericArguments,omitempty"`
	Module            string             `json:"module,omitempty"`
	IsGenericArgument bool               `json:"isGenericArgument,omitempty"`
	IsCollection      bool               `json:"isCollection,omitempty"`
	CollectionSize    option.Option[int] `json:"collectionSize"`
}

func (t Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(typeJSON{
		BaseTypeLanguage:  t.baseTypeLanguage,
		Name:              t.name,
		Pointer:           t.pointer,
		Optional:          t.optional,
		GenericArguments:  t.genericArguments,
		Module:            t.module,
		IsGenericArgument: t.isGenericArgument,
		IsCollection:      t.isCollection,
		CollectionSize:    t.collectionSize,
	})
}

func (t *Type) UnmarshalJSON(data []byte) error {
	var j typeJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*t = Type{
		baseTypeLanguage:  j.BaseTypeLanguage,
		name:              j.Name,
		pointer:           j.Pointer,
		optional:          j.Optional,
		genericArguments:  j.GenericArguments,
		module:            j.Module,
		isGenericArgument: j.IsGenericArgument,
		isCollection:      j.IsCollection,
		collectionSize:    j.CollectionSize,
	}
	return nil
}

type docCommentContractJSON struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

func (c DocCommentContract) MarshalJSON() ([]byte, error) {
	return json.Marshal(docCommentContractJSON{Name: c.name, Body: c.body})
}

func (c *DocCommentContract) UnmarshalJSON(data []byte) error {
	var j docCommentContractJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*c = DocCommentContract{name: j.Name, body: j.Body}
	return nil
}

type docCommentJSON struct {
	Body      string                `json:"body"`
	Contracts []*DocCommentContract `json:"contracts,omitempty"`
}

func (d DocComment) MarshalJSON() ([]byte, error) {
	return json.Marshal(docCommentJSON{Body: d.body, Contracts: d.contracts})
}

func (d *DocComment) UnmarshalJSON(data []byte) error {
	var j docCommentJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*d = DocComment{body: j.Body, contracts: j.Contracts}
	return nil
}

type functionJSON struct {
	BaseIndexable
	FType          FunctionType         `json:"fType"`
	ReturnType     Type                 `json:"returnType"`
	ArgumentIds    []string             `json:"argumentIds,omitempty"`
	TypeIdentifier string               `json:"typeIdentifier,omitempty"`
	Variables      map[string]*Variable `json:"variables,omitempty"`
}

func (f Function) MarshalJSON() ([]byte, error) {
	return json.Marshal(functionJSON{
		BaseIndexable:  f.BaseIndexable,
		FType:          f.fType,
		ReturnType:     f.returnType,
		ArgumentIds:    f.argumentIds,
		TypeIdentifier: f.typeIdentifier,
		Variables:      f.Variables,
	})
}

func (f *Function) UnmarshalJSON(data []byte) error {
	var j functionJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	if j.Variables == nil {
		j.Variables = make(map[string]*Variable)
	}
	*f = Function{
		BaseIndexable:  j.BaseIndexable,
		fType:          j.FType,
		returnType:     j.ReturnType,
		argumentIds:    j.ArgumentIds,
		typeIdentifier: j.TypeIdentifier,
		Variables:      j.Variables,
	}
	return nil
}

type enumJSON struct {
	BaseIndexable
	BaseType    string        `json:"baseType,omitempty"`
	Enumerators []*Enumerator `json:"enumerators,omitempty"`
}

func (e Enum) MarshalJSON() ([]byte, error) {
	return json.Marshal(enumJSON{BaseIndexable: e.BaseIndexable, BaseType: e.baseType, Enumerators: e.enumerators})
}

func (e *Enum) UnmarshalJSON(data []byte) error {
	var j enumJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*e = Enum{BaseIndexable: j.BaseIndexable, baseType: j.BaseType, enumerators: j.Enumerators}
	return nil
}

type enumeratorJSON struct {
	BaseIndexable
	Value            string     `json:"value,omitempty"`
	AssociatedValues []Variable `json:"associatedValues,omitempty"`
	EnumName         string     `json:"enumName"`
}

func (e Enumerator) MarshalJSON() ([]byte, error) {
	return json.Marshal(enumeratorJSON{
		BaseIndexable:    e.BaseIndexable,
		Value:            e.value,
		AssociatedValues: e.AssociatedValues,
		EnumName:         e.EnumName,
	})
}

func (e *Enumerator) UnmarshalJSON(data []byte) error {
	var j enumeratorJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*e = Enumerator{
		BaseIndexable:    j.BaseIndexable,
		value:            j.Value,
		AssociatedValues: j.AssociatedValues,
		EnumName:         j.EnumName,
	}
	return nil
}

type faultJSON struct {
	BaseIndexable
	BaseType  string           `json:"baseType,omitempty"`
	Constants []*FaultConstant `json:"constants,omitempty"`
}

func (f Fault) MarshalJSON() ([]byte, error) {
	return json.Marshal(faultJSON{BaseIndexable: f.BaseIndexable, BaseType: f.baseType, Constants: f.constants})
}

func (f *Fault) UnmarshalJSON(data []byte) error {
	var j faultJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*f = Fault{BaseIndexable: j.BaseIndexable, baseType: j.BaseType, constants: j.Constants}
	return nil
}

type faultConstantJSON struct {
	BaseIndexable
	FaultName string `json:"faultName,omitempty"`
}

func (c FaultConstant) MarshalJSON() ([]byte, error) {
	return json.Marshal(faultConstantJSON{BaseIndexable: c.BaseIndexable, FaultName: c.faultName})
}

func (c *FaultConstant) UnmarshalJSON(data []byte) error {
	var j faultConstantJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*c = FaultConstant{BaseIndexable: j.BaseIndexable, faultName: j.FaultName}
	return nil
}

type structJSON struct {
	BaseIndexable
	Members    []*StructMember `json:"members,omitempty"`
	IsUnion    bool            `json:"isUnion,omitempty"`
	Implements []string        `json:"implements,omitempty"`
}

func (s Struct) MarshalJSON() ([]byte, error) {
	return json.Marshal(structJSON{BaseIndexable: s.BaseIndexable, Members: s.members, IsUnion: s.isUnion, Implements: s.implements})
}

func (s *Struct) UnmarshalJSON(data []byte) error {
	var j structJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*s = Struct{BaseIndexable: j.BaseIndexable, members: j.Members, isUnion: j.IsUnion, implements: j.Implements}
	return nil
}

type structMemberJSON struct {
	BaseIndexable
	BaseType             Type                   `json:"baseType"`
	BitRange             option.Option[[2]uint] `json:"bitRange"`
	InlinePendingResolve bool                   `json:"inlinePendingResolve,omitempty"`
	ExpandedInline       bool                   `json:"expandedInline,omitempty"`
	IsStruct             bool                   `json:"isStruct,omitempty"`
	SubStruct            option.Option[*Struct] `json:"subStruct"`
}

func (m StructMember) MarshalJSON() ([]byte, error) {
	return json.Marshal(structMemberJSON{
		BaseIndexable:        m.BaseIndexable,
		BaseType:             m.baseType,
		BitRange:             m.bitRange,
		InlinePendingResolve: m.inlinePendingResolve,
		ExpandedInline:       m.expandedInline,
		IsStruct:             m.isStruct,
		SubStruct:            m.subStruct,
	})
}

func (m *StructMember) UnmarshalJSON(data []byte) error {
	var j structMemberJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*m = StructMember{
		BaseIndexable:        j.BaseIndexable,
		baseType:             j.BaseType,
		bitRange:             j.BitRange,
		inlinePendingResolve: j.InlinePendingResolve,
		expandedInline:       j.ExpandedInline,
		isStruct:             j.IsStruct,
		subStruct:            j.SubStruct,
	}
	return nil
}

type bitstructJSON struct {
	BaseIndexable
	BackingType Type            `json:"backingType"`
	Members     []*StructMember `json:"members,omitempty"`
	Implements  []string        `json:"implements,omitempty"`
}

func (b Bitstruct) MarshalJSON() ([]byte, error) {
	return json.Marshal(bitstructJSON{BaseIndexable: b.BaseIndexable, BackingType: b.backingType, Members: b.members, Implements: b.implements})
}

func (b *Bitstruct) UnmarshalJSON(data []byte) error {
	var j bitstructJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*b = Bitstruct{BaseIndexable: j.BaseIndexable, backingType: j.BackingType, members: j.Members, implements: j.Implements}
	return nil
}

type defJSON struct {
	BaseIndexable
	ResolvesTo     string               `json:"resolvesTo,omitempty"`
	ResolvesToType option.Option[*Type] `json:"resolvesToType"`
}

func (d Def) MarshalJSON() ([]byte, error) {
	return json.Marshal(defJSON{BaseIndexable: d.BaseIndexable, ResolvesTo: d.resolvesTo, ResolvesToType: d.resolvesToType})
}

func (d *Def) UnmarshalJSON(data []byte) error {
	var j defJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*d = Def{BaseIndexable: j.BaseIndexable, resolvesTo: j.ResolvesTo, resolvesToType: j.ResolvesToType}
	return nil
}

type distinctJSON struct {
	BaseIndexable
	BaseType *Type `json:"baseType,omitempty"`
	Inline   bool  `json:"inline,omitempty"`
}

func (d Distinct) MarshalJSON() ([]byte, error) {
	return json.Marshal(distinctJSON{BaseIndexable: d.BaseIndexable, BaseType: d.baseType, Inline: d.inline})
}

func (d *Distinct) UnmarshalJSON(data []byte) error {
	var j distinctJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*d = Distinct{BaseIndexable: j.BaseIndexable, baseType: j.BaseType, inline: j.Inline}
	return nil
}

type interfaceJSON struct {
	BaseIndexable
	Methods map[string]*Function `json:"methods,omitempty"`
}

func (i Interface) MarshalJSON() ([]byte, error) {
	return json.Marshal(interfaceJSON{BaseIndexable: i.BaseIndexable, Methods: i.methods})
}

func (i *Interface) UnmarshalJSON(data []byte) error {
	var j interfaceJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	if j.Methods == nil {
		j.Methods = make(map[string]*Function)
	}
	*i = Interface{BaseIndexable: j.BaseIndexable, methods: j.Methods}
	return nil
}
