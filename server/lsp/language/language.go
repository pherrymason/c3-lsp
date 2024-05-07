package language

import (
	"fmt"
	"strings"

	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/pherrymason/c3-lsp/lsp/utils"
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

func (l *Language) FindSymbolDeclarationInWorkspace(doc *document.Document, position indexables.Position) (indexables.Indexable, error) {
	searchParams, err := NewSearchParamsFromPosition(doc, position)
	if err != nil {
		return indexables.Variable{}, err
	}

	symbol := l.findClosestSymbolDeclaration(searchParams, FindDebugger{depth: 0})

	return symbol, nil
}

func (l *Language) FindHoverInformation(doc *document.Document, params *protocol.HoverParams) (protocol.Hover, error) {

	//module := l.findModuleInPosition(doc.URI, params.Position)
	//fmt.Println(module)

	search, err := NewSearchParamsFromPosition(doc, indexables.NewPositionFromLSPPosition(params.Position))
	if err != nil {
		return protocol.Hover{}, err
	}

	if IsLanguageKeyword(search.selectedToken.Token) {
		return protocol.Hover{}, err
	}

	foundSymbol := l.findClosestSymbolDeclaration(search, FindDebugger{depth: 0})
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

func (l *Language) debug(message string, debugger FindDebugger) {
	if !debugger.enabled {
		return
	}

	maxo := utils.Min(debugger.depth, 20)
	prep := "|" + strings.Repeat(".", maxo)
	if debugger.depth > 8 {
		prep = fmt.Sprintf("%s (%d)", prep, debugger.depth)
	}

	l.logger.Debug(fmt.Sprintf("%s %s", prep, message))
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
