package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
)

func BuildSymbolTable(astTree ast.Node) SymbolTable {
	visitor := symbolTableGenerator{}
	walk.Walk(&visitor, astTree)

	return visitor.table
}

type symbolTableGenerator struct {
	table SymbolTable

	// State properties to keep track
	currentModule   ast.Module
	currentFilePath *ast.File
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

	case *ast.GenDecl:
		if n.Token == ast.VAR || n.Token == ast.CONST {
			_, symbol := v.table.RegisterSymbol(
				n.Spec.(*ast.ValueSpec).Names[0].Name,
				n.Range,
				n, v.currentModule,
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
				v.currentFilePath.Name,
			)
		}

	case *ast.StructDecl:
		v.table.RegisterSymbol(n.Name, n.Range, n, v.currentModule,
			v.currentFilePath.Name)

	case *ast.FaultDecl:
		v.table.RegisterSymbol(n.Name.Name, n.Range, n, v.currentModule,
			v.currentFilePath.Name)

	case *ast.DefDecl:
		v.table.RegisterSymbol(n.Name.Name, n.Range, n, v.currentModule,
			v.currentFilePath.Name)

	case *ast.FunctionDecl:
		id, _ := v.table.RegisterSymbol(n.Signature.Name.Name, n.Range, n, v.currentModule,
			v.currentFilePath.Name)
		if n.ParentTypeId.IsSome() {
			// Should register method as children of parent type
			sym := v.table.FindSymbol(
				n.ParentTypeId.Get().Name,
				[]string{v.currentModule.Name},
				ast.Token(ast.STRUCT),
			)
			if sym.IsSome() {
				sym.Get().AppendChild(id, Method) // 1 is methods
			}
		}
	}
	return v
}

func (v *symbolTableGenerator) Exit(n ast.Node) {

}
