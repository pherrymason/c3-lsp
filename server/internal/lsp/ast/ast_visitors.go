package ast

import (
	"log"
)

type ASTVisitor interface {
	VisitFile(node *File)
	VisitModule(node *Module)
	VisitImport(node *Import)
	VisitVariableDeclaration(node *VariableDecl)
	VisitConstDeclaration(node *ConstDecl)
	VisitEnumDecl(node *EnumDecl)
	VisitStructDecl(node *StructDecl)
	VisitFaultDecl(node *FaultDecl)
	VisitDefDecl(node *DefDecl)
	VisitMacroDecl(node *MacroDecl)
	VisitLambdaDeclaration(node *LambdaDeclarationExpr)
	VisitFunctionDecl(node *FunctionDecl)
	VisitFunctionParameter(node *FunctionParameter)
	VisitFunctionCall(node *FunctionCall)
	VisitInterfaceDecl(node *InterfaceDecl)
	VisitCompoundStatement(node *CompoundStmt)
	VisitType(node *TypeInfo)
	VisitIdentifier(node *Ident)
	VisitBinaryExpression(node *BinaryExpression)
	VisitIfStatement(node *IfStmt)
	VisitIntegerLiteral(node *IntegerLiteral)
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

	case *VariableDecl:
		v.VisitVariableDeclaration(node.(*VariableDecl))
	case *FunctionDecl:
		n := node.(*FunctionDecl)
		v.VisitFunctionDecl(n)

	case *FunctionParameter:
		n := node.(*FunctionParameter)
		v.VisitFunctionParameter(n)

	case *LambdaDeclarationExpr:
		v.VisitLambdaDeclaration(node.(*LambdaDeclarationExpr))

	case *CompoundStmt:
		v.VisitCompoundStatement(node.(*CompoundStmt))

	case *TypeInfo:
		v.VisitType(node.(*TypeInfo))

	case *IntegerLiteral:
		v.VisitIntegerLiteral(node.(*IntegerLiteral))
	default:
		log.Print("type not found")
	}
}
