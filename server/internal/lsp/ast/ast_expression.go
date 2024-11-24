package ast

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	"go/token"
)

// -------------------------------------------------------------------------
// Expressions and types

// Field : A Field represents a Field declaration list in a struct type, ¿TODO?
// FieldList : A FieldList represents a list of Fields enclosed by parenthesis, curly
// braces or square braces, ¿TODO?

// An expression is represented by a tree consisting of one
// or more of the following concrete expression nodes.
type (
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

	SelectorExpr struct {
		NodeAttributes
		X   Expression
		Sel Ident
	}

	// Ellipsis TODO

	// Literal
	// Deprecated use BasicLit
	Literal struct {
		NodeAttributes
		Value string
	}

	// IntegerLiteral
	// Deprecated use BasicLit
	IntegerLiteral struct {
		NodeAttributes
		Value string
	}

	// RealLiteral
	// Deprecated use BasicLit
	RealLiteral struct {
		NodeAttributes
		Value string
	}

	// BoolLiteral
	// Deprecated use BasicLit
	BoolLiteral struct {
		NodeAttributes
		Value bool
	}

	BasicLit struct {
		NodeAttributes
		Kind  token.Token // token.INT, token.FLOAT, token.IMAG, token.CHAR, or token.STRING
		Value string      // literal string
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
		//  - IntegerLiteral
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

	ArrayType struct {
		NodeAttributes
		Len Expression // length of the array
		Elt Expression // element type
	}

	bStructType struct {
		NodeAttributes
		Fields []Expression
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
		Identifier       Ident
		GenericArguments []Expression
	}
)

func (*ArgFieldSet) exprNode()             {}
func (*ArgParamPathSet) exprNode()         {}
func (Ident) exprNode()                    {}
func (SelectorExpr) exprNode()             {}
func (Path) exprNode()                     {}
func (e *BasicLit) exprNode()              {}
func (l *Literal) exprNode()               {}
func (l *IntegerLiteral) exprNode()        {}
func (l *RealLiteral) exprNode()           {}
func (l *BoolLiteral) exprNode()           {}
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
func (l *ArrayType) exprNode()                  {}
func (l *bStructType) exprNode()                {}
func (l *InterfaceType) exprNode()              {}
func (l *TrailingGenericsExpr) exprNode()       {}
