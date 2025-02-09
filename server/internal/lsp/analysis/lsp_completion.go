package analysis

import (
	"cmp"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"log"
	"slices"
	"strings"
)

func BuildCompletionList(document *document.Document, pos lsp.Position, storage *document.Storage, symbolTable *SymbolTable) []protocol.CompletionItem {

	_, path := FindNode(document.Ast, pos)
	posCtxt := getContextFromPosition(path, pos, document.Text, ContextHintForCompletion)

	var items []protocol.CompletionItem
	fileName := document.Uri

	editRange := getEditRange(document, pos)
	if posCtxt.isSelExpr == false {
		// Search between globally available symbols.
		moduleScope := symbolTable.scopeTree[fileName].GetModuleScope(posCtxt.moduleName.String())

		// get current scope
		scope := FindClosestScope(moduleScope, pos)
		availableSymbols := symbolTable.GetSymbolsInScope(scope)

		for _, symbol := range availableSymbols {
			if strings.HasPrefix(symbol.Identifier, posCtxt.identUnderCursor) {
				if symbol.Range.HasPosition(pos) {
					// Exclude symbol in current cursor position
					continue
				}
				items = append(items, protocol.CompletionItem{
					Label: symbol.Identifier,
					Kind:  cast.ToPtr(getCompletionKind(symbol)),
					TextEdit: protocol.TextEdit{
						NewText: symbol.Identifier,
						Range:   editRange,
					},
					Documentation: getCompletableDocDocument(symbol.NodeDecl),
					Detail:        getCompletionDetail(symbol),
				})
			}
		}
		// TODO If inside a deeper scope, prefer local symbols.
	} else {
		// We need to solve first SelectorExpr.X!
		parentSymbol, parentSymbols := solveXAtSelectorExpr(posCtxt.selExpr, pos, fileName, posCtxt, symbolTable, 0)
		//canReadMembers := canReadMembersOf(parentSymbol)
		symbols := []*Symbol{}

		parentSymbolKind := parentSymbol.Kind
		collect := SymbolAll
		if parentSymbolKind == ast.VAR || parentSymbolKind == ast.FIELD {
			parentSymbol = symbolTable.SolveType(parentSymbol.TypeDef.Name, NewLocation(fileName, pos, posCtxt.moduleName))
			if parentSymbol.Kind == ast.ENUM {
				// Enum instantiated variables will only have access to methods. Not to other enum values
				collect = SymbolMethod
			}
		}

		// Get all available children symbols of parentSymbol
		symbols = collectChildSymbols(parentSymbol, posCtxt, fileName, collect)

		if parentSymbol.Kind == ast.ENUM_VALUE {
			// Enum values also have access to Enum methods
			enumMethods := collectChildSymbols(parentSymbols[len(parentSymbols)-1], posCtxt, fileName, SymbolMethod)
			symbols = append(symbols, enumMethods...)
		}

		// child symbols collected, filter them
		for _, symbol := range symbols {
			if posCtxt.identUnderCursor == "" || strings.HasPrefix(symbol.Identifier, posCtxt.identUnderCursor) {
				items = append(items, protocol.CompletionItem{
					Label: symbol.GetLabel(),
					Kind:  cast.ToPtr(getCompletionKind(symbol)),
					TextEdit: protocol.TextEdit{
						NewText: symbol.Identifier,
						Range:   editRange,
					},
					Documentation: getCompletableDocDocument(symbol.NodeDecl),
					Detail:        getCompletionDetail(symbol),
				})
			}
		}
	}

	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return cmp.Compare(strings.ToLower(a.TextEdit.(protocol.TextEdit).NewText), strings.ToLower(b.TextEdit.(protocol.TextEdit).NewText))
	})

	return items
}

const (
	SymbolAll    = 1 << 0 // 0001
	SymbolMethod = 1 << 1 // 0010
	SymbolMember = 1 << 2 // 0100
)

func collectChildSymbols(parentSymbol *Symbol, astCtxt astContext, fileName string, symbolFilter int) []*Symbol {
	symbols := []*Symbol{}
	filterMember := symbolFilter&SymbolAll != 0 || symbolFilter&SymbolMember != 0
	filterMethod := symbolFilter&SymbolAll != 0 || symbolFilter&SymbolMethod != 0

	// This logic here is very similar to resolveChildSymbol but we want all symbols
	switch parentSymbol.Kind {
	case ast.ENUM, ast.FAULT:
		for _, childRel := range parentSymbol.Children {
			if childRel.Tag == Field && filterMember {
				symbols = append(symbols, childRel.Child)
			} else if childRel.Tag == Method && filterMethod {
				symbols = append(symbols, childRel.Child)
			}
		}

	case ast.STRUCT:
		genDecl := parentSymbol.NodeDecl.(*ast.GenDecl)
		specType := genDecl.Spec.(*ast.TypeSpec).TypeDescription.(*ast.StructType)
		for _, member := range specType.Fields {
			switch t := member.Type.(type) {
			case *ast.TypeInfo:
				if t.BuiltIn && filterMember {
					symbols = append(symbols, &Symbol{Identifier: member.Names[0].Name,
						Module:   astCtxt.moduleName,
						URI:      fileName,
						Range:    member.Range,
						NodeDecl: member,
						Kind:     ast.FIELD,
						TypeDef: TypeDefinition{
							t.Identifier.String(),
							t.BuiltIn,
							t,
						}})
				}
			case *ast.StructType:
				symbols = append(symbols, &Symbol{
					Identifier: member.Names[0].Name,
					Module:     astCtxt.moduleName,
					URI:        fileName,
					Range:      member.Range,
					NodeDecl:   member,
					Kind:       ast.AnonymousStructField,
					TypeDef: TypeDefinition{
						Name:      "",
						IsBuiltIn: false,
						NodeDecl:  member,
					},
				})
			}
		}

		for _, relatedSymbol := range parentSymbol.Children {
			if relatedSymbol.Tag == Method && filterMethod {
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
		if symbol.NodeDecl.(*ast.FunctionDecl).ParentTypeId.IsSome() {
			return protocol.CompletionItemKindMethod
		}

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

	case ast.DEF:
		return protocol.CompletionItemKindTypeParameter

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
		if s.NodeDecl.(*ast.GenDecl).Spec.(*ast.DefSpec).ResolvesToType {
			codeifier := defValueToCodeVisitor{}
			walk.Walk(&codeifier, s.NodeDecl.(*ast.GenDecl), "")
			detail = "Type alias for '" + codeifier.code + "'"
		} else if strings.HasPrefix(s.Identifier, "@") {
			detail = "Alias for '" + s.NodeDecl.(*ast.GenDecl).Spec.(*ast.DefSpec).Value.(*ast.Ident).Name + "'"
		} else {
			codeifier := defValueToCodeVisitor{}
			walk.Walk(&codeifier, s.NodeDecl.(*ast.GenDecl), "")
			detail = "Alias for '" + codeifier.code + "'"
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

// If `true`, this indexable is a type, and so one can access its parent type's (itself)
// associated members, as well as methods.
// If `false`, this indexable is a variable or similar, so its parent type is distinct
// from the indexable itself, therefore only methods can be accessed.
func canReadMembersOf(s *Symbol) bool {
	switch s.Kind {
	case ast.STRUCT, ast.ENUM, ast.FAULT:
		return true
	case ast.DEF:
		return s.NodeDecl.(*ast.GenDecl).Spec.(*ast.DefSpec).ResolvesToType
	default:
		return false
	}
}
