package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
)

/*
type Location struct {
	Uri   protocol.URI
	Range protocol.Range
}*/

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

func FindSymbolAtPosition(pos lsp.Position, fileName string, symbolTable *SymbolTable, tree ast.Node, content string) option.Option[*Symbol] {
	nodeAtPosition, path := FindNode(tree, pos)

	if nodeAtPosition == nil {
		return option.None[*Symbol]()
	}

	//var identName string
	explicitIdentModule := option.None[string]()
	switch n := nodeAtPosition.(type) {
	case *ast.Ident:
		//identName = n.Name
		if n.ModulePath != nil {
			explicitIdentModule = option.Some(n.ModulePath.Name)
		}
	}

	scopeCtxt := getContextFromPosition(path, pos, content, ContextHintForGoTo)

	if scopeCtxt.isSelExpr {
		step := path[scopeCtxt.lowestSelExprIndex+1]
		// If cursor is at last part of a SelectorExpr, we need to solve the type of SelectorExpr.X
		if step.propertyName == "Sel" {
			// We need to solve first SelectorExpr.X!
			symbol, _ := solveSelAtSelectorExpr(path[scopeCtxt.lowestSelExprIndex].node.(*ast.SelectorExpr), pos, fileName, scopeCtxt, symbolTable, 0)

			if symbol != nil {
				return option.Some(symbol)
			} else {
				return option.None[*Symbol]()
			}
		} else {
			// As cursor is at X, we can just search normally.
		}
	}

	// -------------------------------------------------
	// Normal search
	from := NewLocation(fileName, pos, scopeCtxt.moduleName)
	sym := symbolTable.FindSymbolByPosition(scopeCtxt.fullIdentUnderCursor, explicitIdentModule, from)

	return sym
}

func isWrapperNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.ExpressionStmt, *ast.DeclarationStmt:
		return true
	default:
		return false // Ignore other nodes
	}
}
