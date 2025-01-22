package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
)

func BuildSymbolTable(astTree ast.Node, fileName string) *SymbolTable {
	visitor := newSymbolTableVisitor(nil)
	walk.Walk(&visitor, astTree, "")

	return visitor.table
}
func UpdateSymbolTable(symbolTable *SymbolTable, astTree ast.Node, fileName string) {
	visitor := newSymbolTableVisitor(symbolTable)
	walk.Walk(&visitor, astTree, "")
}

type symbolTableGenerator struct {
	table *SymbolTable

	// State properties to keep track
	currentModule   ast.Module
	currentFilePath *ast.File
	currentScope    *Scope
	scopePushed     uint
}

func newSymbolTableVisitor(symbolTable *SymbolTable) symbolTableGenerator {
	if symbolTable == nil {
		return symbolTableGenerator{
			table: NewSymbolTable(),
		}
	}

	return symbolTableGenerator{
		table: symbolTable,
	}
}

func (v *symbolTableGenerator) Enter(node ast.Node, propertyName string) walk.Visitor {
	switch n := node.(type) {
	case *ast.File:
		v.currentFilePath = n

	case ast.Module:
		v.currentModule = ast.Module{
			Name:              n.Name,
			GenericParameters: n.GenericParameters,
			NodeAttributes:    n.NodeAttributes,
		}
		v.currentScope = v.table.RegisterNewRootScope(v.currentFilePath.URI, n)
		v.scopePushed++

	case *ast.GenDecl:
		if n.Token == ast.VAR || n.Token == ast.CONST {
			_, symbol := v.currentScope.RegisterSymbol(
				n.Spec.(*ast.ValueSpec).Names[0].Name,
				n.Range,
				n,
				v.currentModule,
				v.currentFilePath.URI,
				n.Token,
			)

			typeExpression := n.Spec.(*ast.ValueSpec).Type
			typeName := typeExpression.Identifier.String()
			symbol.Type = TypeDefinition{
				Name:      typeName, // TODO does this having module path break anything?
				IsBuiltIn: typeExpression.BuiltIn,
				NodeDecl:  typeExpression,
			}

		} else if n.Token == ast.ENUM {
			_, enumSym := v.currentScope.RegisterSymbol(
				n.Spec.(*ast.TypeSpec).Name.Name,
				n.Range,
				n,
				v.currentModule,
				v.currentFilePath.URI,
				ast.ENUM,
			)

			enumType := n.Spec.(*ast.TypeSpec).TypeDescription.(*ast.EnumType)
			for _, value := range enumType.Values {
				_, enumFieldSym := v.currentScope.RegisterSymbol(
					value.Name.Name,
					value.Range,
					value,
					v.currentModule,
					v.currentFilePath.URI,
					ast.FIELD,
				)
				enumSym.AppendChild(enumFieldSym, Field)
			}
		}

	case *ast.StructDecl:
		v.currentScope.RegisterSymbol(n.Name, n.Range, n, v.currentModule, v.currentFilePath.URI, ast.STRUCT)

	case *ast.InterfaceDecl:
		v.currentScope.RegisterSymbol(n.Name.Name, n.Range, n, v.currentModule, v.currentFilePath.URI, ast.INTERFACE)

	case *ast.FaultDecl:
		_, fault := v.currentScope.RegisterSymbol(n.Name.Name, n.Range, n, v.currentModule, v.currentFilePath.URI, ast.FAULT)

		for _, f := range n.Members {
			_, m := v.currentScope.RegisterSymbol(
				f.Name.Name,
				f.GetRange(),
				f,
				v.currentModule,
				v.currentFilePath.URI,
				ast.FIELD,
			)
			fault.AppendChild(m, Field)
		}

	case *ast.DefDecl:
		v.currentScope.RegisterSymbol(n.Name.Name, n.Range, n, v.currentModule, v.currentFilePath.URI, ast.DEF)

	case *ast.FunctionDecl:
		_, symbol := v.currentScope.RegisterSymbol(n.Signature.Name.Name, n.Range, n, v.currentModule, v.currentFilePath.URI, ast.FUNCTION)
		if n.Body != nil {
			v.pushScope(n)
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
				v.currentFilePath.URI,
				ast.VAR,
			)

			typeName := param.Type.Identifier.String()
			sym.Type = TypeDefinition{
				Name:      typeName, // TODO does this having module path break anything?
				IsBuiltIn: param.Type.BuiltIn,
				NodeDecl:  param.Type,
			}
		}

	case *ast.CompoundStmt:

	case *ast.BlockExpr:
		v.pushScope(n)
	}
	return v
}

func (v *symbolTableGenerator) pushScope(n ast.Node) {
	v.currentScope = v.currentScope.pushScope(n.GetRange())
	v.scopePushed++
}

func (v *symbolTableGenerator) Exit(node ast.Node, propertyName string) {
	switch n := node.(type) {
	case *ast.BlockExpr:
		v.popScope()

	case *ast.FunctionDecl:
		if n.Body != nil {
			v.popScope()
		}
	}
}

func (v *symbolTableGenerator) popScope() {
	if v.scopePushed > 0 {
		v.currentScope = v.currentScope.Parent.Get() // Retrocede al scope padre
		v.scopePushed--                              // Actualiza el contador
	}
}
