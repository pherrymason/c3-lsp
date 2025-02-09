package analysis

import (
	"cmp"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"log"
	"slices"
	"strings"
)

func BuildCompletionList(document *document.Document, pos lsp.Position, storage *document.Storage, symbolTable *SymbolTable) []protocol.CompletionItem {

	nodeAtPosition, path := FindNode(document.Ast, pos)
	astCtxt := getASTNodeContext(path)

	var search string

	if path != nil && nodeAtPosition != nil {
		switch n := nodeAtPosition.(type) {
		case *ast.Ident:
			search = n.Name
		case *ast.ErrorNode:
			if n.DetectedIdent != nil {
				search = n.DetectedIdent.Content
			} else {
				// TODO n.Content might contain dirt
				search = n.Content
			}
			astCtxt = getAstContextFromString(astCtxt, search)
		}
	}

	var items []protocol.CompletionItem
	fileName := document.Uri

	editRange := getEditRange(document, pos)
	if astCtxt.isSelExpr == false {
		// Search between globally available symbols.
		moduleScope := symbolTable.scopeTree[fileName].GetModuleScope(astCtxt.moduleName.String())

		// get current scope
		scope := FindClosestScope(moduleScope, pos)
		availableSymbols := symbolTable.GetSymbolsInScope(scope)

		for _, symbol := range availableSymbols {
			if strings.HasPrefix(symbol.Name, search) {
				if symbol.Range.HasPosition(pos) {
					// Exclude symbol in current cursor position
					continue
				}
				items = append(items, protocol.CompletionItem{
					Label: symbol.Name,
					Kind:  cast.ToPtr(getCompletionKind(symbol)),
					TextEdit: protocol.TextEdit{
						NewText: symbol.Name,
						Range:   editRange,
					},
					Documentation: getCompletableDocDocument(symbol.NodeDecl),
					Detail:        getCompletionDetail(symbol),
				})
			}
		}
		// TODO If inside a deeper scope, prefer local symbols.
	} else {
		// When autocompleting a selector Expression, search is selectorExpr.Sel.Name
		if astCtxt.selExpr.Sel == nil {
			search = ""
		} else {
			search = astCtxt.selExpr.Sel.Name
		}

		// We need to solve first SelectorExpr.X!
		parentSymbol := solveXAtSelectorExpr(astCtxt.selExpr, pos, fileName, astCtxt, symbolTable, 0)

		// Get all available children symbols of parentSymbol
		symbols := collectChildSymbols(parentSymbol, astCtxt, fileName)

		// child symbols collected, filter them
		for _, symbol := range symbols {
			if strings.HasPrefix(symbol.Name, search) {
				items = append(items, protocol.CompletionItem{
					Label: symbol.Name,
					Kind:  cast.ToPtr(getCompletionKind(symbol)),
					TextEdit: protocol.TextEdit{
						NewText: symbol.Name,
						Range:   editRange,
					},
					Documentation: getCompletableDocDocument(symbol.NodeDecl),
					Detail:        getCompletionDetail(symbol),
				})
			}
		}
	}

	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return cmp.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
	})

	return items
}

func collectChildSymbols(parentSymbol *Symbol, astCtxt astContext, fileName string) []*Symbol {
	symbols := []*Symbol{}
	// This logic here is very similar to resolveChildSymbol but we want all symbols
	switch parentSymbol.Kind {
	case ast.ENUM, ast.FAULT:
		for _, childRel := range parentSymbol.Children {
			if childRel.Tag == Field /*&& childRel.Child.Name == nextIdent*/ {
				symbols = append(symbols, childRel.Child)
			} else if childRel.Tag == Method /*&& childRel.Child.Name == nextIdent*/ {
				symbols = append(symbols, childRel.Child)
			}
		}

	case ast.STRUCT:
		genDecl := parentSymbol.NodeDecl.(*ast.GenDecl)
		specType := genDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType)
		for _, member := range specType.Fields {
			switch t := member.Type.(type) {
			case *ast.TypeInfo:
				if t.BuiltIn {
					symbols = append(symbols, &Symbol{Name: member.Names[0].Name,
						Module:   astCtxt.moduleName,
						URI:      fileName,
						Range:    member.Range,
						NodeDecl: member,
						Kind:     ast.FIELD,
						Type: TypeDefinition{
							t.Identifier.String(),
							t.BuiltIn,
							t,
						}})
				}
			case *ast.StructType:
				symbols = append(symbols, &Symbol{
					Name:     member.Names[0].Name,
					Module:   astCtxt.moduleName,
					URI:      fileName,
					Range:    member.Range,
					NodeDecl: member,
					Kind:     ast.AnonymousStructField,
					Type: TypeDefinition{
						Name:      "",
						IsBuiltIn: false,
						NodeDecl:  member,
					},
				})
			}
		}

		for _, relatedSymbol := range parentSymbol.Children {
			if relatedSymbol.Tag == Method {
				symbols = append(symbols, relatedSymbol.Child)
			}
		}
	}
	return symbols
}

func getEditRange(document *document.Document, pos lsp.Position) protocol.Range {
	// Some string manipulation...
	// Find where current ident starts
	index := pos.IndexIn(document.Text)
	log.Printf("%c", rune(document.Text[index-1]))
	if rune(document.Text[index-1]) == '.' {
		// We will replace just until this point
		return protocol.Range{
			Start: lsp.NewPositionFromIndex(index, document.Text).ToProtocol(),
			End:   lsp.NewPositionFromIndex(index, document.Text).ToProtocol(),
		}
	} else if !utils.IsAZ09_(rune(document.Text[index-1])) {
		panic("not a valid range")
	}

	symbolStart := 0
	for i := index - 1; i >= 0; i-- {
		r := rune(document.Text[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained first character of symbol
			symbolStart = i + 1
			break
		}
	}

	return protocol.Range{
		Start: lsp.NewPositionFromIndex(symbolStart, document.Text).ToProtocol(),
		End:   lsp.NewPositionFromIndex(index, document.Text).ToProtocol(),
	}
}

func getCompletionKind(symbol *Symbol) protocol.CompletionItemKind {
	switch symbol.Kind {
	case ast.FUNCTION:
		return protocol.CompletionItemKindFunction

	case ast.METHOD:
		return protocol.CompletionItemKindMethod
	case ast.MACRO:
		return protocol.CompletionItemKindFunction

	case ast.ENUM, ast.FAULT:
		return protocol.CompletionItemKindEnum
	case ast.ENUM_VALUE:
		return protocol.CompletionItemKindEnumMember

	case ast.VAR:
		return protocol.CompletionItemKindVariable

	case ast.CONST:
		return protocol.CompletionItemKindConstant
	case ast.STRUCT:
		return protocol.CompletionItemKindStruct
	case ast.INTERFACE:
		return protocol.CompletionItemKindInterface
	case ast.FIELD:
		return protocol.CompletionItemKindField

	default:
		return protocol.CompletionItemKindText
	}
}

func getCompletableDocDocument(node ast.Node) *protocol.MarkupContent {
	docComment := node.GetDocComment()
	if docComment.IsNone() || docComment.Get().GetBody() == "" {
		return nil
	} else {
		return &protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: docComment.Get().GetBody(),
		}
	}
}

func getCompletionDetail(s *Symbol) *string {
	var detail string
	switch s.Kind {
	case ast.FUNCTION:
		// TODO
		detail = functionDescriptionString(s)
	case ast.MACRO:
		detail = macroDescriptionString(s, false)
	case ast.ENUM:
		detail = "Enum"
	case ast.ENUM_VALUE:
		detail = "Enum Value"
	case ast.FAULT:
		detail = "Fault"
	case ast.VAR:
		detail = s.Type.Name
	case ast.CONST:
		detail = s.Type.Name
	case ast.DEF:
		if false { //d.resolvesToType.IsSome() {
			//return "Type"
		} else if strings.HasPrefix(s.Name, "@") {
			detail = ""
		} else {
			// TODO: Resolve the identifier and display its information?
			detail = "Alias for '" + "???" + "'"
		}
	case ast.STRUCT:
		detail = "Type"
	case ast.INTERFACE:
		detail = "Interface"
	case ast.DISTINCT:
		detail = "Type"
	case ast.FIELD:
		switch s.NodeDecl.(type) {
		case *ast.EnumValue:
			detail = "Enum Value"
		case *ast.FaultMember:
			detail = "Fault Constant"
		case *ast.StructField:
			detail = "Struct member"
		}
	default:
		detail = ""
	}

	if detail == "" {
		return nil
	}
	return &detail
}
