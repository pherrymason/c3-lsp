package ast

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
)

// -------------------------------------------------------------------------
// Expressions and types

// Field : A Field represents a Field declaration list in a struct type, ¿TODO?
// FieldList : A FieldList represents a list of Fields enclosed by parenthesis, curly
// braces or square braces, ¿TODO?

// An expression is represented by a tree consisting of one
// or more of the following concrete expression nodes.
type (
	// Ident Represent a defined element.
	// myValue
	// myObject.property
	// myObject.subobject.property
	// It can be built
	Ident struct {
		NodeAttributes
		Name       string // Identifier name
		ModulePath string // Module path. Some identifiers, specify module path.
	}

	Path struct {
		NodeAttributes
		PathType  int
		Path      string
		PathStart string
		PathEnd   string
		FieldName string
	}

	ParenExpr struct {
		NodeAttributes
		X Expression
	}

	SelectorExpr struct {
		NodeAttributes
		X   Expression //
		Sel *Ident     //
	}

	// Ellipsis TODO

	BasicLit struct {
		NodeAttributes
		Kind  Token  // ast.INT, ast.FLOAT, ast.IMAG, ast.CHAR, or ast.STRING, ast.BOOLEAN
		Value string // literal string
	}

	CompositeLiteral struct {
		NodeAttributes
		Elements []Expression // list of composite elements
	}

	IndexAccessExpr struct {
		NodeAttributes
		Array Expression
		Index string
	}

	// RangeAccessExpr TODO Replace by RangeIndexExpr
	RangeAccessExpr struct {
		NodeAttributes
		Array      Expression
		RangeStart uint
		RangeEnd   uint
	}

	// RangeIndexExpr TODO document this node
	RangeIndexExpr struct {
		NodeAttributes

		Start option.Option[uint]
		End   option.Option[uint]
	}

	// SubscriptExpression TODO document this node
	SubscriptExpression struct {
		NodeAttributes
		Argument Expression
		Index    Expression // Index can be another expression:
		//  - RangeIndexExpr
		//  - Ident
		//  - CallExpression
		//  - ...
	}

	// FieldAccessExpr TODO document this node
	FieldAccessExpr struct {
		NodeAttributes
		Object Expression
		Field  Expression
	}

	AssignmentExpression struct {
		NodeAttributes
		Left     Expression
		Right    Expression
		Operator string
	}

	// A StarExpr node represents an expression of the form "*" Expression.
	// Semantically it could be a unary "*" expression, or a pointer type.
	StarExpr struct {
		NodeAttributes
		X Expression
	}

	// A FunctionCall node represents an expression followed by an argument list.
	FunctionCall struct {
		NodeAttributes
		Identifier       Expression
		GenericArguments option.Option[[]Expression]
		Arguments        []Expression
		TrailingBlock    option.Option[*CompoundStmt]
	}

	LambdaDeclarationExpr struct {
		NodeAttributes
		Parameters []FunctionParameter
		ReturnType option.Option[TypeInfo]
		Body       Statement
	}

	// A UnaryExpression
	UnaryExpression struct {
		NodeAttributes
		Operator string
		Argument Expression
	}

	// BinaryExpression represents a binary expression (like sum, subtract, etc.)
	BinaryExpression struct {
		NodeAttributes
		Left     Node
		Operator string
		Right    Node
	}

	OptionalExpression struct {
		NodeAttributes
		Argument Expression
		Operator string
	}

	CastExpression struct {
		NodeAttributes
		Type     TypeInfo
		Argument Expression
	}

	RethrowExpression struct {
		NodeAttributes
		Operator string
		Argument Expression
	}

	TernaryExpression struct {
		NodeAttributes
		Condition   Expression
		Consequence Expression
		Alternative Expression
	}

	UpdateExpression struct {
		NodeAttributes
		Operator string
		Argument Expression
	}

	// InlineTypeWithInitialization
	// TODO I thing this is a Statement
	InlineTypeWithInitialization struct {
		NodeAttributes
		Type            TypeInfo
		InitializerList *InitializerList
	}

	// InitializerList
	// TODO I thing this is a Statement
	InitializerList struct {
		NodeAttributes
		Args []Expression
	}

	ArgParamPathSet struct {
		NodeAttributes
		Path string
		Expr Expression
	}

	ArgFieldSet struct {
		NodeAttributes
		FieldName string
		Expr      Expression
	}
)

const (
	PathTypeIndexed = iota
	PathTypeField
	PathTypeRange
)

func (arg *ArgFieldSet) SetExpr(expr Expression) {
	arg.Expr = expr
}

// A type is represented by a tree consisting of one
// or more of the following type-specific expression nodes.
type (
	TypeInfo struct {
		NodeAttributes
		ResolveStatus int
		Identifier    Ident
		Pointer       uint
		Optional      bool
		BuiltIn       bool
		Static        bool
		Reference     bool
		TLocal        bool
		Generics      []TypeInfo
	}

	BaseType struct {
		NodeAttributes
		Name *Ident
	}

	Field struct {
		NodeAttributes
		Name  *Ident
		Type  TypeInfo
		Value Expression // Value if applicable.
	}

	// An ArrayType represents an array or slice type.
	ArrayType struct {
		NodeAttributes
		Len Expression // length of the array
		Elt Expression // element type
	}

	// A bStructType represents a struct type
	bStructType struct {
		NodeAttributes
		Fields []Expression
	}

	EnumType struct {
		NodeAttributes
		BaseType option.Option[TypeInfo] // Enums can be typed.
		Fields   []Expression            // Enums in C3 can have fields
		Values   []*EnumValue
	}

	EnumValue struct {
		Name  *Ident
		Value Expression
	}
	/*
		FuncType struct {
			NodeAttributes
			Params []Expression
			Result []Expression
		}
	*/
	FunctionSignature struct {
		NodeAttributes
		Name       Ident
		Parameters []FunctionParameter
		ReturnType TypeInfo
	}

	InterfaceType struct {
		NodeAttributes
		Methods []Expression
	}

	// TrailingGenericsExpr Used only as a temporal container.
	// It is decomposed and its info extracted to build other ast nodes.
	TrailingGenericsExpr struct {
		NodeAttributes
		Identifier       *Ident
		GenericArguments []Expression
	}
)

func (*ArgFieldSet) exprNode()             {}
func (*ArgParamPathSet) exprNode()         {}
func (e *AssignmentExpression) exprNode()  {}
func (Ident) exprNode()                    {}
func (*ParenExpr) exprNode()               {}
func (SelectorExpr) exprNode()             {}
func (Path) exprNode()                     {}
func (*BaseType) exprNode()                {}
func (e *BasicLit) exprNode()              {}
func (l *CompositeLiteral) exprNode()      {}
func (l *IndexAccessExpr) exprNode()       {}
func (l *RangeAccessExpr) exprNode()       {}
func (l *RangeIndexExpr) exprNode()        {}
func (l *SubscriptExpression) exprNode()   {}
func (l *FieldAccessExpr) exprNode()       {}
func (l *FunctionCall) exprNode()          {}
func (v *LambdaDeclarationExpr) exprNode() {}
func (l *UnaryExpression) exprNode()       {}
func (l *BinaryExpression) exprNode()      {}
func (l *OptionalExpression) exprNode()    {}
func (l *CastExpression) exprNode()        {}
func (l *RethrowExpression) exprNode()     {}
func (l *TernaryExpression) exprNode()     {}
func (l *UpdateExpression) exprNode()      {}

func (TypeInfo) exprNode()                      {}
func (*InitializerList) exprNode()              {}
func (*InlineTypeWithInitialization) exprNode() {}
func (l *Field) exprNode()                      {}
func (l *ArrayType) exprNode()                  {}
func (l *EnumType) exprNode()                   {}
func (l *bStructType) exprNode()                {}
func (l *InterfaceType) exprNode()              {}
func (l *TrailingGenericsExpr) exprNode()       {}
