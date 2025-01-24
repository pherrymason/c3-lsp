package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
)

type PathStep struct {
	node         ast.Node
	propertyName string
}

type FindNodeVisitor struct {
	pos        lsp.Position
	found      ast.Node
	Path       []PathStep
	stopSearch bool
}

// Visit implementa el método del visitor.
func (v *FindNodeVisitor) Enter(node ast.Node, propertyName string) walk.Visitor {
	if node == nil {
		return nil
	}

	// Verify if the position is inside the range of the node
	if node.GetRange().HasPosition(v.pos) {
		// Guardar el nodo actual si es más específico
		v.found = node
		v.Path = append(v.Path, PathStep{node: node, propertyName: propertyName})

		// Continuar recorriendo los nodos hijos
		switch node.(type) {
		case *ast.Ident, ast.Ident, *ast.BasicLit:
			v.stopSearch = true
		default:
		}

		return v
	}

	return nil
}

func (v *FindNodeVisitor) Exit(n ast.Node, propertyName string) {
	if v.stopSearch {
		return
	}

	if len(v.Path) > 0 && v.Path[len(v.Path)-1].node.GetId() == n.GetId() {
		v.Path = v.Path[:len(v.Path)-1]
	}
}

// FindNode encuentra el nodo del AST que contiene la posición dada.
func FindNode(root ast.Node, pos lsp.Position) (ast.Node, []PathStep) {
	if root == nil {
		panic("FindNode with nil!")
	}

	visitor := &FindNodeVisitor{pos: pos}
	walk.Walk(visitor, root, "")

	if len(visitor.Path) == 0 {
		// When Path is empty, it means node was not found
		return nil, visitor.Path
	}

	return visitor.found, visitor.Path
}
