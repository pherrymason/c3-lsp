package ast

type Expression interface {
	ASTNode
}

type Statement interface {
	ASTNode
}

type ExpressionStatement struct {
	ASTNode
	Expr Expression
}

type AssignmentStatement struct {
	ASTNode
	Left  Expression
	Right Expression
}
