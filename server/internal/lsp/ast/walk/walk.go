package walk

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	goAst "go/ast"
)

type WalkContext struct {
	File   ast.File // File Node
	Scopes goAst.Scope
}

// Visitor Enter method is invoked for each node encountered by Walk.
// If the result visitor w is not nil, Walk visits each of the children
// of node with the visitor v, followed by a call of the Exit method.
type Visitor interface {
	Enter(n ast.Node, propertyName string) (v Visitor)
	Exit(n ast.Node, propertyName string)
}

func walkList[N ast.Node](v Visitor, list []N, propertyName string) {
	for _, node := range list {
		Walk(v, node, propertyName)
	}
}

// Walk traverses an AST in depth-first order: It starts by calling
// v.Enter(node); node must not be nil. If the visitor v returned by
// v.Enter(node) is not nil, Walk is invoked recursively with visitor
// v for each of the non-nil children of node, followed by a call
// of v.Exit(node).
// Alternative Walk method signature Walk(v Visitor, n Node, p *Path)
func Walk(v Visitor, node ast.Node, propertyName string) {
	if node == nil {
		return
	}

	if v = v.Enter(node, propertyName); v == nil {
		return
	}

	defer v.Exit(node, propertyName)

	switch n := node.(type) {
	// Comments and fields
	//case *Comment:
	// Nothing to do
	//case *CommentGroup:
	// Nothing to do

	case *ast.File:
		walkList(v, n.Modules, "Modules")

	case ast.Module:
		walkList(v, n.Imports, "Imports")
		walkList(v, n.Declarations, "Declarations")

	case ast.Import:
		// Do nothing

	// Expressions
	case ast.Ident:
		if n.ModulePath != nil {
			Walk(v, n.ModulePath, "ModulePath")
		}

	case *ast.Ident:
		if n.ModulePath != nil {
			Walk(v, n.ModulePath, "ModulePath")
		}

	case *ast.BasicLit:
	// Nothing to do

	case *ast.ArgFieldSet:
		Walk(v, n.Expr, "Expr")

	case *ast.ArgParamPathSet:
		Walk(v, n.Expr, "Expr")

	case *ast.AssertStatement:
		walkList(v, n.Assertions, "Assertions")

	case *ast.AssignmentExpression:
		Walk(v, n.Left, "Left")
		Walk(v, n.Right, "Right")

	case *ast.BlockExpr:
		walkList(v, n.List, "List")

	case *ast.BreakStatement:
		// TODO label

	case *ast.BinaryExpression:
		Walk(v, n.Left, "Left")
		Walk(v, n.Right, "Right")

	case *ast.CastExpression:
		Walk(v, n.Type, "Type")
		Walk(v, n.Argument, "Argument")

	case *ast.CompoundStmt:
		walkList(v, n.Statements, "Statements")

	case *ast.CompositeLiteral:
		walkList(v, n.Elements, "Elements")

	case *ast.ContinueStatement:
		// TODO label

	case *ast.DeclarationStmt:
		Walk(v, n.Decl, "Decl")

	case *ast.DeferStatement:
		Walk(v, n.Statement, "Statement")

	case *ast.DoStatement:
		Walk(v, n.Condition, "Condition")
		Walk(v, n.Body, "Body")

	case *ast.EnumType:
		if n.BaseType.IsSome() {
			Walk(v, n.BaseType.Get(), "BaseType")
		}
		walkList(v, n.StaticValues, "StaticValues")
		walkList(v, n.Values, "Values")

	case *ast.ElseStatement:
		Walk(v, n.Statement, "Statement")

	case *ast.ExpressionStmt:
		Walk(v, n.Expr, "Expr")

	case *ast.FaultDecl:
		Walk(v, n.Name, "Name")
		if n.BackingType.IsSome() {
			Walk(v, n.BackingType.Get(), "BackingType")
		}
		walkList(v, n.Members, "Members")

	case ast.FaultMember:
		Walk(v, n.Name, "Name")

	case *ast.FunctionDecl:
		if n.ParentTypeId.IsSome() {
			Walk(v, n.ParentTypeId.Get(), "ParentTypeId")
		}
		Walk(v, n.Signature, "Signature")
		Walk(v, n.Body, "Body")

	case *ast.FieldAccessExpr:
		Walk(v, n.Object, "Object")
		Walk(v, n.Field, "Field")

	case *ast.ForeachStatement:
		Walk(v, n.Value, "Value")
		Walk(v, n.Index, "Index")
		Walk(v, n.Collection, "Collection")
		Walk(v, n.Body, "Body")

	case *ast.ForStatement:
		if n.Initializer != nil {
			walkList(v, n.Initializer, "Initializer")
		}
		Walk(v, n.Condition, "Condition")
		if n.Update != nil {
			walkList(v, n.Update, "Update")
		}
		Walk(v, n.Body, "Body")

	case *ast.FunctionCall:
		Walk(v, n.Identifier, "Identifier")
		walkList(v, n.Arguments, "Arguments")
		if n.TrailingBlock.IsSome() {
			Walk(v, n.TrailingBlock.Get(), "TrailingBlock")
		}

	case ast.FunctionSignature:
		Walk(v, n.Name, "URI")
		walkList(v, n.Parameters, "Parameters")
		Walk(v, n.ReturnType, "ReturnType")

	case *ast.GenDecl:
		Walk(v, n.Spec, "Spec")

	case *ast.IfStmt:
		// TODO Label
		if n.Condition != nil {
			walkList(v, n.Condition, "Condition")
		}
		Walk(v, n.Statement, "Statement")
		Walk(v, n.Else, "Else")

	case *ast.IndexAccessExpr:
		Walk(v, n.Array, "Array")

	case *ast.InlineTypeWithInitialization:
		Walk(v, n.Type, "Type")
		if n.InitializerList != nil {
			Walk(v, n.InitializerList, "InitializerList")
		}

	case *ast.InitializerList:
		walkList(v, n.Args, "Args")

	case *ast.LambdaDeclarationExpr:
		walkList(v, n.Parameters, "Parameters")
		if n.ReturnType.IsSome() {
			Walk(v, n.ReturnType.Get(), "ReturnType")
		}
		Walk(v, n.Body, "Body")

	case *ast.ParenExpr:
		Walk(v, n.X, "X")

	case *ast.Nextcase:
		if n.Label.IsSome() {
			// Nothing to do! Label is string
		}
		Walk(v, n.Value, "Value")

	case *ast.OptionalExpression:
		Walk(v, n.Argument, "Argument")

	case *ast.RangeAccessExpr:
		Walk(v, n.Array, "Array")

	case *ast.RangeIndexExpr:

	case *ast.RethrowExpression:
		Walk(v, n.Argument, "Argument")

	case *ast.ReturnStatement:
		if n.Return.IsSome() {
			Walk(v, n.Return.Get(), "Return")
		}

	case *ast.SelectorExpr:
		Walk(v, n.X, "X")
		Walk(v, n.Sel, "Sel")

	case *ast.StarExpr:
		/*
			case *ast.StructDecl:
				walkList(v, n.Implements, "Implements")
				walkList(v, n.Members, "Members")

			case ast.StructMemberDecl:
				Walk(v, n.Type, "Type")
				walkList(v, n.Names, "Names")
		*/

	case *ast.SubscriptExpression:
		Walk(v, n.Argument, "Argument")
		Walk(v, n.Index, "Index")

	case ast.SwitchCase:
		Walk(v, n.Value, "Value")
		walkList(v, n.Statements, "Statements")

	case ast.SwitchCaseRange:
		Walk(v, n.Start, "Start")
		Walk(v, n.End, "End")

	case *ast.SwitchStatement:
		walkList(v, n.Condition, "Condition")
		walkList(v, n.Cases, "Cases")
		walkList(v, n.Default, "Default")

	case *ast.TernaryExpression:
		Walk(v, n.Condition, "Condition")
		Walk(v, n.Consequence, "Consequence")
		Walk(v, n.Alternative, "Alternative")

	case *ast.TypeSpec:
		Walk(v, n.Name, "Name")
		walkList(v, n.TypeParams, "TypeParams")
		Walk(v, n.TypeDescription, "TypeDescription")

	case ast.TypeInfo:
		if n.Identifier != nil {
			Walk(v, n.Identifier, "Identifier")
		}

	case *ast.UnaryExpression:
		Walk(v, n.Argument, "Argument")

	case *ast.UpdateExpression:
		Walk(v, n.Argument, "Argument")

	case *ast.ValueSpec:
		Walk(v, n.Type, "Type")
		walkList(v, n.Names, "Names")
		Walk(v, n.Value, "Value")

	case *ast.StructType:
		walkList(v, n.Implements, "Implements")
		if n.BackingType.IsSome() {
			Walk(v, n.BackingType.Get(), "BackingType")
		}
		walkList(v, n.Fields, "Fields")

	case *ast.WhileStatement:
		if n.Condition != nil {
			walkList(v, n.Condition, "Condition")
		}
		Walk(v, n.Body, "Body")

	default:
		//panic(fmt.Sprintf("Walk: unexpected node type %T", n))
	}
}
