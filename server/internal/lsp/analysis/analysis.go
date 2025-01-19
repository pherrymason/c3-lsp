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

type findContext struct {
	selfType           *ast.Ident
	pathStep           []PathStep
	lowestSelExprIndex int
}

func FindSymbolAtPosition(pos lsp.Position, fileName string, symbolTable *SymbolTable, tree ast.Node) option.Option[*Symbol] {
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
			break
			//	scopeStack = append(scopeStack, moduleNode.GetRange())
		} // else if fnDecl, ok := step.node.(*ast.FunctionDecl); ok {
		//	scopeStack = append(scopeStack, fnDecl.Body.GetRange())
		//}
	}
	// If parent is a SelectExpr, we will need to first search chain of elements to be able to find `name`.

	totalSteps := len(path)
	parentNodeIsSelectorExpr := false
	var parentSelectorExpr *ast.SelectorExpr

	selectorsChained := 0
	context := findContext{
		pathStep:           path,
		lowestSelExprIndex: 0,
	}
	for i := totalSteps - 1; i >= 0; i-- {
		switch stepNode := path[i].node.(type) {
		case *ast.Ident, ast.Ident:

		case *ast.SelectorExpr:
			selectorsChained++
			parentSelectorExpr = stepNode
			if !parentNodeIsSelectorExpr {
				parentNodeIsSelectorExpr = true
				context.lowestSelExprIndex = i
			}

		case *ast.FunctionDecl:
			// Check if we are inside a struct/enum/fault method with `self` defined.
			for _, param := range stepNode.Signature.Parameters {
				if param.Name.Name == "self" {
					if stepNode.ParentTypeId.IsSome() {
						ident := stepNode.ParentTypeId.Get()
						context.selfType = &ident
					}
				}
			}

		default:
			//if parentNodeIsSelectorExpr {
			//	i = 0
			//}
		}
	}

	if parentNodeIsSelectorExpr {
		step := path[context.lowestSelExprIndex+1]
		if step.propertyName == "Sel" {
			if selectorsChained > 1 {
				// Even if we are resolving final part of a SelectorExpr, we are in the middle of a bigger chain of SelectorExpr. This means
			}

			// We need to solve first SelectorExpr.X!
			symbol := solveSelAtSelectorExpr(
				path[context.lowestSelExprIndex].node.(*ast.SelectorExpr),
				pos,
				fileName,
				moduleName,
				context,
				symbolTable,
				0,
			)

			if symbol != nil {
				return option.Some(symbol)
			}
		} else {
			// As cursor is at X, we can just search normally.
		}
	}
	if parentNodeIsSelectorExpr {
		parentSelectorExpr.StartPosition()
		parentSelectorExpr = nil
	}
	// -------------------------------------------------
	// Normal search
	sym := symbolTable.FindSymbolByPosition(pos, fileName, name, moduleName, 0)

	return sym
}

// solveSelAtSelectorExpr resolves Sel Ident symbol.
func solveSelAtSelectorExpr(
	selectorExpr *ast.SelectorExpr,
	pos lsp.Position,
	fileName string,
	moduleName ModuleName,
	context findContext,
	symbolTable *SymbolTable,
	deepLevel uint) *Symbol {
	// To be able to resolve selectorExpr.Sel, we need to know first what is selectorExpr.X is or what does it return.
	var parentSymbol *Symbol
	switch base := selectorExpr.X.(type) {
	case *ast.Ident:
		// X is a plain Ident. We need to resolve Ident Type:
		// - Ident might be a variable. What's its type? Struct/Enum/Fault?
		// - Ident might be `self`.
		parentSymbolName := base.Name
		if parentSymbolName == "self" {
			// We need to go to parent FunctionDecl and see if `self` is a defined argument
			if context.selfType != nil {
				parentSymbolName = context.selfType.Name
				result := symbolTable.FindSymbolByPosition(
					context.selfType.StartPosition(),
					fileName,
					context.selfType.Name,
					moduleName,
					0,
				)
				parentSymbol = result.GetOrElse(nil)
			} else {
				// !!!!! we've found a self, but function is not flagged as method! Confusion triggered!!!
			}
		} else {
			parentSymbol = symbolTable.SolveType(base.Name, pos, fileName)
		}

		if parentSymbol == nil {
			return nil
		}

	case *ast.SelectorExpr:
		// X is a SelectorExpr itself, we need to solve the type of base.Sel
		parentSymbol = solveSelAtSelectorExpr(base, pos, fileName, moduleName, context, symbolTable, deepLevel+1)
		if parentSymbol == nil {
			return nil
		}

	case *ast.FunctionCall:
		ident := base.Identifier
		switch i := ident.(type) {
		case *ast.SelectorExpr:
			parentSymbol = solveSelAtSelectorExpr(i, pos, fileName, moduleName, context, symbolTable, deepLevel+1)
			if parentSymbol == nil {
				return nil
			}
		case *ast.Ident:
			sym := symbolTable.FindSymbolByPosition(pos, fileName, i.Name, moduleName, 0)
			if sym.IsNone() {
				return nil
			}
			parentSymbol = sym.Get()
		}

	default:
		return nil
	}

	// We've found X type, we are ready to find selectorExpr.Sel inside `X`'s type:
	solveElementType := true
	if deepLevel == 0 {
		solveElementType = false
	}
	return resolveChildSymbol(parentSymbol, selectorExpr.Sel.Name, moduleName, fileName, symbolTable, solveElementType)
}

func resolveChildSymbol(symbol *Symbol, nextIdent string, moduleName ModuleName, fileName string, symbolTable *SymbolTable, solveType bool) *Symbol {
	if symbol == nil {
		return nil
	}

	switch symbol.Kind {
	case ast.ENUM, ast.FAULT:
		for _, childRel := range symbol.Children {
			if childRel.Tag == Field && childRel.Child.Name == nextIdent {
				return childRel.Child
			} else if childRel.Tag == Method && childRel.Child.Name == nextIdent {
				return childRel.Child
			}
		}

	case ast.STRUCT:
		// Search In Members
		for _, member := range symbol.NodeDecl.(*ast.StructDecl).Members {
			if member.Names[0].Name == nextIdent {
				if member.Type.BuiltIn || !solveType {
					return &Symbol{
						Name:     member.Names[0].Name,
						Module:   moduleName,
						URI:      fileName,
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

				// If nextIdent is the last element in the chain of SelectorExpr, we don't need to resolve the type.
				// Else, we need to check for the type to continue resolving each step of the chain
				value := symbolTable.FindSymbolByPosition(
					member.Range.Start,
					fileName,
					member.Type.Identifier.Name,
					moduleName,
					0,
				)
				if value.IsSome() {
					return value.Get()
				} else {
					return nil
				}
			}
		}

		// Not found in members, we need to search struct methods
		for _, relatedSymbol := range symbol.Children {
			if relatedSymbol.Tag == Method && relatedSymbol.Child.Name == nextIdent {
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
			return resolveChildSymbol(
				returnTypeSymbol.Get(),
				nextIdent,
				moduleName,
				fileName,
				symbolTable,
				solveType,
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
