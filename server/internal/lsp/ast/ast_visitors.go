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
	VisitLambdaDeclaration(node *LambdaDeclaration)
	VisitFunctionDecl(node *FunctionDecl)
	VisitFunctionCall(node *FunctionCall)
	VisitInterfaceDecl(node *InterfaceDecl)
	VisitType(node *TypeInfo)
	VisitIdentifier(node *Identifier)
	VisitBinaryExpression(node *BinaryExpression)
	VisitIfStatement(node *IfStatement)
	VisitIntegerLiteral(node *IntegerLiteral)
}

type VisitableNode interface {
	Accept(visitor ASTVisitor)
}

// ----------------------------------------

func Visit(node ASTNode, v ASTVisitor) {
	switch node.(type) {
	case *File:
		v.VisitFile(node.(*File))
	case *Module:
		v.VisitModule(node.(*Module))

	case *VariableDecl:
		v.VisitVariableDeclaration(node.(*VariableDecl))
	case *LambdaDeclaration:
		v.VisitLambdaDeclaration(node.(*LambdaDeclaration))
	case *IntegerLiteral:
		v.VisitIntegerLiteral(node.(*IntegerLiteral))
	case *TypeInfo:
		v.VisitType(node.(*TypeInfo))
	default:
		log.Print("type not found")
	}
}
