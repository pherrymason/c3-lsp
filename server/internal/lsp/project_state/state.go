package project_state

import (
	"fmt"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// ProjectState will be the center of knowledge of everything parsed.
type ProjectState struct {
	_documents             map[string]*document.Document
	documents              *document.DocumentStore
	symbolsTable           symbols_table.SymbolsTable
	indexByFQN             IndexStore // TODO simplify this and use trie.Trie directly!
	languageVersion        Version
	calculatingDiagnostics bool

	logger       commonlog.Logger
	debugEnabled bool
}

func NewProjectState(logger commonlog.Logger, languageVersion option.Option[string], debug bool) ProjectState {
	projectState := ProjectState{
		_documents:      map[string]*document.Document{},
		documents:       document.NewDocumentStore(fs.FileStorage{}),
		symbolsTable:    symbols_table.NewSymbolsTable(),
		indexByFQN:      NewIndexStore(),
		logger:          logger,
		languageVersion: GetVersion(languageVersion),
		debugEnabled:    debug,
	}

	// Install stdlib symbols
	stdlibModules := projectState.languageVersion.stdLibSymbols()
	projectState.indexParsedSymbols(stdlibModules, stdlibModules.DocId())

	projectState.symbolsTable.Register(stdlibModules, symbols_table.PendingToResolve{})

	return projectState
}

func (s ProjectState) GetProjectRootURI() string {
	return s.documents.RootURI
}
func (s *ProjectState) SetProjectRootURI(rootURI string) {
	s.documents.RootURI = rootURI
}

func (s *ProjectState) IsCalculatingDiagnostics() bool {
	return s.calculatingDiagnostics
}

func (s *ProjectState) SetCalculateDiagnostics(running bool) {
	s.calculatingDiagnostics = running
}

func (s *ProjectState) GetDocument(docId string) *document.Document {
	return s._documents[docId]
}

func (s *ProjectState) GetUnitModulesByDoc(docId string) *symbols_table.UnitModules {
	value := s.symbolsTable.GetByDoc(docId)
	return value
}

func (s *ProjectState) GetAllUnitModules() map[protocol.DocumentUri]symbols_table.UnitModules {
	return s.symbolsTable.All()
}

func (s *ProjectState) SearchByFQN(query string) []symbols.Indexable {
	return s.indexByFQN.SearchByFQN(query)
}

func (s *ProjectState) RefreshDocumentIdentifiers(doc *document.Document, parser *parser.Parser) {
	parsedModules, pendingTypes := parser.ParseSymbols(doc)

	// Store elements in the state
	s._documents[doc.URI] = doc
	s.symbolsTable.Register(parsedModules, pendingTypes)
	s.indexParsedSymbols(parsedModules, doc.URI)
}

func (s *ProjectState) DeleteDocument(docId string) {
	s.symbolsTable.DeleteDocument(docId)
	s.indexByFQN.ClearByTag(docId)
}

func (s *ProjectState) RenameDocument(oldDocId string, newDocId string) {
	s.indexByFQN.ClearByTag(oldDocId)
	s.symbolsTable.RenameDocument(oldDocId, newDocId)

	x := s.symbolsTable.GetByDoc(newDocId)
	s.indexParsedSymbols(*x, newDocId)
}

func (s *ProjectState) UpdateDocument(docURI protocol.DocumentUri, changes []interface{}, parser *parser.Parser) {
	docId := utils.NormalizePath(docURI)
	doc, ok := s._documents[docId]
	if !ok {
		return
	}

	doc.ApplyChanges(changes)

	s.RefreshDocumentIdentifiers(doc, parser)
}

func (s *ProjectState) CloseDocument(uri protocol.DocumentUri) {
	docId := utils.NormalizePath(uri)
	delete(s._documents, docId)
}

func (s *ProjectState) indexParsedSymbols(parsedModules symbols_table.UnitModules, docId string) {
	s.indexByFQN.ClearByTag(docId)

	// Register in the index, the root elements
	for _, module := range parsedModules.Modules() {
		for _, fun := range module.ChildrenFunctions {
			s.indexByFQN.RegisterSymbol(fun)
		}
		for _, variable := range module.Variables {
			s.indexByFQN.RegisterSymbol(variable)
		}
		for _, enum := range module.Enums {
			s.indexByFQN.RegisterSymbol(enum)
		}
		for _, fault := range module.Faults {
			s.indexByFQN.RegisterSymbol(fault)
		}
		for _, strukt := range module.Structs {
			s.indexByFQN.RegisterSymbol(strukt)
		}
		for _, def := range module.Defs {
			s.indexByFQN.RegisterSymbol(def)
		}
	}
}

func (s *ProjectState) debug(message string, debugger FindDebugger) {
	if !s.debugEnabled {
		return
	}

	maxo := utils.Min(debugger.depth, 20)
	prep := "|" + strings.Repeat(".", maxo)
	if debugger.depth > 8 {
		prep = fmt.Sprintf("%s (%d)", prep, debugger.depth)
	}

	s.logger.Debug(fmt.Sprintf("%s %s", prep, message))
}
