package ast

import "github.com/pherrymason/c3-lsp/pkg/option"

// ----------------------------------------------------------------------------
// Statements

// A statement is represented by a tree consisting of one
// or more of the following concrete statement nodes.
type (
	// An ExpressionStmt represents  (stand-alone) expression
	// in a statement list.
	ExpressionStmt struct {
		NodeAttributes
		Expr Expression
	}

	// A BlockStmt represents a braced statement list
	BlockStmt struct {
		NodeAttributes
		List []Statement
	}

	// A CompoundStmt TODO What's the difference with BlockStmt?
	CompoundStmt struct {
		NodeAttributes
		Statements []Statement
	}

	// DeclarationStmt represents a declaration in a statement list
	DeclarationStmt struct {
		NodeAttributes
		Decl Declaration
	}

	IfStmt struct {
		NodeAttributes
		Label     option.Option[string]
		Condition []*DeclOrExpr
		Statement Statement
		Else      ElseStatement
	}

	ElseStatement struct {
		NodeAttributes
		Statement Statement
	}

	ReturnStatement struct {
		NodeAttributes
		Return option.Option[Expression]
	}

	ContinueStatement struct {
		NodeAttributes
		Label option.Option[string]
	}

	BreakStatement struct {
		NodeAttributes
		Label option.Option[string]
	}

	SwitchStatement struct {
		NodeAttributes
		Label     option.Option[string]
		Condition []*DeclOrExpr
		Cases     []SwitchCase
		Default   []Statement
	}

	SwitchCase struct {
		NodeAttributes
		Value      Statement
		Statements []Statement
	}

	SwitchCaseRange struct {
		NodeAttributes
		Start Expression
		End   Expression
	}

	Nextcase struct {
		NodeAttributes
		Label option.Option[string]
		Value Expression
	}

	ForStatement struct {
		NodeAttributes
		Label       option.Option[string]
		Initializer []*DeclOrExpr
		Condition   Expression
		Update      []*DeclOrExpr
		Body        Statement
	}

	ForeachStatement struct {
		NodeAttributes
		Value      ForeachValue
		Index      ForeachValue
		Collection Expression
		Body       Statement
	}

	ForeachValue struct {
		Type       TypeInfo
		Identifier *Ident
	}

	WhileStatement struct {
		NodeAttributes
		Condition []*DeclOrExpr
		Body      Statement
	}

	DoStatement struct {
		NodeAttributes
		Condition Expression
		Body      Statement
	}

	DeferStatement struct {
		NodeAttributes
		Statement Statement
	}

	AssertStatement struct {
		NodeAttributes
		Assertions []Expression
	}
)

func (e *ExpressionStmt) stmtNode()    {}
func (e *BlockStmt) stmtNode()         {}
func (e *CompoundStmt) stmtNode()      {}
func (e *DeclarationStmt) stmtNode()   {}
func (e *IfStmt) stmtNode()            {}
func (e *ElseStatement) stmtNode()     {}
func (e *ReturnStatement) stmtNode()   {}
func (e *ContinueStatement) stmtNode() {}
func (e *BreakStatement) stmtNode()    {}
func (e *SwitchStatement) stmtNode()   {}
func (e *SwitchCase) stmtNode()        {}
func (e *SwitchCaseRange) stmtNode()   {}
func (e *Nextcase) stmtNode()          {}
func (e *ForStatement) stmtNode()      {}
func (e *ForeachStatement) stmtNode()  {}
func (e *WhileStatement) stmtNode()    {}
func (e *DoStatement) stmtNode()       {}
func (e *DeferStatement) stmtNode()    {}
func (e *AssertStatement) stmtNode()   {}
