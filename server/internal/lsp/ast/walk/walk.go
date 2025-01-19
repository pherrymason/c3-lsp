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
func Walk(v Visitor, n ast.Node, propertyName string) {
	if n == nil {
		return
	}

	if v = v.Enter(n, propertyName); v == nil {
		return
	}

	defer v.Exit(n, propertyName)

	switch n := n.(type) {
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
	case *ast.Ident, *ast.BasicLit:
		// Nothing to do

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

	case *ast.LambdaDeclarationExpr:
		walkList(v, n.Parameters, "Parameters")
		Walk(v, n.Body, "Body")

	case *ast.Nextcase:
		if n.Label.IsSome() {
			// Nothing to do! Label is string
		}
		Walk(v, n.Value, "Value")

	case *ast.RangeAccessExpr:
		Walk(v, n.Array, "Array")

	case *ast.ReturnStatement:
		if n.Return.IsSome() {
			Walk(v, n.Return.Get(), "Return")
		}

	case *ast.SelectorExpr:
		Walk(v, n.X, "X")
		Walk(v, n.Sel, "Sel")

	case *ast.StarExpr:

	case *ast.StructDecl:
		walkList(v, n.Members, "Members")

	case ast.StructMemberDecl:
		Walk(v, n.Type, "Type")
		walkList(v, n.Names, "Names")

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

	case *ast.TypeSpec:
		Walk(v, n.Name, "Name")
		walkList(v, n.TypeParams, "TypeParams")
		Walk(v, n.TypeDescription, "TypeDescription")

	case ast.TypeInfo:
		Walk(v, n.Identifier, "Identifier")

	case *ast.ValueSpec:
		Walk(v, n.Type, "Type")
		walkList(v, n.Names, "Names")
		Walk(v, n.Value, "Value")

	case *ast.WhileStatement:
		if n.Condition != nil {
			walkList(v, n.Condition, "Condition")
		}
		Walk(v, n.Body, "Body")
		/*
			case *AssignExpression:
				if n != nil {
					Walk(v, n.Left)
					Walk(v, n.Right)
				}
			case *BadExpression:
			case *BadStatement:
			case *BinaryExpression:
				if n != nil {
					Walk(v, n.Left)
					Walk(v, n.Right)
				}
			case *BlockStatement:
				if n != nil {
					for _, s := range n.List {
						Walk(v, s)
					}
				}
			case *BooleanLiteral:
			case *BracketExpression:
				if n != nil {
					Walk(v, n.Left)
					Walk(v, n.Member)
				}
			case *BranchStatement:
				if n != nil {
					Walk(v, n.Label)
				}
			case *CallExpression:
				if n != nil {
					Walk(v, n.Callee)
					for _, a := range n.ArgumentList {
						Walk(v, a)
					}
				}
			case *CaseStatement:
				if n != nil {
					Walk(v, n.Test)
					for _, c := range n.Consequent {
						Walk(v, c)
					}
				}
			case *CatchStatement:
				if n != nil {
					Walk(v, n.Parameter)
					Walk(v, n.Body)
				}
			case *ConditionalExpression:
				if n != nil {
					Walk(v, n.Test)
					Walk(v, n.Consequent)
					Walk(v, n.Alternate)
				}
			case *DebuggerStatement:
			case *DoWhileStatement:
				if n != nil {
					Walk(v, n.Test)
					Walk(v, n.Body)
				}
			case *DotExpression:
				if n != nil {
					Walk(v, n.Left)
					Walk(v, n.Identifier)
				}
			case *EmptyExpression:
			case *EmptyStatement:
			case *ExpressionStatement:
				if n != nil {
					Walk(v, n.Expression)
				}
			case *ForInStatement:
				if n != nil {
					Walk(v, n.Into)
					Walk(v, n.Source)
					Walk(v, n.Body)
				}
			case *ForStatement:
				if n != nil {
					Walk(v, n.Initializer)
					Walk(v, n.Update)
					Walk(v, n.Test)
					Walk(v, n.Body)
				}
			case *FunctionLiteral:
				if n != nil {
					Walk(v, n.URI)
					for _, p := range n.ParameterList.List {
						Walk(v, p)
					}
					Walk(v, n.Body)
				}
			case *FunctionStatement:
				if n != nil {
					Walk(v, n.Function)
				}
			case *Identifier:
			case *IfStatement:
				if n != nil {
					Walk(v, n.Test)
					Walk(v, n.Consequent)
					Walk(v, n.Alternate)
				}
			case *LabelledStatement:
				if n != nil {
					Walk(v, n.Label)
					Walk(v, n.Statement)
				}
			case *NewExpression:
				if n != nil {
					Walk(v, n.Callee)
					for _, a := range n.ArgumentList {
						Walk(v, a)
					}
				}
			case *NullLiteral:
			case *NumberLiteral:
			case *ObjectLiteral:
				if n != nil {
					for _, p := range n.Value {
						Walk(v, p.Value)
					}
				}
			case *Program:
				if n != nil {
					for _, b := range n.Body {
						Walk(v, b)
					}
				}
			case *RegExpLiteral:
			case *ReturnStatement:
				if n != nil {
					Walk(v, n.Argument)
				}
			case *SequenceExpression:
				if n != nil {
					for _, e := range n.Sequence {
						Walk(v, e)
					}
				}
			case *StringLiteral:
			case *SwitchStatement:
				if n != nil {
					Walk(v, n.Discriminant)
					for _, c := range n.Body {
						Walk(v, c)
					}
				}
			case *ThisExpression:
			case *ThrowStatement:
				if n != nil {
					Walk(v, n.Argument)
				}
			case *TryStatement:
				if n != nil {
					Walk(v, n.Body)
					Walk(v, n.Catch)
					Walk(v, n.Finally)
				}
			case *UnaryExpression:
				if n != nil {
					Walk(v, n.Operand)
				}
			case *VariableExpression:
				if n != nil {
					Walk(v, n.Initializer)
				}
			case *VariableStatement:
				if n != nil {
					for _, e := range n.List {
						Walk(v, e)
					}
				}
			case *WhileStatement:
				if n != nil {
					Walk(v, n.Test)
					Walk(v, n.Body)
				}
			case *WithStatement:
				if n != nil {
					Walk(v, n.Object)
					Walk(v, n.Body)
				}*/
	default:
		//panic(fmt.Sprintf("Walk: unexpected node type %T", n))
	}
}
