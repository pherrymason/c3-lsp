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
		//Node
		specNode()
	}

	// TODO Not yet used
	ImportSpec struct {
		//NodeAttributes
		Path string
	}

	ValueSpec struct {
		//NodeAttributes
		Names []*Ident
		Type  *TypeInfo  // value type, or nil
		Value Expression // initial values, or nil
	}

	// TypeSpec represents declarations of types like aliases, definition of types
	// or parametrized types (generics)
	TypeSpec struct {
		//NodeAttributes
		Ident           *Ident       // type name
		TypeParams      []Expression // Generic type parameters; or nil
		Assign          token.Pos    // position of '=', if any
		TypeDescription Expression   // ast node describing the type with detail: EnumType, bStructType, StructType, DefType (TODO), DistinctType (TODO)
	}

	// TODO Use TypeSpec instead! It covers everything for defs

	DefSpec struct {
		Name              *Ident
		Value             Expression
		GenericParameters []*TypeInfo
		ResolvesToType    bool
	}
)

func (i *ImportSpec) specNode() {}
func (v *ValueSpec) specNode()  {}
func (v *TypeSpec) specNode()   {}
func (v *DefSpec) specNode()    {}

const (
	StructTypeNormal = iota
	StructTypeUnion
	StructTypeBitStruct
)

type StructTypeID int

type (
	GenDecl struct {
		NodeAttributes
		Token Token // const, variable
		Spec  Spec
	}

	// TODO: move to GenDecl
	FaultDecl struct {
		NodeAttributes
		Name        *Ident
		BackingType option.Option[*TypeInfo]
		Members     []*FaultMember
	}

	MacroDecl struct {
		NodeAttributes
		Signature *MacroSignature
		Body      *CompoundStmt
	}
	MacroSignature struct {
		ParentTypeId       option.Option[*Ident]
		Name               *Ident
		Parameters         []*FunctionParameter
		TrailingBlockParam *TrailingBlockParam
		ReturnType         *TypeInfo
	}

	TrailingBlockParam struct {
		NodeAttributes
		Name       *Ident
		Parameters []*FunctionParameter
	}

	// DefDecl can be used for
	// defining a new type: def Int32 = int
	// defining a pointer type: def Callback = fn void(int value);
	DefDecl struct {
		NodeAttributes
		Ident          *Ident
		Expr           Expression
		ResolvesToType option.Option[*TypeInfo] // Deprecated
	}

	FunctionDecl struct {
		NodeAttributes
		ParentTypeId option.Option[*Ident]
		Signature    *FunctionSignature
		Body         Node
	}

	FunctionSignature struct {
		NodeAttributes
		Name       *Ident
		Parameters []*FunctionParameter
		ReturnType *TypeInfo
	}

	InterfaceDecl struct {
		NodeAttributes
		Name    *Ident
		Methods []*FunctionSignature
	}
)

func (v *GenDecl) declNode()            {}
func (v *FaultDecl) declNode()          {}
func (v *DefDecl) declNode()            {}
func (v *MacroDecl) declNode()          {}
func (v *MacroSignature) declNode()     {}
func (v *TrailingBlockParam) declNode() {}

func (v *FunctionDecl) declNode()  {}
func (v *InterfaceDecl) declNode() {}
