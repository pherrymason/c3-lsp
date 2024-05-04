package language

import (
	"strings"

	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Language will be the center of knowledge of everything parsed.
type Language struct {
	index                  IndexStore
	functionTreeByDocument map[protocol.DocumentUri]parser.ParsedModules
	logger                 commonlog.Logger
}

func NewLanguage(logger commonlog.Logger) Language {
	return Language{
		index:                  NewIndexStore(),
		functionTreeByDocument: make(map[protocol.DocumentUri]parser.ParsedModules),
		logger:                 logger,
	}
}

func (l *Language) RefreshDocumentIdentifiers(doc *document.Document, parser *parser.Parser) {
	parsedSymbols := parser.ParseSymbols(doc)

	l.functionTreeByDocument[parsedSymbols.DocId()] = parsedSymbols
}

func (l *Language) BuildCompletionList(doc *document.Document, position protocol.Position) []protocol.CompletionItem {
	// 1 - TODO find scoped symbols starting with same letters
	// 2 - TODO if previous character is '.', find previous symbol and if a struct, complete only with struct methods
	// 3 - TODO if writing function call arguments, complete with argument names. ¿Feasible?

	symbolInPosition, _ := doc.SymbolInPosition(
		protocol.Position{
			Line:      position.Line,
			Character: position.Character - 1,
		})

	// Find symbols in document
	moduleSymbols := l.functionTreeByDocument[doc.URI]
	scopeSymbols := l.findAllScopeSymbols(&moduleSymbols, position)

	var items []protocol.CompletionItem
	for _, storedIdentifier := range scopeSymbols {
		if !strings.HasPrefix(storedIdentifier.GetName(), symbolInPosition) {
			continue
		}

		tempKind := storedIdentifier.GetKind()

		items = append(items, protocol.CompletionItem{
			Label: storedIdentifier.GetName(),
			Kind:  &tempKind,
		})
	}

	return items
}

const (
	AnyPosition FindMode = iota
	InScope
)

type FindMode int

func (l *Language) FindSymbolDeclarationInWorkspace(doc *document.Document, position protocol.Position) (indexables.Indexable, error) {
	searchParams, err := NewSearchParamsFromPosition(doc, position)
	if err != nil {
		return indexables.Variable{}, err
	}

	symbol := l.findClosestSymbolDeclaration(searchParams, DebugFind{depth: 0})

	return symbol, nil
}

func (l *Language) FindHoverInformation(doc *document.Document, params *protocol.HoverParams) (protocol.Hover, error) {

	//module := l.findModuleInPosition(doc.URI, params.Position)
	//fmt.Println(module)

	search, err := NewSearchParamsFromPosition(doc, params.Position)
	if err != nil {
		return protocol.Hover{}, err
	}

	if IsLanguageKeyword(search.selectedSymbol.token) {
		return protocol.Hover{}, err
	}

	foundSymbol := l.findClosestSymbolDeclaration(search, DebugFind{depth: 0})
	if foundSymbol == nil {
		return protocol.Hover{}, nil
	}

	// expected behaviour:
	// hovering on variables: display variable type + any description
	// hovering on functions: display function signature
	// hovering on members: same as variable
	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: foundSymbol.GetHoverInfo(),
		},
	}

	return hover, nil
}

func IsLanguageKeyword(symbol string) bool {
	keywords := []string{
		"void", "bool", "char", "double",
		"float", "float16", "int128", "ichar",
		"int", "iptr", "isz", "long",
		"short", "uint128", "uint", "ulong",
		"uptr", "ushort", "usz", "float128",
		"any", "anyfault", "typeid", "assert",
		"asm", "bitstruct", "break", "case",
		"catch", "const", "continue", "def",
		"default", "defer", "distinct", "do",
		"else", "enum", "extern", "false",
		"fault", "for", "foreach", "foreach_r",
		"fn", "tlocal", "if", "inline",
		"import", "macro", "module", "nextcase",
		"null", "return", "static", "struct",
		"switch", "true", "try", "union",
		"var", "while",

		"$alignof", "$assert", "$case", "$default",
		"$defined", "$echo", "$embed", "$exec",
		"$else", "$endfor", "$endforeach", "$endif",
		"$endswitch", "$eval", "$evaltype", "$error",
		"$extnameof", "$for", "$foreach", "$if",
		"$include", "$nameof", "$offsetof", "$qnameof",
		"$sizeof", "$stringify", "$switch", "$typefrom",
		"$typeof", "$vacount", "$vatype", "$vaconst",
		"$varef", "$vaarg", "$vaexpr", "$vasplat",
	}
	for _, w := range keywords {
		if w == symbol {
			return true
		}
	}
	return false
}
