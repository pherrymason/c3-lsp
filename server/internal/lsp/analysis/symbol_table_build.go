package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
)

func BuildSymbolTable(astTree ast.Node, fileName string) SymbolTable {
	visitor := newSymbolTableVisitor()
	walk.Walk(&visitor, astTree, "")

	return visitor.table
}

type symbolTableGenerator struct {
	table SymbolTable

	// State properties to keep track
	currentModule   ast.Module
	currentFilePath *ast.File
	currentScope    *Scope
	scopePushed     uint
}

func newSymbolTableVisitor() symbolTableGenerator {
	return symbolTableGenerator{
		table: *NewSymbolTable(),
	}
}

func (v *symbolTableGenerator) Enter(node ast.Node, propertyName string) walk.Visitor {
	switch n := node.(type) {
	case ast.File:
		v.currentFilePath = &n

	case ast.Module:
		v.currentModule = ast.Module{
			Name:              n.Name,
			GenericParameters: n.GenericParameters,
			NodeAttributes:    n.NodeAttributes,
		}
		v.currentScope = v.table.RegisterNewRootScope(v.currentFilePath.Name, n.GetRange())
		v.scopePushed++

	case *ast.GenDecl:
		if n.Token == ast.VAR || n.Token == ast.CONST {
			_, symbol := v.currentScope.RegisterSymbol(
				n.Spec.(*ast.ValueSpec).Names[0].Name,
				n.Range,
				n,
				v.currentModule,
				v.currentFilePath.Name,
				n.Token,
			)

			typeExpression := n.Spec.(*ast.ValueSpec).Type
			symbol.Type = TypeDefinition{
				Name:      typeExpression.Identifier.Name,
				IsBuiltIn: typeExpression.BuiltIn,
				NodeDecl:  typeExpression,
			}

		} else if n.Token == ast.ENUM {
			v.currentScope.RegisterSymbol(
				n.Spec.(*ast.TypeSpec).Name.Name,
				n.Range,
				n,
				v.currentModule,
				v.currentFilePath.Name,
				ast.ENUM,
			)
		}

	case *ast.StructDecl:
		v.currentScope.RegisterSymbol(n.Name, n.Range, n, v.currentModule, v.currentFilePath.Name, ast.STRUCT)

	case *ast.FaultDecl:
		v.currentScope.RegisterSymbol(n.Name.Name, n.Range, n, v.currentModule, v.currentFilePath.Name, ast.FAULT)

	case *ast.DefDecl:
		v.currentScope.RegisterSymbol(n.Name.Name, n.Range, n, v.currentModule, v.currentFilePath.Name, ast.DEF)

	case *ast.FunctionDecl:
		_, symbol := v.currentScope.RegisterSymbol(n.Signature.Name.Name, n.Range, n, v.currentModule, v.currentFilePath.Name, ast.FUNCTION)
		if n.Body != nil {
			v.currentScope = v.currentScope.pushScope(n.Body.GetRange())
			v.scopePushed++
		}

		if n.ParentTypeId.IsSome() {
			// Should register method as children of parent type
			parentSymbol := v.table.findSymbolInScope(
				n.ParentTypeId.Get().Name,
				v.currentScope,
			)
			if parentSymbol != nil {
				parentSymbol.AppendChild(symbol, Method)
			}
		}

	case ast.FunctionSignature:
		for _, param := range n.Parameters {
			_, sym := v.currentScope.RegisterSymbol(param.Name.Name,
				param.GetRange(),
				param,
				v.currentModule,
				v.currentFilePath.Name,
				ast.VAR,
			)

			sym.Type = TypeDefinition{
				Name:      param.Type.Identifier.Name,
				IsBuiltIn: param.Type.BuiltIn,
				NodeDecl:  param.Type,
			}
		}

	case *ast.CompoundStmt:

	case *ast.BlockExpr:
		v.currentScope = v.currentScope.pushScope(n.GetRange())
		v.scopePushed++
	}
	return v
}

func (v *symbolTableGenerator) Exit(n ast.Node, propertyName string) {
	switch n.(type) {
	case *ast.BlockExpr, *ast.FunctionDecl:
		if v.scopePushed > 0 {
			v.currentScope = v.currentScope.Parent.Get()
			v.scopePushed--
		}
	}
}