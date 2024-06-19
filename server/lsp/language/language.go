package language

import (
	"fmt"
	"strings"

	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/pherrymason/c3-lsp/lsp/search_params"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/lsp/symbols_table"
	"github.com/pherrymason/c3-lsp/lsp/utils"
	"github.com/pherrymason/c3-lsp/option"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Language will be the center of knowledge of everything parsed.
type Language struct {
	indexByFQN      IndexStore
	symbolsTable    symbols_table.SymbolsTable
	logger          commonlog.Logger
	languageVersion Version
	debugEnabled    bool
}

func NewLanguage(logger commonlog.Logger, languageVersion option.Option[string]) Language {
	language := Language{
		indexByFQN:      NewIndexStore(),
		symbolsTable:    symbols_table.NewSymbolsTable(),
		logger:          logger,
		languageVersion: GetVersion(languageVersion),
		debugEnabled:    false,
	}

	// Install stdlib symbols
	stdlibModules := language.languageVersion.stdLibSymbols()
	language.indexParsedSymbols(stdlibModules, stdlibModules.DocId())

	language.symbolsTable.Register(stdlibModules, symbols_table.PendingToResolve{})

	return language
}

func (l *Language) RefreshDocumentIdentifiers(doc *document.Document, parser *parser.Parser) {

	//l.logger.Debug(fmt.Sprint("Parsing ", doc.URI))
	parsedModules, pendingTypes := parser.ParseSymbols(doc)
	l.symbolsTable.Register(parsedModules, pendingTypes)
	l.indexParsedSymbols(parsedModules, doc.URI)
}

func (l *Language) DeleteDocument(docId string) {
	l.symbolsTable.DeleteDocument(docId)
	l.indexByFQN.ClearByTag(docId)
}

func (l *Language) indexParsedSymbols(parsedModules symbols_table.UnitModules, docId string) {
	l.indexByFQN.ClearByTag(docId)

	// Register in the index, the root elements
	for _, module := range parsedModules.Modules() {
		for _, fun := range module.ChildrenFunctions {
			l.indexByFQN.RegisterSymbol(fun)
		}
		for _, variable := range module.Variables {
			l.indexByFQN.RegisterSymbol(variable)
		}
		for _, enum := range module.Enums {
			l.indexByFQN.RegisterSymbol(enum)
		}
		for _, fault := range module.Faults {
			l.indexByFQN.RegisterSymbol(fault)
		}
		for _, strukt := range module.Structs {
			l.indexByFQN.RegisterSymbol(strukt)
		}
		for _, def := range module.Defs {
			l.indexByFQN.RegisterSymbol(def)
		}
	}
}

func (l *Language) FindSymbolDeclarationInWorkspace(doc *document.Document, position symbols.Position) option.Option[symbols.Indexable] {

	searchParams := search_params.BuildSearchBySymbolUnderCursor(
		doc,
		*l.symbolsTable.GetByDoc(doc.URI),
		position,
	)

	searchResult := l.findClosestSymbolDeclaration(searchParams, FindDebugger{enabled: l.debugEnabled, depth: 0})

	return searchResult.result
}

func (l *Language) FindHoverInformation(doc *document.Document, params *protocol.HoverParams) option.Option[protocol.Hover] {

	search := search_params.BuildSearchBySymbolUnderCursor(
		doc,
		*l.symbolsTable.GetByDoc(doc.URI),
		symbols.NewPositionFromLSPPosition(params.Position),
	)

	if IsLanguageKeyword(search.Symbol()) {
		return option.None[protocol.Hover]()
	}

	foundSymbolOption := l.findClosestSymbolDeclaration(search, FindDebugger{depth: 0})
	if foundSymbolOption.IsNone() {
		return option.None[protocol.Hover]()
	}

	foundSymbol := foundSymbolOption.Get()

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

	return option.Some(hover)
}

func (l *Language) debug(message string, debugger FindDebugger) {
	if !l.debugEnabled {
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
