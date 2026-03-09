package project_state

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"

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
	revision     uint64

	docModules       map[string][]string
	moduleImports    map[string]map[string]struct{}
	moduleDependents map[string]map[string]struct{}
	moduleSignatures map[string]uint64
	lastInvalidation InvalidationScope

	// stateMu protects all mutable shared state in ProjectState (documents, symbol tables,
	// indexes, diagnostics, dependency graph metadata and snapshot rebuilds).
	stateMu *sync.RWMutex
	// parseMu serializes parsing work to keep parser/tree-sitter interactions deterministic.
	// Lock ordering rule: never acquire parseMu while holding stateMu.
	// Parse first under parseMu, then commit parsed state under stateMu.
	parseMu  *sync.Mutex
	snapshot atomic.Value // *ProjectSnapshot

	documentLocksMu *sync.Mutex
	documentLocks   map[string]*sync.Mutex
}

func NewProjectState(logger commonlog.Logger, languageVersion option.Option[string], debug bool) ProjectState {
	projectState := ProjectState{
		documents:     document.NewDocumentStore(fs.FileStorage{}),
		symbolsTable:  symbols_table.NewSymbolsTable(),
		fqnIndex:      trie.NewTrie(),
		diagnostics:   make(map[string][]protocol.Diagnostic),
		documentLocks: make(map[string]*sync.Mutex),

		logger:           logger,
		debugEnabled:     debug,
		docModules:       make(map[string][]string),
		moduleImports:    make(map[string]map[string]struct{}),
		moduleDependents: make(map[string]map[string]struct{}),
		moduleSignatures: make(map[string]uint64),
		stateMu:          &sync.RWMutex{},
		parseMu:          &sync.Mutex{},
		documentLocksMu:  &sync.Mutex{},
	}
	projectState.rebuildSnapshotLocked()

	return projectState
}

func (s *ProjectState) Snapshot() *ProjectSnapshot {
	v := s.snapshot.Load()
	if v == nil {
		return nil
	}

	return v.(*ProjectSnapshot)
}

func (s *ProjectState) rebuildSnapshotLocked() {
	all := s.symbolsTable.All()
	allCopy := make(map[protocol.DocumentUri]symbols_table.UnitModules, len(all))
	for k, v := range all {
		allCopy[k] = v
	}
	modulesByName, docsByModule, moduleNamesByShort := buildSnapshotIndexes(allCopy)
	scopeIdx := buildScopeCompletionIndex(allCopy)

	s.snapshot.Store(&ProjectSnapshot{
		revision:           s.Revision(),
		allUnitModules:     allCopy,
		fqnIndex:           s.fqnIndex.Clone(),
		modulesByName:      modulesByName,
		docsByModule:       docsByModule,
		moduleNamesByShort: moduleNamesByShort,
		scopeIndex:         scopeIdx,
	})
}

func (s *ProjectState) GetProjectRootURI() string {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.documents.RootURI
}
func (s *ProjectState) SetProjectRootURI(rootURI string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.documents.RootURI = rootURI
}

func (s *ProjectState) GetDocument(docId string) *document.Document {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	doc, _ := s.documents.Get(docId)
	return doc
}

func (s *ProjectState) GetDocumentByNormalizedID(docId string) *document.Document {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	doc, _ := s.documents.GetNormalized(docId)
	return doc
}

func (s *ProjectState) GetUnitModulesByDoc(docId string) *symbols_table.UnitModules {
	snapshot := s.Snapshot()
	if snapshot == nil {
		return nil
	}

	return snapshot.GetUnitModulesByDoc(docId)
}

func (s *ProjectState) GetAllUnitModules() map[protocol.DocumentUri]symbols_table.UnitModules {
	snapshot := s.Snapshot()
	if snapshot == nil {
		return map[protocol.DocumentUri]symbols_table.UnitModules{}
	}

	return snapshot.GetAllUnitModules()
}

// ForEachModule calls fn for every module across all documents in the current
// snapshot.  It is a convenience wrapper around ProjectSnapshot.ForEachModule
// for callers that hold a *ProjectState rather than a snapshot reference.
func (s *ProjectState) ForEachModule(fn func(module *symbols.Module)) {
	s.Snapshot().ForEachModule(fn)
}

// ForEachModuleUntil calls fn for every module, stopping early when fn returns true.
// Returns true if early termination occurred.
func (s *ProjectState) ForEachModuleUntil(fn func(module *symbols.Module) bool) bool {
	return s.Snapshot().ForEachModuleUntil(fn)
}

func (s *ProjectState) GetDocumentsForModules(moduleNames []string) []string {
	if len(moduleNames) == 0 {
		return nil
	}

	snapshot := s.Snapshot()
	if snapshot != nil {
		docs := map[string]struct{}{}
		for _, moduleName := range moduleNames {
			for _, docURI := range snapshot.DocsByModule(moduleName) {
				docs[string(docURI)] = struct{}{}
			}
		}
		return mapKeys(docs)
	}

	need := make(map[string]struct{}, len(moduleNames))
	for _, moduleName := range moduleNames {
		need[moduleName] = struct{}{}
	}

	docs := map[string]struct{}{}
	for docURI, unitModules := range s.GetAllUnitModules() {
		for _, module := range unitModules.Modules() {
			if _, ok := need[module.GetName()]; ok {
				docs[string(docURI)] = struct{}{}
				break
			}
		}
	}

	return mapKeys(docs)
}

func (s *ProjectState) SearchByFQN(query string) []symbols.Indexable {
	snapshot := s.Snapshot()
	if snapshot == nil {
		return nil
	}

	return snapshot.SearchByFQN(query)
}

func (s *ProjectState) GetModuleImports(moduleName string) []string {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	importsSet, ok := s.moduleImports[moduleName]
	if !ok {
		return nil
	}

	imports := make([]string, 0, len(importsSet))
	for imported := range importsSet {
		imports = append(imports, imported)
	}
	sort.Strings(imports)

	return imports
}

func (s *ProjectState) GetModuleDependents(moduleName string) []string {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	dependentsSet, ok := s.moduleDependents[moduleName]
	if !ok {
		return nil
	}

	dependents := make([]string, 0, len(dependentsSet))
	for dependent := range dependentsSet {
		dependents = append(dependents, dependent)
	}
	sort.Strings(dependents)

	return dependents
}

func (s *ProjectState) GetImpactedModules(changedModules []string) []string {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.getImpactedModulesUnlocked(changedModules)
}

func (s *ProjectState) getImpactedModulesUnlocked(changedModules []string) []string {
	if len(changedModules) == 0 {
		return nil
	}

	visited := make(map[string]struct{})
	queue := append([]string(nil), changedModules...)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if _, ok := visited[current]; ok {
			continue
		}
		visited[current] = struct{}{}

		for dependent := range s.moduleDependents[current] {
			if _, ok := visited[dependent]; !ok {
				queue = append(queue, dependent)
			}
		}
	}

	impacted := make([]string, 0, len(visited))
	for moduleName := range visited {
		impacted = append(impacted, moduleName)
	}
	sort.Strings(impacted)

	return impacted
}

func (s *ProjectState) GetLastInvalidationScope() InvalidationScope {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	return s.lastInvalidation
}

func (s *ProjectState) GetDocumentDiagnostics() map[string][]protocol.Diagnostic {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	copy := make(map[string][]protocol.Diagnostic, len(s.diagnostics))
	for k, v := range s.diagnostics {
		copy[k] = append([]protocol.Diagnostic(nil), v...)
	}

	return copy
}

func (s *ProjectState) Revision() uint64 {
	return atomic.LoadUint64(&s.revision)
}

func (s *ProjectState) bumpRevision() {
	atomic.AddUint64(&s.revision, 1)
}

func (s *ProjectState) SetLanguageVersion(languageVersion string, c3cLibPath string) {
	stdlibModules := LoadStdLib(s.logger, languageVersion, c3cLibPath, func(rebuilt symbols_table.UnitModules) {
		s.logger.Info("applying rebuilt stdlib modules in memory", "version", languageVersion)
		s.applyStdlibModules(rebuilt)
	})
	s.applyStdlibModules(stdlibModules)
}

func (s *ProjectState) applyStdlibModules(stdlibModules symbols_table.UnitModules) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.indexParsedSymbols(stdlibModules, stdlibModules.DocId())
	s.symbolsTable.Register(stdlibModules, symbols_table.PendingToResolve{})
	s.bumpRevision()
	s.rebuildSnapshotLocked()
}

func (s *ProjectState) SetDocumentDiagnostics(docId string, diagnostics []protocol.Diagnostic) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.diagnostics[docId] = diagnostics
}

func (s *ProjectState) ClearDocumentDiagnostics() {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	for k := range s.diagnostics {
		delete(s.diagnostics, k)
	}
}
func (s *ProjectState) RemoveDocumentDiagnostics(docId string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	delete(s.diagnostics, docId)
}

func (s *ProjectState) LockDocument(docId string) func() {
	docLock := s.getDocumentLock(docId)
	docLock.Lock()

	return func() {
		docLock.Unlock()
	}
}

func (s *ProjectState) getDocumentLock(docId string) *sync.Mutex {
	s.documentLocksMu.Lock()
	defer s.documentLocksMu.Unlock()

	docLock, ok := s.documentLocks[docId]
	if !ok {
		docLock = &sync.Mutex{}
		s.documentLocks[docId] = docLock
	}

	return docLock
}

func (s *ProjectState) RefreshDocumentIdentifiers(doc *document.Document, parser *parser.Parser) {
	refreshStarted := time.Now()
	var oldModules []string
	var oldSignatures map[string]uint64

	firstLockStart := time.Now()
	s.stateMu.RLock()
	firstLockWait := time.Since(firstLockStart)
	oldModules = append([]string(nil), s.docModules[doc.URI]...)
	oldSignatures = make(map[string]uint64, len(oldModules))
	for _, moduleName := range oldModules {
		if signature, ok := s.moduleSignatures[moduleName]; ok {
			oldSignatures[moduleName] = signature
		}
	}
	s.stateMu.RUnlock()

	parseLockStart := time.Now()
	s.parseMu.Lock()
	parseLockWait := time.Since(parseLockStart)

	parseStart := time.Now()
	parsedModules, pendingTypes := parser.ParseSymbols(doc)
	parseDuration := time.Since(parseStart)
	s.parseMu.Unlock()

	commitLockStart := time.Now()
	s.stateMu.Lock()
	commitLockWait := time.Since(commitLockStart)
	defer s.stateMu.Unlock()

	// Store elements in the state
	s.documents.Set(doc)
	s.symbolsTable.Register(parsedModules, pendingTypes)
	s.indexParsedSymbols(parsedModules, doc.URI)
	s.updateInvalidationScope(oldModules, oldSignatures, parsedModules)
	s.rebuildDependencyGraphForDocument(doc.URI, parsedModules)

	// Keep query caches hot for unrelated modules when edits are local-only.
	// We only bump global state revision when module signatures change.
	if len(s.lastInvalidation.SignatureChangedModules) > 0 {
		s.bumpRevision()
	}
	s.rebuildSnapshotLocked()

	if utils.IsFeatureEnabled("STATE_LOCK_TRACE") {
		s.logger.Info("state refresh",
			"doc", doc.URI,
			"first_lock_wait", firstLockWait.String(),
			"parse_lock_wait", parseLockWait.String(),
			"parse", parseDuration.String(),
			"commit_lock_wait", commitLockWait.String(),
			"total", time.Since(refreshStarted).String(),
		)
	}
}

func (s *ProjectState) DeleteDocument(docId string) {
	unlockDocument := s.LockDocument(docId)
	defer unlockDocument()

	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.documents.Delete(docId)
	s.symbolsTable.DeleteDocument(docId)
	s.fqnIndex.ClearByTag(docId)
	s.removeDependencyGraphForDocument(docId)
	s.bumpRevision()
	s.rebuildSnapshotLocked()
}

func (s *ProjectState) RenameDocument(oldDocId string, newDocId string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.fqnIndex.ClearByTag(oldDocId)
	s.documents.Rename(oldDocId, newDocId)
	s.symbolsTable.RenameDocument(oldDocId, newDocId)
	s.renameDependencyGraphDocument(oldDocId, newDocId)

	if unitModules := s.symbolsTable.GetByDoc(newDocId); unitModules != nil {
		s.indexParsedSymbols(*unitModules, newDocId)
	}
	s.bumpRevision()
	s.rebuildSnapshotLocked()
}

func (s *ProjectState) rebuildDependencyGraphForDocument(docId string, parsedModules symbols_table.UnitModules) {
	s.removeDependencyGraphForDocument(docId)

	moduleNames := make([]string, 0, len(parsedModules.Modules()))
	for _, module := range parsedModules.Modules() {
		moduleName := module.GetName()
		moduleNames = append(moduleNames, moduleName)

		if _, ok := s.moduleImports[moduleName]; !ok {
			s.moduleImports[moduleName] = make(map[string]struct{})
		}

		for _, imported := range module.Imports {
			s.moduleImports[moduleName][imported] = struct{}{}
			if _, ok := s.moduleDependents[imported]; !ok {
				s.moduleDependents[imported] = make(map[string]struct{})
			}
			s.moduleDependents[imported][moduleName] = struct{}{}
		}

		s.moduleSignatures[moduleName] = computeModuleSignature(module)
	}

	s.docModules[docId] = moduleNames
}

func (s *ProjectState) removeDependencyGraphForDocument(docId string) {
	oldModules, ok := s.docModules[docId]
	if !ok {
		return
	}

	for _, moduleName := range oldModules {
		for imported := range s.moduleImports[moduleName] {
			if dependents, ok := s.moduleDependents[imported]; ok {
				delete(dependents, moduleName)
				if len(dependents) == 0 {
					delete(s.moduleDependents, imported)
				}
			}
		}

		delete(s.moduleImports, moduleName)
		delete(s.moduleSignatures, moduleName)
	}

	delete(s.docModules, docId)
}

func (s *ProjectState) renameDependencyGraphDocument(oldDocId string, newDocId string) {
	modules, ok := s.docModules[oldDocId]
	if !ok {
		return
	}

	s.docModules[newDocId] = modules
	delete(s.docModules, oldDocId)
}

func (s *ProjectState) updateInvalidationScope(oldModules []string, oldSignatures map[string]uint64, parsedModules symbols_table.UnitModules) {
	changedModulesSet := make(map[string]struct{})
	for _, moduleName := range oldModules {
		changedModulesSet[moduleName] = struct{}{}
	}

	newSignatures := make(map[string]uint64)
	for _, module := range parsedModules.Modules() {
		moduleName := module.GetName()
		changedModulesSet[moduleName] = struct{}{}
		newSignatures[moduleName] = computeModuleSignature(module)
	}

	signatureChangedSet := make(map[string]struct{})
	for moduleName, newSignature := range newSignatures {
		oldSignature, existed := oldSignatures[moduleName]
		if !existed || oldSignature != newSignature {
			signatureChangedSet[moduleName] = struct{}{}
		}
	}
	for _, oldModule := range oldModules {
		if _, stillPresent := newSignatures[oldModule]; !stillPresent {
			signatureChangedSet[oldModule] = struct{}{}
		}
	}

	changedModules := mapKeys(changedModulesSet)
	signatureChangedModules := mapKeys(signatureChangedSet)

	impactedSet := make(map[string]struct{})
	for _, moduleName := range changedModules {
		impactedSet[moduleName] = struct{}{}
	}
	for _, moduleName := range signatureChangedModules {
		for _, impacted := range s.getImpactedModulesUnlocked([]string{moduleName}) {
			impactedSet[impacted] = struct{}{}
		}
	}

	s.lastInvalidation = InvalidationScope{
		ChangedModules:          changedModules,
		SignatureChangedModules: signatureChangedModules,
		ImpactedModules:         mapKeys(impactedSet),
	}
}

func mapKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}

	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

func (s *ProjectState) UpdateDocument(docURI protocol.DocumentUri, docVersion int32, changes []interface{}, parser *parser.Parser) {
	docId := utils.NormalizePath(docURI)
	s.updateDocumentByNormalizedID(docId, docVersion, changes, parser)
}

func (s *ProjectState) UpdateDocumentByNormalizedID(docID string, docVersion int32, changes []interface{}, parser *parser.Parser) {
	s.updateDocumentByNormalizedID(docID, docVersion, changes, parser)
}

func (s *ProjectState) updateDocumentByNormalizedID(docID string, docVersion int32, changes []interface{}, parser *parser.Parser) {
	s.stateMu.RLock()
	doc, ok := s.documents.GetNormalized(docID)
	s.stateMu.RUnlock()
	if !ok {
		return
	}
	if len(changes) == 0 {
		s.stateMu.Lock()
		currentDoc, currentOk := s.documents.GetNormalized(docID)
		if currentOk && docVersion > currentDoc.Version {
			currentDoc.Version = docVersion
		}
		s.stateMu.Unlock()
		return
	}
	if docVersion <= doc.Version {
		return
	}

	updatedDoc := *doc
	updatedDoc.ApplyChanges(changes)
	updatedDoc.Version = docVersion

	s.RefreshDocumentIdentifiers(&updatedDoc, parser)
}

func (s *ProjectState) CloseDocument(uri protocol.DocumentUri) {
	docId := utils.NormalizePath(uri)
	s.CloseDocumentByNormalizedID(docId)
}

func (s *ProjectState) CloseDocumentByNormalizedID(docID string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.documents.Close(docID)
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
		for _, fault := range module.FaultDefs {
			s.fqnIndex.Insert(fault)
		}
		for _, strukt := range module.Structs {
			s.fqnIndex.Insert(strukt)
		}
		for _, def := range module.Aliases {
			s.fqnIndex.Insert(def)
		}
		for _, distinct := range module.TypeDefs {
			s.fqnIndex.Insert(distinct)
		}
	}
}
