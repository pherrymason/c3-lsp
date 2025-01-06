package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
)

func BuildSymbolTable(astTree ast.Node) SymbolTable {
	visitor := newSymbolTableVisitor()
	walk.Walk(&visitor, astTree)

	return visitor.table
}

type symbolTableGenerator struct {
	table SymbolTable

	// State properties to keep track
	currentModule   ast.Module
	currentFilePath *ast.File
	currentScope    *Scope
}

func newSymbolTableVisitor() symbolTableGenerator {
	return symbolTableGenerator{
		table: *NewSymbolTable(),
	}
}

func (v *symbolTableGenerator) Enter(node ast.Node) walk.Visitor {
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

	case *ast.GenDecl:
		if n.Token == ast.VAR || n.Token == ast.CONST {
			_, symbol := v.table.RegisterSymbol(
				n.Spec.(*ast.ValueSpec).Names[0].Name,
				n.Range,
				n, v.currentModule,
				v.currentScope,
				v.currentFilePath.Name,
			)

			typeExpression := n.Spec.(*ast.ValueSpec).Type
			symbol.Type = TypeDefinition{
				Name:      typeExpression.Identifier.Name,
				IsBuiltIn: typeExpression.BuiltIn,
				NodeDecl:  typeExpression,
			}

		} else if n.Token == ast.ENUM {
			v.table.RegisterSymbol(
				n.Spec.(*ast.TypeSpec).Name.Name,
				n.Range,
				n,
				v.currentModule,
				v.currentScope,
				v.currentFilePath.Name,
			)
		}

	case *ast.StructDecl:
		v.table.RegisterSymbol(n.Name, n.Range, n, v.currentModule,
			v.currentScope,
			v.currentFilePath.Name)

	case *ast.FaultDecl:
		v.table.RegisterSymbol(n.Name.Name, n.Range, n, v.currentModule,
			v.currentScope,
			v.currentFilePath.Name)

	case *ast.DefDecl:
		v.table.RegisterSymbol(n.Name.Name, n.Range, n, v.currentModule,
			v.currentScope,
			v.currentFilePath.Name)

	case *ast.FunctionDecl:
		id, _ := v.table.RegisterSymbol(n.Signature.Name.Name, n.Range, n, v.currentModule,
			v.currentScope,
			v.currentFilePath.Name)
		v.currentScope = v.currentScope.pushScope(n.Body.GetRange())
		if n.ParentTypeId.IsSome() {
			// Should register method as children of parent type
			sym := v.table.FindSymbol(
				n.ParentTypeId.Get().Name,
				[]string{v.currentModule.Name},
				[]lsp.Range{}, // Do not limit search to any scope
				ast.Token(ast.STRUCT),
			)
			if sym.IsSome() {
				sym.Get().AppendChild(id, Method)
			}
		}

	case ast.FunctionSignature:
		for _, param := range n.Parameters {
			_, sym := v.table.RegisterSymbol(
				param.Name.Name,
				param.GetRange(),
				param,
				v.currentModule,
				v.currentScope,
				v.currentFilePath.Name,
			)
			sym.Type = TypeDefinition{
				Name:      param.Type.Identifier.Name,
				IsBuiltIn: param.Type.BuiltIn,
				NodeDecl:  param.Type,
			}
		}

	case *ast.CompoundStmt:
		v.currentScope = v.currentScope.pushScope(n.GetRange())
	}
	return v
}

func (v *symbolTableGenerator) Exit(n ast.Node) {
	switch n.(type) {
	case *ast.CompoundStmt, *ast.FunctionDecl:
		v.currentScope = v.currentScope.Parent
	}
}
