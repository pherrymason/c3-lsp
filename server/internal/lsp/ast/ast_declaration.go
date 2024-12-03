package ast

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	"go/token"
)

// ----------------------------------------------------------------------------
// Declarations

type (
	// The Spec type stands for any of *ImportSpec, *ValueSpec, and *TypeSpec.
	Spec interface {
		Node
		specNode()
	}

	ImportSpec struct {
		NodeAttributes
		Path string
	}

	ValueSpec struct {
		Names  []*Ident
		Type   Expression   // value type, or nil
		Values []Expression // initial values, or nil
	}

	// TypeSpec represents declarations of types like aliases, definition of types
	// or parametrized types (generics)
	TypeSpec struct {
		Name       *Ident       // type name
		TypeParams []Expression // type parameters; or nil
		Assign     token.Pos    // position of '=', if any
		Type       Expression   // *Ident, *ParenExpr, *SelectorExpr, *StarExpr, or any of the *XxxTypes
	}
)

func (*ImportSpec) specNode() {}
func (*ValueSpec) specNode()  {}

const (
	StructTypeNormal = iota
	StructTypeUnion
	StructTypeBitStruct
)

type StructType int

type (
	// VariableDecl
	// Deprecated use GenDecl with Token as token.VAR
	VariableDecl struct {
		NodeAttributes
		Names       []*Ident
		Type        TypeInfo
		Initializer Expression
	}

	// ConstDecl
	// Deprecated use GenDecl with Token as token.CONST
	ConstDecl struct {
		NodeAttributes
		Names       []*Ident
		Type        option.Option[TypeInfo]
		Initializer Expression
	}

	GenDecl struct {
		NodeAttributes
		Token token.Token
		Specs []Spec
	}

	EnumDecl struct {
		NodeAttributes
		Name       string
		BaseType   TypeInfo
		Properties []EnumProperty
		Members    []EnumMember
	}

	FaultDecl struct {
		NodeAttributes
		Name        Ident
		BackingType option.Option[TypeInfo]
		Members     []FaultMember
	}

	MacroDecl struct {
		NodeAttributes
		Signature MacroSignature
		Body      Block
	}

	DefDecl struct {
		NodeAttributes
		Name           Ident
		resolvesTo     string
		resolvesToType option.Option[TypeInfo]
	}

	StructDecl struct {
		NodeAttributes
		Name        string
		BackingType option.Option[TypeInfo]
		Members     []StructMemberDecl
		StructType  StructType
		Implements  []string
	}

	FunctionDecl struct {
		NodeAttributes
		ParentTypeId option.Option[Ident]
		Signature    FunctionSignature
		Body         Node
	}

	InterfaceDecl struct {
		NodeAttributes
		Name    Ident
		Methods []FunctionSignature
	}
)

func (v *VariableDecl) declNode() {}
func (v *ConstDecl) declNode()    {}
func (v *EnumDecl) declNode()     {}
func (v *FaultDecl) declNode()    {}
func (v *StructDecl) declNode()   {}
func (v *DefDecl) declNode()      {}
func (v *MacroDecl) declNode()    {}

func (v *FunctionDecl) declNode()  {}
func (v *InterfaceDecl) declNode() {}
