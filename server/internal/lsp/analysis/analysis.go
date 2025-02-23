package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"log"
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

	//scopeCtxt := getASTNodeContext(path)
	scopeCtxt := getContextFromPosition(path, pos, content, ContextHintForGoTo)

	if scopeCtxt.isSelExpr {
		step := path[scopeCtxt.lowestSelExprIndex+1]
		//log.Print("Prop:", step.propertyName)
		// If cursor is at last part of a SelectorExpr, we need to solve the type of SelectorExpr.X
		if step.propertyName == "Sel" {
			//if selectorsChained > 1 {
			// Even if we are resolving final part of a SelectorExpr, we are in the middle of a bigger chain of SelectorExpr. This means
			//}

			// We need to solve first SelectorExpr.X!
			symbol, _ := solveSelAtSelectorExpr(path[scopeCtxt.lowestSelExprIndex].node.(*ast.SelectorExpr), pos, fileName, scopeCtxt, symbolTable, 0)

			if symbol != nil {
				return option.Some(symbol)
			}
		} else {
			// As cursor is at X, we can just search normally.
		}
	}
	/*
		if parentNodeIsSelectorExpr {
			parentSelectorExpr = nil
		}*/
	// -------------------------------------------------
	// Normal search
	from := NewLocation(fileName, pos, scopeCtxt.moduleName)
	sym := symbolTable.FindSymbolByPosition(scopeCtxt.fullIdentUnderCursor, explicitIdentModule, from)

	return sym
}

// solveSelAtSelectorExpr resolves Sel Ident symbol.
func solveSelAtSelectorExpr(selectorExpr *ast.SelectorExpr, pos lsp.Position, fileName string, context astContext, symbolTable *SymbolTable, deepLevel uint) (*Symbol, []*Symbol) {
	// To be able to resolve selectorExpr.Sel, we need to know first what is selectorExpr.X is or what does it return.
	var xSymbol *Symbol
	chainedSymbols := []*Symbol{}
	isInstance := false
	isType := false

	// Get definition of selectorExpr.X
	xSymbol, chainedSymbols, isType = solveXAtSelectorExpr(selectorExpr, pos, fileName, context, symbolTable, deepLevel)
	log.Printf("%t", isType)
	// Do we need to check its type?
	if xSymbol.Kind == ast.VAR {
		location := Location{FileName: fileName, Position: pos, Module: context.moduleName}
		xSymbol = symbolTable.SolveType(xSymbol.Identifier, option.None[string](), location)
		isInstance = true
	} else if xSymbol.Kind == ast.FIELD {
		explicitModule := option.None[string]()
		typeName := ""
		switch typeNode := xSymbol.TypeDef.NodeDecl.(type) {
		case *ast.TypeInfo:
			if typeNode.Identifier.ModulePath != nil {
				explicitModule = option.Some(typeNode.Identifier.ModulePath.String())
			}
			typeName = typeNode.Identifier.Name
		}

		location := Location{FileName: fileName, Position: pos, Module: context.moduleName}
		optRes := symbolTable.FindSymbolByPosition(typeName, explicitModule, location)
		xSymbol = optRes.GetOrElse(nil)
		isInstance = true
	}

	/*
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
					from := NewLocation(fileName, context.selfType.StartPosition(), context.moduleName)
					explicitModule := option.None[string]()
					if context.selfType.ModulePath != nil {
						explicitModule = option.Some(context.selfType.ModulePath.Name)
					}

					result := symbolTable.FindSymbolByPosition(
						context.selfType.Name,
						explicitModule,
						from,
					)
					isInstance = true
					xSymbol = result.GetOrElse(nil)
				} else {
					// !!!!! we've found a self, but function is not flagged as method! Confusion triggered!!!
				}
			} else {
				location := Location{FileName: fileName, Position: pos, Module: context.moduleName}
				xSymbol = symbolTable.SolveType(base.Name, option.None[string](), location)
				isInstance = true
			}

			if xSymbol == nil {
				return nil, nil
			}
			chainedSymbols = append(chainedSymbols, xSymbol)

		case *ast.TypeInfo:
			explicitModule := option.None[string]()
			if base.Identifier.ModulePath != nil {
				explicitModule = option.Some(base.Identifier.ModulePath.Name)
			}

			from := NewLocation(fileName, base.Identifier.StartPosition(), context.moduleName)
			result := symbolTable.FindSymbolByPosition(
				base.Identifier.Name,
				explicitModule,
				from,
			)
			xSymbol = result.GetOrElse(nil)
			chainedSymbols = append(chainedSymbols, xSymbol)

		case *ast.SelectorExpr:
			// X is a SelectorExpr itself, we need to solve the type of base.Sel
			selSymbol, parentChain := solveSelAtSelectorExpr(base, pos, fileName, context, symbolTable, deepLevel+1)
			if selSymbol == nil {
				return nil, nil
			}

			explicitModule := option.None[string]()
			typeName := selSymbol.TypeDef.Name
			switch typeNode := selSymbol.TypeDef.NodeDecl.(type) {
			case *ast.TypeInfo:
				if typeNode.Identifier.ModulePath != nil {
					explicitModule = option.Some(typeNode.Identifier.ModulePath.String())
				}
				typeName = typeNode.Identifier.Name
			}

			location := Location{FileName: fileName, Position: pos, Module: context.moduleName}
			optRes := symbolTable.FindSymbolByPosition(typeName, explicitModule, location)
			xSymbol = optRes.GetOrElse(nil)

			isInstance = true
			chainedSymbols = append(chainedSymbols, parentChain...)

		case *ast.CallExpr:
			ident := base.Identifier
			isInstance = true
			switch i := ident.(type) {
			case *ast.SelectorExpr:
				selSymbol, parentChain := solveSelAtSelectorExpr(i, pos, fileName, context, symbolTable, deepLevel+1)
				if selSymbol == nil {
					return nil, nil
				}
				xSymbol = selSymbol
				chainedSymbols = append(chainedSymbols, parentChain...)

			case *ast.Ident:
				explicitModule := option.None[string]()
				if i.ModulePath != nil {
					explicitModule = option.Some(i.ModulePath.Name)
				}
				from := NewLocation(fileName, pos, context.moduleName)
				sym := symbolTable.FindSymbolByPosition(i.Name, explicitModule, from)
				if sym.IsNone() {
					return nil, nil
				}
				xSymbol = sym.Get()
				chainedSymbols = append(chainedSymbols, xSymbol)
			}

		default:
			return nil, nil
		}*/

	// We've found X type, we are ready to find selectorExpr.Sel inside `X`'s type:
	solveElementType := false
	if deepLevel == 0 {
		solveElementType = false
	}

	// Resolves selectorExpr.Sel identifier under the elements of xSymbol
	childSymbol := resolveChildSymbol(
		xSymbol,
		selectorExpr.Sel.Name,
		context.moduleName,
		fileName,
		symbolTable,
		solveElementType,
		isInstance,
	)
	return childSymbol, chainedSymbols
}

// solveXAtSelectorExpr resolves ast.SelectorExpr.X. It also returns all symbols traversed until X.Sel if applies.
func solveXAtSelectorExpr(selectorExpr *ast.SelectorExpr, pos lsp.Position, fileName string, context astContext, symbolTable *SymbolTable, deepLevel uint) (*Symbol, []*Symbol, bool) {
	// To be able to resolve selectorExpr.Sel, we need to know first what is selectorExpr.X is or what does it return.
	var parentSymbol *Symbol
	chainedSymbols := []*Symbol{}
	isType := false

	switch base := selectorExpr.X.(type) {
	case *ast.Ident:
		// X is a plain Ident. We need to resolve Ident TypeDef:
		// - Ident might be a variable. What's its type? Struct/Enum/Fault?
		// - Ident might be `self`.
		parentSymbolName := base.Name
		if parentSymbolName == "self" {
			// We need to go to parent FunctionDecl and see if `self` is a defined argument
			if context.selfType != nil {
				parentSymbolName = context.selfType.Name
				explicitModule := option.None[string]()
				if context.selfType.ModulePath != nil {
					explicitModule = option.Some(context.selfType.ModulePath.Name)
				}

				from := NewLocation(fileName, context.selfType.StartPosition(), context.moduleName)
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
			// TODO in some completion cases, we are more interested in the symbol of base.Identifier itself than in its type.
			location := Location{FileName: fileName, Position: pos, Module: context.moduleName}
			parentSymbol = symbolTable.findSymbolInLocation(base.Name, location)
			//parentSymbol = symbolTable.SolveType(base.Identifier, location)
		}

		if parentSymbol == nil {
			return nil, nil, false
		}

	case *ast.TypeInfo:
		isType = true
		explicitModule := option.None[string]()
		if base.Identifier.ModulePath != nil {
			explicitModule = option.Some(base.Identifier.ModulePath.Name)
		}

		from := NewLocation(fileName, base.Identifier.StartPosition(), context.moduleName)
		result := symbolTable.FindSymbolByPosition(
			base.Identifier.Name,
			explicitModule,
			from,
		)
		parentSymbol = result.GetOrElse(nil)

	case *ast.SelectorExpr:
		// X is a SelectorExpr itself, we need to solve the type of base.Sel
		selSymbol, selChainedSymbols := solveSelAtSelectorExpr(base, pos, fileName, context, symbolTable, deepLevel+1)
		if selSymbol == nil {
			return nil, nil, false
		}
		parentSymbol = selSymbol
		chainedSymbols = append(chainedSymbols, selChainedSymbols...)

	case *ast.CallExpr:
		ident := base.Identifier
		switch i := ident.(type) {
		case *ast.SelectorExpr:
			parentSymbol, _ = solveSelAtSelectorExpr(i, pos, fileName, context, symbolTable, deepLevel+1)
			if parentSymbol == nil {
				return nil, nil, false
			}
		case *ast.Ident:
			isType = true
			explicitModule := option.None[string]()
			if i.ModulePath != nil {
				explicitModule = option.Some(i.ModulePath.Name)
			}
			from := NewLocation(fileName, pos, context.moduleName)
			sym := symbolTable.FindSymbolByPosition(i.Name, explicitModule, from)
			if sym.IsNone() {
				return nil, nil, false
			}
			parentSymbol = sym.Get()
		}

	default:
		return nil, nil, false
	}

	return parentSymbol, chainedSymbols, isType
}

func resolveChildSymbol(parentSymbol *Symbol,
	nextIdent string, moduleName ModuleName, fileName string, symbolTable *SymbolTable, solveType bool, canRedMembers bool) *Symbol {
	if parentSymbol == nil {
		return nil
	}

	switch parentSymbol.Kind {
	case ast.ENUM, ast.FAULT:
		for _, childRel := range parentSymbol.Children {
			if childRel.Tag == Field && childRel.Child.Identifier == nextIdent {
				return childRel.Child
			} else if childRel.Tag == Method && childRel.Child.Identifier == nextIdent {
				return childRel.Child
			}
		}

		// provide also associated values
		if canRedMembers {
			enumGenDeclNode := parentSymbol.NodeDecl.(*ast.GenDecl)
			symbol := searchIdentInEnumAssocValues(enumGenDeclNode, nextIdent, moduleName, fileName)
			if symbol != nil {
				return symbol
			}
		}

	case ast.ENUM_VALUE:
		// enum value can access methods, and associated values
		enumGenDeclNode := parentSymbol.TypeSymbol.NodeDecl.(*ast.GenDecl)
		symbol := searchIdentInEnumAssocValues(enumGenDeclNode, nextIdent, moduleName, fileName)
		if symbol != nil {
			return symbol
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
		from := NewLocation(fileName, returnType.Range.Start, parentSymbol.Module)
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
				true,
			)
		}
	}

	return nil
}

func searchIdentInEnumAssocValues(enumGenDeclNode *ast.GenDecl, nextIdent string, moduleName ModuleName, fileName string) *Symbol {
	for _, assoc := range enumGenDeclNode.Spec.(*ast.TypeSpec).TypeDescription.(*ast.EnumType).AssociatedValues {
		if assoc.Name.Name == nextIdent {
			return &Symbol{
				Identifier: assoc.Name.Name,
				Module:     moduleName,
				URI:        fileName,
				Range:      assoc.Range,
				NodeDecl:   assoc,
				Kind:       ast.FIELD,
				TypeDef: TypeDefinition{
					Name:      assoc.Type.Identifier.Name,
					IsBuiltIn: assoc.Type.BuiltIn,
				},
			}
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
						Identifier: member.Names[0].Name,
						Module:     moduleName,
						URI:        fileName,
						Range:      member.Range,
						NodeDecl:   member,
						Kind:       ast.FIELD,
						TypeDef: TypeDefinition{
							typeName,
							t.BuiltIn,
							t,
						},
					}
				}
				from := NewLocation(fileName, member.Range.Start, moduleName)
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
					Identifier: member.Names[0].Name,
					Module:     moduleName,
					URI:        fileName,
					Range:      member.Range,
					NodeDecl:   member,
					Kind:       ast.AnonymousStructField,
					TypeDef: TypeDefinition{
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
		if relatedSymbol.Tag == Method && relatedSymbol.Child.Identifier == searchIdent {
			return relatedSymbol.Child
		}
	}

	// Not found, look inside each inlinedCandidates, maybe is a subproperty of them
	for _, inlinedTypeIdent := range inlinedCandidates {
		from := NewLocation(fileName, inlinedTypeIdent.StartPosition(), moduleName)
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
				true,
			)
			if child != nil && child.Identifier == searchIdent {
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
