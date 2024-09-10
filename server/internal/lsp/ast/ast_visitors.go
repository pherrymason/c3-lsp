package ast

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
	switch any(node).(type) {
	case *File:
		v.VisitFile(any(node).(*File))
	case *Module:
		v.VisitModule(any(node).(*Module))
	case *VariableDecl:
		v.VisitVariableDeclaration(any(node).(*VariableDecl))
	case *LambdaDeclaration:
		v.VisitLambdaDeclaration(any(node).(*LambdaDeclaration))
	case *IntegerLiteral:
		v.VisitIntegerLiteral(any(node).(*IntegerLiteral))
	case *TypeInfo:
		v.VisitType(any(node).(*TypeInfo))
	}
}
