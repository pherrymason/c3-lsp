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

func FindSymbolAtPosition(pos lsp.Position, symbolTable SymbolTable, tree ast.Node) option.Option[*Symbol] {
	nodeAtPosition, path := FindNode(tree, pos)

	var name string
	switch n := nodeAtPosition.(type) {
	case *ast.Ident:
		name = n.Name
	}

	scopeStack := []lsp.Range{}

	moduleName := []string{}
	for _, n := range path {
		if moduleNode, ok := n.(ast.Module); ok {
			moduleName = append(moduleName, moduleNode.Name)
			scopeStack = append(scopeStack, moduleNode.GetRange())
		} else if fnDecl, ok := n.(*ast.FunctionDecl); ok {
			scopeStack = append(scopeStack, fnDecl.Body.GetRange())
		}
	}

	sym := symbolTable.FindSymbol(name, moduleName, scopeStack, 0)

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
