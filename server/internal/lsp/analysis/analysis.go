package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Location struct {
	Uri   protocol.URI
	Range protocol.Range
}

func PositionInNode(node ast.Node, pos lsp.Position) bool {
	char := pos.Column
	line := pos.Line

	return node != nil &&
		node.StartPosition().Column <= char &&
		node.StartPosition().Line <= line &&
		node.EndPosition().Column >= char &&
		node.EndPosition().Line >= line
}

func getPositionContext(document *document.Document, pos lsp.Position) PositionContext {
	posContext := PositionContext{
		Pos: pos,
	}

	for _, mod := range document.Ast.Modules {
		for _, include := range mod.Imports {
			if PositionInNode(include, pos) {
				posContext.ImportStmt = include
			}
		}
	}

	return posContext
}

func FindSymbolAtPosition(pos lsp.Position, fileName string, symbolTable SymbolTable, tree ast.Node) option.Option[*Symbol] {
	nodeAtPosition, path := FindNode(tree, pos)

	var name string
	switch n := nodeAtPosition.(type) {
	case *ast.Ident:
		name = n.Name
	case ast.Ident:
		name = n.Name
	}

	//scopeStack := []lsp.Range{}

	// Analyze parent nodes to better understand context
	// -------------------------------------------------
	var moduleName ModuleName
	for _, step := range path {
		if moduleNode, ok := step.node.(ast.Module); ok {
			moduleName = ModuleName(moduleNode.Name)
			//	scopeStack = append(scopeStack, moduleNode.GetRange())
		} // else if fnDecl, ok := step.node.(*ast.FunctionDecl); ok {
		//	scopeStack = append(scopeStack, fnDecl.Body.GetRange())
		//}
	}
	// If parent is a SelectExpr, we will need to first search chain of elements to be able to find `name`.

	totalSteps := len(path)
	selectExpr := false
	//identFound := false
	index := 0
	for i := totalSteps - 1; i >= 0; i-- {
		switch path[i].node.(type) {
		case *ast.Ident, ast.Ident:
			//identFound = true

		case *ast.SelectorExpr:
			selectExpr = true
			index = i
			i = 0
		}
	}

	if selectExpr {
		if path[index+1].propertyName == "Sel" {
			// We need to solve first SelectorExpr.X!
			symbol := solveSelAtSelectorExpr(
				path[index].node.(*ast.SelectorExpr),
				pos,
				fileName,
				moduleName,
				symbolTable,
			)

			if symbol != nil {
				return option.Some(symbol)
			}
		} else {
			// As cursor is at X, we can just search normally.
		}
	}

	// -------------------------------------------------

	sym := symbolTable.FindSymbolByPosition(pos, fileName, name, moduleName, 0)

	return sym
}

// solveSelAtSelectorExpr solves iteratively the X part of SelectorExpr
// Solves X. If X is itself a SelectorExpr, it will follow the chain and solve the symbol just before the last '.'
func solveSelAtSelectorExpr(selectorExpr *ast.SelectorExpr, pos lsp.Position, fileName string, moduleName ModuleName, symbolTable SymbolTable) *Symbol {
	var parentSymbol *Symbol
	switch base := selectorExpr.X.(type) {
	case *ast.Ident:
		parentSymbol = symbolTable.SolveType(base.Name, pos, fileName)
		if parentSymbol == nil {
			return nil
		}

	case *ast.SelectorExpr:
		parentSymbol = solveSelAtSelectorExpr(base, pos, fileName, moduleName, symbolTable)
		if parentSymbol == nil {
			return nil
		}

	case *ast.FunctionCall:
		ident := base.Identifier
		switch i := ident.(type) {
		case *ast.SelectorExpr:
			parentSymbol = solveSelAtSelectorExpr(i, pos, fileName, moduleName, symbolTable)
			if parentSymbol == nil {
				return nil
			}
		}

	default:
		return nil
	}

	return solveSymbolChild(parentSymbol, selectorExpr.Sel.Name, moduleName, fileName, &symbolTable)
}

func solveSymbolChild(symbol *Symbol, childName string, moduleName ModuleName, fileName string, symbolTable *SymbolTable) *Symbol {
	if symbol == nil {
		return nil
	}
	
	selIdent := childName
	switch symbol.Kind {
	case ast.STRUCT:
		// Search In Members
		for _, member := range symbol.NodeDecl.(*ast.StructDecl).Members {
			if member.Names[0].Name == selIdent {
				if member.Type.BuiltIn {
					return &Symbol{
						Name:     member.Names[0].Name,
						Module:   moduleName,
						FilePath: fileName,
						Range:    member.Range,
						NodeDecl: member,
						Kind:     ast.FIELD,
						Type: TypeDefinition{
							member.Type.Identifier.Name,
							member.Type.BuiltIn,
							member.Type,
						},
					}
				}

				value := symbolTable.FindSymbolByPosition(
					member.Range.Start,
					fileName,
					member.Type.Identifier.Name,
					moduleName,
					0,
				)
				return value.Get()
			}
		}

		// Not found in members, we need to search struct methods
		for _, relatedSymbol := range symbol.Children {
			if relatedSymbol.Tag == Method && relatedSymbol.Child.Name == selIdent {
				return relatedSymbol.Child
			}
		}

	case ast.FUNCTION:
		fn := symbol.NodeDecl.(*ast.FunctionDecl)
		returnType := fn.Signature.ReturnType
		returnTypeSymbol := symbolTable.FindSymbolByPosition(
			returnType.Range.Start,
			fileName,
			returnType.Identifier.Name,
			moduleName,
			0,
		)

		if returnTypeSymbol.IsSome() {
			return solveSymbolChild(
				returnTypeSymbol.Get(),
				selIdent,
				moduleName,
				fileName,
				symbolTable,
			)
		}
	}

	return nil
}

func isWrapperNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.ExpressionStmt, *ast.DeclarationStmt:
		return true
	default:
		return false // Ignore other nodes
	}
}
