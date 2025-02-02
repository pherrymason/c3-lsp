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
	moduleName         ModuleName
}

func FindSymbolAtPosition(pos lsp.Position, fileName string, symbolTable *SymbolTable, tree ast.Node) option.Option[*Symbol] {
	nodeAtPosition, path := FindNode(tree, pos)

	if nodeAtPosition == nil {
		return option.None[*Symbol]()
	}

	var identName string
	explicitIdentModule := option.None[string]()
	switch n := nodeAtPosition.(type) {
	case *ast.Ident:
		identName = n.Name
		if n.ModulePath != nil {
			explicitIdentModule = option.Some(n.ModulePath.Name)
		}
	}

	// Analyze parent nodes to better understand context
	// -------------------------------------------------

	totalSteps := len(path)
	parentNodeIsSelectorExpr := false
	var parentSelectorExpr *ast.SelectorExpr

	// --------------------------------------
	// Get context info
	selectorsChained := 0
	scopeCtxt := findContext{
		pathStep:           path,
		lowestSelExprIndex: 0,
		moduleName:         NewModuleName(""),
	}
	for i := totalSteps - 1; i >= 0; i-- {
		switch stepNode := path[i].node.(type) {
		case *ast.Module:
			scopeCtxt.moduleName = NewModuleName(stepNode.Name)

		case *ast.Ident:

		case *ast.SelectorExpr:
			selectorsChained++
			parentSelectorExpr = stepNode
			if !parentNodeIsSelectorExpr {
				parentNodeIsSelectorExpr = true
				scopeCtxt.lowestSelExprIndex = i
			}

		case *ast.FunctionDecl:
			// Check if we are inside a struct/enum/fault method with `self` defined.
			for _, param := range stepNode.Signature.Parameters {
				if param.Name.Name == "self" {
					if stepNode.ParentTypeId.IsSome() {
						ident := stepNode.ParentTypeId.Get()
						scopeCtxt.selfType = ident
					}
				}
			}

		default:
			//if parentNodeIsSelectorExpr {
			//	i = 0
			//}
		}
	}
	// End of getting context info
	// --------------------------------------

	if parentNodeIsSelectorExpr {
		step := path[scopeCtxt.lowestSelExprIndex+1]
		if step.propertyName == "Sel" {
			if selectorsChained > 1 {
				// Even if we are resolving final part of a SelectorExpr, we are in the middle of a bigger chain of SelectorExpr. This means
			}

			// We need to solve first SelectorExpr.X!
			symbol := solveSelAtSelectorExpr(path[scopeCtxt.lowestSelExprIndex].node.(*ast.SelectorExpr), pos, fileName, scopeCtxt, symbolTable, 0)

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
	from := fromPosition{position: pos, fileName: fileName, module: scopeCtxt.moduleName}
	sym := symbolTable.FindSymbolByPosition(identName, explicitIdentModule, from)

	return sym
}

// solveSelAtSelectorExpr resolves Sel Ident symbol.
func solveSelAtSelectorExpr(selectorExpr *ast.SelectorExpr, pos lsp.Position, fileName string, context findContext, symbolTable *SymbolTable, deepLevel uint) *Symbol {
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
				from := fromPosition{position: context.selfType.StartPosition(), fileName: fileName, module: context.moduleName}
				explicitModule := option.None[string]()
				if context.selfType.ModulePath != nil {
					explicitModule = option.Some(context.selfType.ModulePath.Name)
				}

				result := symbolTable.FindSymbolByPosition(
					context.selfType.Name,
					explicitModule,
					from,
				)
				parentSymbol = result.GetOrElse(nil)
			} else {
				// !!!!! we've found a self, but function is not flagged as method! Confusion triggered!!!
			}
		} else {
			parentSymbol = symbolTable.SolveType(base.Name, pos, fileName, context.moduleName)
		}

		if parentSymbol == nil {
			return nil
		}

	case *ast.TypeInfo:
		from := fromPosition{position: base.Identifier.StartPosition(), fileName: fileName, module: context.moduleName}
		explicitModule := option.None[string]()
		if base.Identifier.ModulePath != nil {
			explicitModule = option.Some(base.Identifier.ModulePath.Name)
		}
		result := symbolTable.FindSymbolByPosition(
			base.Identifier.Name,
			explicitModule,
			from,
		)
		parentSymbol = result.GetOrElse(nil)

	case *ast.SelectorExpr:
		// X is a SelectorExpr itself, we need to solve the type of base.Sel
		parentSymbol = solveSelAtSelectorExpr(base, pos, fileName, context, symbolTable, deepLevel+1)
		if parentSymbol == nil {
			return nil
		}

	case *ast.CallExpr:
		ident := base.Identifier
		switch i := ident.(type) {
		case *ast.SelectorExpr:
			parentSymbol = solveSelAtSelectorExpr(i, pos, fileName, context, symbolTable, deepLevel+1)
			if parentSymbol == nil {
				return nil
			}
		case *ast.Ident:
			from := fromPosition{position: pos, fileName: fileName, module: context.moduleName}
			explicitModule := option.None[string]()
			if i.ModulePath != nil {
				explicitModule = option.Some(i.ModulePath.Name)
			}
			sym := symbolTable.FindSymbolByPosition(i.Name, explicitModule, from)
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

	return resolveChildSymbol(parentSymbol, selectorExpr.Sel.Name, context.moduleName, fileName, symbolTable, solveElementType)
}

func resolveChildSymbol(parentSymbol *Symbol,
	nextIdent string, moduleName ModuleName, fileName string, symbolTable *SymbolTable, solveType bool) *Symbol {
	if parentSymbol == nil {
		return nil
	}

	switch parentSymbol.Kind {
	case ast.ENUM, ast.FAULT:
		for _, childRel := range parentSymbol.Children {
			if childRel.Tag == Field && childRel.Child.Name == nextIdent {
				return childRel.Child
			} else if childRel.Tag == Method && childRel.Child.Name == nextIdent {
				return childRel.Child
			}
		}

	case ast.STRUCT:
		// Search In Members
		// There are two cases:
		// - NodeDecl == *ast.GenDecl
		// - NodeDecl == *ast.StructType // <-- this case is when traversing an anonymous sub struct
		var specType *ast.StructType
		if genDecl, ok := parentSymbol.NodeDecl.(*ast.GenDecl); ok {
			specType = genDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType)
		} else {
			panic("????")
		}

		return resolveChildSymbolInStructFields(
			nextIdent,
			specType,
			parentSymbol,
			parentSymbol.Module,
			fileName,
			symbolTable,
			solveType,
		)
	case ast.AnonymousStructField:
		field, _ := parentSymbol.NodeDecl.(*ast.StructField)
		specType := field.Type.(*ast.StructType)
		return resolveChildSymbolInStructFields(
			nextIdent,
			specType,
			parentSymbol,
			moduleName,
			fileName,
			symbolTable,
			solveType,
		)

	case ast.FUNCTION:
		fn := parentSymbol.NodeDecl.(*ast.FunctionDecl)
		returnType := fn.Signature.ReturnType
		from := fromPosition{position: returnType.Range.Start, fileName: fileName, module: parentSymbol.Module}
		explicitModule := option.None[string]()
		if returnType.Identifier.ModulePath != nil {
			explicitModule = option.Some(returnType.Identifier.ModulePath.Name)
			moduleName = NewModuleName(returnType.Identifier.ModulePath.Name)
		}
		returnTypeSymbol := symbolTable.FindSymbolByPosition(
			returnType.Identifier.Name,
			explicitModule,
			from,
		)

		if returnTypeSymbol.IsSome() {
			return resolveChildSymbol(
				returnTypeSymbol.Get(),
				nextIdent,
				returnTypeSymbol.Get().Module,
				fileName,
				symbolTable,
				solveType,
			)
		}
	}

	return nil
}

func resolveChildSymbolInStructFields(searchIdent string, structType *ast.StructType, parentSymbol *Symbol, moduleName ModuleName, fileName string, symbolTable *SymbolTable, solveType bool) *Symbol {

	inlinedCandidates := []*ast.Ident{}

	for _, member := range structType.Fields {
		if member.Names[0].Name == searchIdent {
			switch t := member.Type.(type) {
			case *ast.TypeInfo:
				if t.BuiltIn || !solveType {
					typeName := ""
					if t.Identifier != nil {
						// It could be an anonymous struct, protect against that
						typeName = t.Identifier.String()
					}

					return &Symbol{
						Name:     member.Names[0].Name,
						Module:   moduleName,
						URI:      fileName,
						Range:    member.Range,
						NodeDecl: member,
						Kind:     ast.FIELD,
						Type: TypeDefinition{
							typeName,
							t.BuiltIn,
							t,
						},
					}
				}
				from := fromPosition{position: member.Range.Start, fileName: fileName, module: moduleName}
				explicitModule := option.None[string]()
				if t.Identifier.ModulePath != nil {
					explicitModule = option.Some(t.Identifier.ModulePath.Name)
				}
				value := symbolTable.FindSymbolByPosition(
					t.Identifier.Name,
					explicitModule,
					from,
				)
				if value.IsSome() {
					return value.Get()
				} else {
					return nil
				}

			case *ast.StructType:
				return &Symbol{
					Name:     member.Names[0].Name,
					Module:   moduleName,
					URI:      fileName,
					Range:    member.Range,
					NodeDecl: member,
					Kind:     ast.AnonymousStructField,
					Type: TypeDefinition{
						Name:      "",
						IsBuiltIn: false,
						NodeDecl:  member,
					},
				}
			}

			// If nextIdent is the last element in the chain of SelectorExpr, we don't need to resolve the type.
			// Else, we need to check for the type to continue resolving each step of the chain

		} else if member.Inlined {
			inlinedCandidates = append(inlinedCandidates, member.Type.(*ast.TypeInfo).Identifier)
		}
	}

	// Not found in members, we need to search struct methods
	for _, relatedSymbol := range parentSymbol.Children {
		if relatedSymbol.Tag == Method && relatedSymbol.Child.Name == searchIdent {
			return relatedSymbol.Child
		}
	}

	// Not found, look inside each inlinedCandidates, maybe is a subproperty of them
	for _, inlinedTypeIdent := range inlinedCandidates {
		from := fromPosition{position: inlinedTypeIdent.Range.Start, fileName: fileName, module: moduleName}
		explicitModule := option.None[string]()
		if inlinedTypeIdent.ModulePath != nil {
			explicitModule = option.Some(inlinedTypeIdent.ModulePath.Name)
			moduleName = NewModuleName(inlinedTypeIdent.ModulePath.Name)
		}

		inlinedTypeSymbol := symbolTable.FindSymbolByPosition(
			inlinedTypeIdent.Name,
			explicitModule,
			from,
		)
		if inlinedTypeSymbol.IsSome() {
			inlinedStructSymbol := inlinedTypeSymbol.Get()
			child := resolveChildSymbol(
				inlinedStructSymbol,
				searchIdent,
				moduleName,
				fileName,
				symbolTable,
				solveType,
			)
			if child != nil && child.Name == searchIdent {
				return child
			}
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
