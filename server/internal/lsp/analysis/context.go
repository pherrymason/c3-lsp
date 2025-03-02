package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/analysis/symbol"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"strings"
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
	moduleName symbols.ModuleName

	identUnderCursor     string // single ident under cursor
	fullIdentUnderCursor string // full ident under cursor. Difference with `identUnderCursor`is that this one includes the whole chain of selectors

	// Information related to node being under an ast.SelectorExpr
	isSelExpr          bool
	selExpr            *ast.SelectorExpr
	lowestSelExprIndex int
	hasParseError      bool
}

func getASTNodeContext(path []PathStep) astContext {
	ctxt := astContext{
		pathStep:           path,
		isSelExpr:          false,
		lowestSelExprIndex: 0,
		moduleName:         symbols.NewModuleName(""),
	}

	totalSteps := len(path)
	parentNodeIsSelectorExpr := false
	selectorsChained := 0

	for i := totalSteps - 1; i >= 0; i-- {
		switch stepNode := path[i].node.(type) {
		case *ast.Module:
			ctxt.moduleName = symbols.NewModuleName(stepNode.Name)

		case *ast.Ident:
			ctxt.identUnderCursor = stepNode.Name
			ctxt.fullIdentUnderCursor = stepNode.Name

		case *ast.SelectorExpr:
			selectorsChained++
			if !parentNodeIsSelectorExpr {
				parentNodeIsSelectorExpr = true
				ctxt.isSelExpr = true
				ctxt.lowestSelExprIndex = i
				ctxt.selExpr = stepNode
			}

		case *ast.FunctionDecl:
			// Check if we are inside a struct/enum/fault method with `self` defined.
			for _, param := range stepNode.Signature.Parameters {
				if param.Name.Name == "self" {
					if stepNode.ParentTypeId.IsSome() {
						ident := stepNode.ParentTypeId.Get()
						ctxt.selfType = ident
					}
				}
			}

		case *ast.ErrorNode:
			ctxt.hasParseError = true

		default:
			//if parentNodeIsSelectorExpr {
			//	i = 0
			//}
		}
	}

	return ctxt
}

const (
	ContextHintForGoTo = iota
	ContextHintForCompletion
)

// getContextFromPosition returns the context of the AST node at the given position.
// If there are any parse errors, it will try to get the context from the content string. This will be weaker than the context from the AST.
func getContextFromPosition(path []PathStep, pos lsp.Position, content string, hint int) astContext {
	ctxt := getASTNodeContext(path)

	if ctxt.hasParseError {
		identUnderCursor, fullIdentUnderCursor, hasFieldAccess := findWordUnderCursor(pos, content, hint, ctxt)

		ctxt.identUnderCursor = identUnderCursor
		ctxt.fullIdentUnderCursor = fullIdentUnderCursor
		if hasFieldAccess {
			ctxt.isSelExpr = true
			ctxt.selExpr = parseSelectorExpression(ctxt.fullIdentUnderCursor)
		}
	}

	return ctxt
}

func findWordUnderCursor(pos lsp.Position, content string, hint int, ctxt astContext) (string, string, bool) {
	index := pos.IndexIn(content)
	identUnderCursor := ""
	fullIdentUnderCursor := ""

	startFullWord := index
	startIdentWord := index
	endIdentWord := index
	hasFieldAccess := false
	numberOfDots := 0

	// Rewind in content until a space or {()} is found
	for i := index - 1; i >= 0; i-- {
		r := rune(content[i])
		//log.Printf("%c ", r)

		if utils.IsIdentValidCharacter(r) || r == '.' {
			if r == '.' {
				hasFieldAccess = true
				numberOfDots++
			}

			if !hasFieldAccess {
				startIdentWord = i
			}
			startFullWord = i
		} else {
			break
		}
	}

	if hint == ContextHintForGoTo {
		// Forward to find end of word
		endFullWord := index
		stopCheckingIdentWord := false
		for i := index; i < len(content); i++ {
			r := rune(content[i])
			//log.Printf("%c ", r)

			if utils.IsIdentValidCharacter(r) {
				if !stopCheckingIdentWord {
					endIdentWord = i
				}
				endFullWord = i
				if r == '.' {
					stopCheckingIdentWord = true
					endFullWord = i
					hasFieldAccess = true
				}
			} else {
				break
			}
		}
		fullIdentUnderCursor = content[startFullWord:endFullWord]
		identUnderCursor = content[startIdentWord:endIdentWord]
	} else {
		fullIdentUnderCursor = content[startFullWord:index]
		identUnderCursor = content[startIdentWord:index]
	}

	//if numberOfDots <= 1 {
	//	hasFieldAccess = false
	//}

	return identUnderCursor, fullIdentUnderCursor, hasFieldAccess
}

// getAstContextFromString does some analysis from pure string.
// This is weaker than getASTNodeContext, but we are forced to do it because treesitter sometimes emits error nodes.
func getAstContextFromString(astCtxt astContext, content string) astContext {
	length := len(content)
	hasFieldAccess := false
	for i := length - 1; i >= 0; i-- {
		if rune(content[i]) == '.' {
			hasFieldAccess = true
			break
		}
	}

	if hasFieldAccess {
		astCtxt.isSelExpr = true
		// Build an ast.SelectorExpr from string
		// Rewind in content until a space or {()} is found
		startWordPosition := length
		for i := length - 1; i >= 0; i-- {
			r := rune(content[i])
			//log.Printf("%c ", r)
			if utils.IsIdentValidCharacter(r) || r == '.' {
				startWordPosition = i
			} else {
				break
			}
		}

		ident := content[startWordPosition:]
		astCtxt.selExpr = parseSelectorExpression(ident)
		//strings.Split(ident, ".")
		//log.Printf("Rewinded symbol: %s", ident)
	}

	return astCtxt
}

func parseSelectorExpression(input string) *ast.SelectorExpr {
	parts := strings.SplitN(input, "::", 2) // Divide by "::"
	var baseExpr ast.Expression
	var moduleIdent *ast.Ident

	if len(parts) == 2 {
		// Si hay "::", la parte antes de "::" es el módulo
		moduleIdent = &ast.Ident{Name: parts[0]}
		// La parte después de "::" contiene los fields
		parts = parts[1:]
	}

	fields := strings.Split(parts[0], ".")

	// Build nested ast.SelectorExpr
	for _, field := range fields {
		//if field == "" {
		//	continue
		//}

		if baseExpr == nil {
			baseExpr = &ast.Ident{Name: field}
			if moduleIdent != nil {
				baseExpr.(*ast.Ident).ModulePath = moduleIdent
			}
		} else {
			baseExpr = &ast.SelectorExpr{
				X:   baseExpr,
				Sel: &ast.Ident{Name: field},
			}
		}
	}

	_, ok := baseExpr.(*ast.Ident)
	if ok {
		baseExpr = &ast.SelectorExpr{
			X: baseExpr,
		}
	}

	return baseExpr.(*ast.SelectorExpr)
}
