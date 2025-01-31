package ast

import (
	"log"
)

type ASTVisitor interface {
	VisitFile(node *File)
	VisitModule(node *Module)
	VisitImport(node *Import)

	VisitFunctionParameter(node *FunctionParameter)

	// ----------------------------
	// Declarations

	VisitFunctionDecl(node *FunctionDecl)
	VisitStructDecl(node *StructDecl)
	VisitFaultDecl(node *FaultDecl)
	VisitDefDecl(node *DefDecl)
	VisitMacroDecl(node *MacroDecl)
	VisitLambdaDeclaration(node *LambdaDeclarationExpr)

	VisitInterfaceDecl(node *InterfaceDecl)

	// ----------------------------
	// Expressions

	VisitType(node *TypeInfo)
	VisitIdentifier(node *Ident)
	VisitSelectorExpr(node *SelectorExpr)
	VisitBinaryExpression(node *BinaryExpression)
	VisitBasicLiteral(node *BasicLit)
	VisitFunctionCall(node *CallExpr)

	// ----------------------------
	// Statements

	VisitExpressionStatement(node *ExpressionStmt)
	VisitCompoundStatement(node *CompoundStmt)
	VisitIfStatement(node *IfStmt)
}

type VisitableNode interface {
	Accept(visitor ASTVisitor)
}

// ----------------------------------------

func Visit(node Node, v ASTVisitor) {
	switch node.(type) {
	case *File:
		v.VisitFile(node.(*File))
	case *Module:
		v.VisitModule(node.(*Module))

	case *FunctionDecl:
		n := node.(*FunctionDecl)
		v.VisitFunctionDecl(n)

	case *FunctionParameter:
		n := node.(*FunctionParameter)
		v.VisitFunctionParameter(n)

	// ----------------------------
	// Expressions
	case *Ident:
		v.VisitIdentifier(node.(*Ident))

	case *TypeInfo:
		v.VisitType(node.(*TypeInfo))

	case *SelectorExpr:
		v.VisitSelectorExpr(node.(*SelectorExpr))
	case *BasicLit:
		v.VisitBasicLiteral(node.(*BasicLit))

	case *LambdaDeclarationExpr:
		v.VisitLambdaDeclaration(node.(*LambdaDeclarationExpr))

	// ----------------------------
	// Statements

	case *ExpressionStmt:
		v.VisitExpressionStatement(node.(*ExpressionStmt))

	case *CompoundStmt:
		v.VisitCompoundStatement(node.(*CompoundStmt))

	case *CallExpr:
		v.VisitFunctionCall(node.(*CallExpr))

	default:
		log.Print("type not found")
	}
}
