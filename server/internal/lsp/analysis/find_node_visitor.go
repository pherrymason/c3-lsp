package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
)

type FindNodeVisitor struct {
	pos   lsp.Position
	found ast.Node
	Path  []ast.Node
}

// Visit implementa el método del visitor.
func (v *FindNodeVisitor) Enter(node ast.Node) walk.Visitor {
	if node == nil {
		return nil
	}

	// Verify if the position is inside the range of the node
	if node.GetRange().HasPosition(v.pos) {
		//if v.pos >= node.StartPosition() && v.pos < node.EndPosition() {
		// Guardar el nodo actual si es más específico
		v.found = node
		v.Path = append(v.Path, node)

		// Continuar recorriendo los nodos hijos
		return v
	}

	return nil
}

func (v *FindNodeVisitor) Exit(n ast.Node) {
	if len(v.Path) > 0 && v.Path[len(v.Path)-1].GetId() == n.GetId() {
		v.Path = v.Path[:len(v.Path)-1]
	}
}

// FindNode encuentra el nodo del AST que contiene la posición dada.
func FindNode(root ast.Node, pos lsp.Position) ast.Node {
	visitor := &FindNodeVisitor{pos: pos}
	walk.Walk(visitor, root)
	return visitor.found
}
