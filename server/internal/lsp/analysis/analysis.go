package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"go/token"
)

// SymbolTable stores list of symbols defined in the project.
// Each symbol has an "address". This address allows to now where the symbol
/*
// Ideas:
// module.[global|scope(funName)].symbolName
// this would cover:
//  	module foo;
//      int symbol;
//      fn void foo(){ int symbol; }
// This does not cover:
//      module foo;
//      fn void foo(){
//     		int symbol; // #1
//			{
//				int symbol; // #2
//			}
//			{
//				char symbol; // #3
//				{
//					float symbol; //#4
//				}
//			}
//    	}
//
// Ideas:
// #1: foo.foo.symbol    <-- declared in root scope
// #2: foo.foo[1].symbol <-- First sub scope
// #3: foo.foo[2].symbol <-- Second sub scope
// #4: foo.foo[2][0].symbol <-- Second sub scope, First sub scope
//
// Idea 2. Not sure if this will allow to find them, but maybe this scope
// location would be useful to have to disambiguate, better storing it
// in a different column

 	symbol table
 	--------------
	id: primary key
	symbol_name: name of the symbol
	module: module name where this is defined
	path: full route to reach this symbol
	scope_path: Example [2][0]. Helps determining what is the scope this is defined.
*/
type SymbolTable struct {
}

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

func FindNodeAtPosition(n ast.Node, fset *token.FileSet, pos lsp.Position) ast.Node {
	if n == nil {
		return nil
	}

	// Convertimos la posición del nodo a coordenadas (línea y columna)
	start := n.StartPosition()
	end := n.EndPosition()

	// Verificamos si la posición está dentro del rango del nodo
	if (start.Line < pos.Line || (start.Line == pos.Line && start.Column <= pos.Column)) &&
		(end.Line > pos.Line || (end.Line == pos.Line && end.Column >= pos.Column)) {

	}
}
