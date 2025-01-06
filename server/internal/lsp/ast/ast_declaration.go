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
		NodeAttributes
		Names []*Ident
		Type  Expression // value type, or nil
		Value Expression // initial values, or nil
	}

	// TypeSpec represents declarations of types like aliases, definition of types
	// or parametrized types (generics)
	TypeSpec struct {
		NodeAttributes
		Name            *Ident       // type name
		TypeParams      []Expression // Generic type parameters; or nil
		Assign          token.Pos    // position of '=', if any
		TypeDescription Expression   // ast node describing the type with detail: EnumType, bStructType
	}
)

func (i *ImportSpec) specNode() {}
func (v *ValueSpec) specNode()  {}
func (v *TypeSpec) specNode()   {}

const (
	StructTypeNormal = iota
	StructTypeUnion
	StructTypeBitStruct
)

type StructType int

type (
	GenDecl struct {
		NodeAttributes
		Token Token // const, variable
		Spec  Spec
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

	// DefDecl can be used for
	// defining a new type: def Int32 = int
	// defining a pointer type: def Callback = fn void(int value);
	DefDecl struct {
		NodeAttributes
		Name           Ident
		Expr           Expression
		ResolvesToType option.Option[TypeInfo] // Deprecated
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

func (v *GenDecl) declNode()    {}
func (v *FaultDecl) declNode()  {}
func (v *StructDecl) declNode() {}
func (v *DefDecl) declNode()    {}
func (v *MacroDecl) declNode()  {}

func (v *FunctionDecl) declNode()  {}
func (v *InterfaceDecl) declNode() {}
