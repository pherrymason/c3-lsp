package project_state

import (
	"fmt"
	"strings"

	trie "github.com/pherrymason/c3-lsp/internal/lsp/symbol_trie"
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

// ProjectState is the central state manager that holds all parsed information.
type ProjectState struct {
	documents    *document.DocumentStore    // Active documents store
	symbolsTable symbols_table.SymbolsTable // Source of truth - hierarchical storage (Document → Module → Symbols)
	fqnIndex     *trie.Trie                 // Fast lookup index - trie-based Full Qualified Name search (module::symbol)

	diagnostics map[string][]protocol.Diagnostic

	logger       commonlog.Logger
	debugEnabled bool
}

func NewProjectState(logger commonlog.Logger, languageVersion option.Option[string], debug bool) ProjectState {
	projectState := ProjectState{
		documents:    document.NewDocumentStore(fs.FileStorage{}),
		symbolsTable: symbols_table.NewSymbolsTable(),
		fqnIndex:     trie.NewTrie(),
		diagnostics:  make(map[string][]protocol.Diagnostic),

		logger:       logger,
		debugEnabled: debug,
	}

	return projectState
}

func (s ProjectState) GetProjectRootURI() string {
	return s.documents.RootURI
}
func (s *ProjectState) SetProjectRootURI(rootURI string) {
	s.documents.RootURI = rootURI
}

func (s *ProjectState) GetDocument(docId string) *document.Document {
	doc, _ := s.documents.Get(docId)
	return doc
}

func (s *ProjectState) GetUnitModulesByDoc(docId string) *symbols_table.UnitModules {
	value := s.symbolsTable.GetByDoc(docId)
	return value
}

func (s *ProjectState) GetAllUnitModules() map[protocol.DocumentUri]symbols_table.UnitModules {
	return s.symbolsTable.All()
}

func (s *ProjectState) SearchByFQN(query string) []symbols.Indexable {
	return s.fqnIndex.Search(query)
}

func (s *ProjectState) GetDocumentDiagnostics() map[string][]protocol.Diagnostic {
	return s.diagnostics
}

func (s *ProjectState) SetLanguageVersion(languageVersion string, c3cLibPath string) {
	stdlibModules := LoadStdLib(s.logger, languageVersion, c3cLibPath)
	s.indexParsedSymbols(stdlibModules, stdlibModules.DocId())

	s.symbolsTable.Register(stdlibModules, symbols_table.PendingToResolve{})
}

func (s *ProjectState) SetDocumentDiagnostics(docId string, diagnostics []protocol.Diagnostic) {
	s.diagnostics[docId] = diagnostics
}

func (s *ProjectState) ClearDocumentDiagnostics() {
	for k := range s.diagnostics {
		delete(s.diagnostics, k)
	}
}
func (s *ProjectState) RemoveDocumentDiagnostics(docId string) {
	delete(s.diagnostics, docId)
}

func (s *ProjectState) RefreshDocumentIdentifiers(doc *document.Document, parser *parser.Parser) {
	parsedModules, pendingTypes := parser.ParseSymbols(doc)

	// Store elements in the state
	s.documents.Set(doc)
	s.symbolsTable.Register(parsedModules, pendingTypes)
	s.indexParsedSymbols(parsedModules, doc.URI)
}

func (s *ProjectState) DeleteDocument(docId string) {
	s.symbolsTable.DeleteDocument(docId)
	s.fqnIndex.ClearByTag(docId)
}

func (s *ProjectState) RenameDocument(oldDocId string, newDocId string) {
	s.fqnIndex.ClearByTag(oldDocId)
	s.symbolsTable.RenameDocument(oldDocId, newDocId)

	x := s.symbolsTable.GetByDoc(newDocId)
	s.indexParsedSymbols(*x, newDocId)
}

func (s *ProjectState) UpdateDocument(docURI protocol.DocumentUri, changes []interface{}, parser *parser.Parser) {
	docId := utils.NormalizePath(docURI)
	doc, ok := s.documents.Get(docId)
	if !ok {
		return
	}

	doc.ApplyChanges(changes)

	s.RefreshDocumentIdentifiers(doc, parser)
}

func (s *ProjectState) CloseDocument(uri protocol.DocumentUri) {
	docId := utils.NormalizePath(uri)
	s.documents.Close(docId)
}

func (s *ProjectState) indexParsedSymbols(parsedModules symbols_table.UnitModules, docId string) {
	s.fqnIndex.ClearByTag(docId)

	// Register in the index, the root elements
	for _, module := range parsedModules.Modules() {
		for _, fun := range module.ChildrenFunctions {
			s.fqnIndex.Insert(fun)
		}
		for _, variable := range module.Variables {
			s.fqnIndex.Insert(variable)
		}
		for _, enum := range module.Enums {
			s.fqnIndex.Insert(enum)
		}
		for _, fault := range module.Faults {
			s.fqnIndex.Insert(fault)
		}
		for _, strukt := range module.Structs {
			s.fqnIndex.Insert(strukt)
		}
		for _, def := range module.Defs {
			s.fqnIndex.Insert(def)
		}
		for _, distinct := range module.Distincts {
			s.fqnIndex.Insert(distinct)
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
