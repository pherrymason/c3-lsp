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

func Visit[T any](node T, v ASTVisitor) {
	anyNode := any(node)
	switch anyNode.(type) {
	case *File:
		v.VisitFile(anyNode.(*File))
	case *Module:
		v.VisitModule(anyNode.(*Module))
	case *VariableDecl:
		v.VisitVariableDeclaration(anyNode.(*VariableDecl))
	case *LambdaDeclaration:
		v.VisitLambdaDeclaration(anyNode.(*LambdaDeclaration))
	case *IntegerLiteral:
		v.VisitIntegerLiteral(anyNode.(*IntegerLiteral))
	case *TypeInfo:
		v.VisitType(anyNode.(*TypeInfo))
	default:
		log.Print("type not found")
	}
}
