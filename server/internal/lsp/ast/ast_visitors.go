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
	VisitFunctionParameter(node *FunctionParameter)
	VisitFunctionCall(node *FunctionCall)
	VisitInterfaceDecl(node *InterfaceDecl)
	VisitCompounStatement(node *CompoundStatement)
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
	switch node.TypeNode() {
	case TypeFile:
		v.VisitFile(node.(*File))
	case TypeModule:
		v.VisitModule(node.(*Module))

	case TypeVariableDecl:
		v.VisitVariableDeclaration(node.(*VariableDecl))
	case TypeFunctionDecl:
		n := node.(FunctionDecl)
		v.VisitFunctionDecl(&n)

	case TypeFunctionParameter:
		n := node.(FunctionParameter)
		v.VisitFunctionParameter(&n)

	case TypeLambdaDecl:
		v.VisitLambdaDeclaration(node.(*LambdaDeclaration))

	case TypeCompoundStatement:
		var arg *CompoundStatement
		switch node.(type) {
		case CompoundStatement:
			n := node.(CompoundStatement)
			arg = &n
		case *CompoundStatement:
			arg = node.(*CompoundStatement)
		}
		v.VisitCompounStatement(arg)

	case TypeTypeInfo:
		n := node.(*TypeInfo)
		v.VisitType(n)

	case TypeIntegerLiteral:
		n := node.(IntegerLiteral)
		v.VisitIntegerLiteral(&n)
	default:
		log.Print("type not found")
	}
}
