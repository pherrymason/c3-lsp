package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
)

// PositionContext defines information about what's in current position
type PositionContext struct {
	Pos  lsp.Position
	kind uint // Element type the cursor is at

	IsLiteral          bool
	IsIdentifier       bool
	IsModuleIdentifier bool
	ImportStmt         ast.Node
}

// astContext
// Has context information about a specific AST node.
type astContext struct {
	selfType   *ast.Ident
	pathStep   []PathStep
	moduleName ModuleName

	// Information related to node being under an ast.SelectorExpr
	isSelExpr          bool
	lowestSelExprIndex int
}

func getASTNodeContext(path []PathStep) astContext {
	astCtxt := astContext{
		pathStep:           path,
		isSelExpr:          false,
		lowestSelExprIndex: 0,
		moduleName:         NewModuleName(""),
	}

	totalSteps := len(path)
	parentNodeIsSelectorExpr := false
	selectorsChained := 0

	for i := totalSteps - 1; i >= 0; i-- {
		switch stepNode := path[i].node.(type) {
		case *ast.Module:
			astCtxt.moduleName = NewModuleName(stepNode.Name)

		case *ast.Ident:

		case *ast.SelectorExpr:
			selectorsChained++
			if !parentNodeIsSelectorExpr {
				parentNodeIsSelectorExpr = true
				astCtxt.isSelExpr = true
				astCtxt.lowestSelExprIndex = i
			}

		case *ast.FunctionDecl:
			// Check if we are inside a struct/enum/fault method with `self` defined.
			for _, param := range stepNode.Signature.Parameters {
				if param.Name.Name == "self" {
					if stepNode.ParentTypeId.IsSome() {
						ident := stepNode.ParentTypeId.Get()
						astCtxt.selfType = ident
					}
				}
			}

		default:
			//if parentNodeIsSelectorExpr {
			//	i = 0
			//}
		}
	}

	return astCtxt
}
