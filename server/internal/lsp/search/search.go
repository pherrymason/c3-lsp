package search

import (
	"fmt"
	"sort"
	"strings"
	"time"

	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/c3"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Search struct {
	debugEnabled   bool
	logger         commonlog.Logger
	declCache      *symbolLookupCache
	paramsCache    *searchParamsCache
	parentCache    *parentResolutionCache
	searchTimeout  time.Duration
	searchMaxDepth int
}

func NewSearch(logger commonlog.Logger, debugEnabled bool) Search {
	s := Search{
		debugEnabled: debugEnabled,
		logger:       logger,
		declCache:    newSymbolLookupCache(),
		paramsCache:  newSearchParamsCache(),
		parentCache:  newParentResolutionCache(),
	}
	// Read env vars once at construction time.
	s.searchTimeout = symbolSearchTimeout()
	s.searchMaxDepth = symbolSearchMaxDepth()
	return s
}

func (s *Search) debug(message string, debugger FindDebugger) {
	if !s.debugEnabled {
		return
	}

	maxo := utils.Min(debugger.depth, 20)
	prep := "|" + strings.Repeat(".", maxo)
	if debugger.depth > 8 {
		prep = fmt.Sprintf("%s (%d)", prep, debugger.depth)
	}

	s.logger.Debug("search debug", "depth", debugger.depth, "message", prep+" "+message)
}

func (s *Search) FindSymbolDeclarationInWorkspace(
	docId string,
	position symbols.Position,
	state *l.ProjectState,
) option.Option[symbols.Indexable] {

	doc := state.GetDocument(docId)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}

	cacheKey := buildSymbolLookupCacheKey(docId, doc, position, state.Revision())
	if cached, ok := s.declCache.get(cacheKey); ok {
		return cached
	}

	snapshot := state.Snapshot()
	if snapshot == nil {
		result := option.None[symbols.Indexable]()
		s.declCache.set(cacheKey, result)
		return result
	}

	unitModules, ok := snapshot.UnitModulesByDocValue(doc.URI)
	if !ok {
		result := option.None[symbols.Indexable]()
		s.declCache.set(cacheKey, result)
		return result
	}

	searchParamsKey := buildSearchParamsCacheKey(docId, doc, position, state.Revision())
	searchParams, ok := s.paramsCache.get(searchParamsKey)
	if !ok {
		searchParams = search_params.BuildSearchBySymbolUnderCursor(
			doc,
			unitModules,
			position,
		)
		s.paramsCache.set(searchParamsKey, searchParams)
	}
	debugger := FindDebugger{enabled: s.debugEnabled, depth: 0}
	if s.searchTimeout > 0 {
		debugger = debugger.withDeadline(time.Now().Add(s.searchTimeout))
	}

	searchResult := s.findClosestSymbolDeclaration(searchParams, state, debugger)
	if searchResult.IsSome() {
		s.declCache.set(cacheKey, searchResult.result)
		return searchResult.result
	}

	if builtinFallback := resolveBuiltinFaultFallback(searchParams.Symbol(), state); builtinFallback.IsSome() {
		s.declCache.set(cacheKey, builtinFallback)
		return builtinFallback
	}
	if qualifiedFaultFallback := s.resolveQualifiedFaultFallback(searchParams, state); qualifiedFaultFallback.IsSome() {
		s.declCache.set(cacheKey, qualifiedFaultFallback)
		return qualifiedFaultFallback
	}

	if !shouldRetryLookupAtPreviousPosition(searchParams.Symbol()) {
		result := option.None[symbols.Indexable]()
		s.declCache.set(cacheKey, result)
		return result
	}

	nextPositions := []symbols.Position{
		symbols.NewPosition(position.Line, position.Character+1),
		symbols.NewPosition(position.Line, position.Character+2),
	}
	previousPositions := []symbols.Position{}
	if position.Character > 0 {
		previousPositions = append(previousPositions, symbols.NewPosition(position.Line, position.Character-1))
	}
	if position.Character > 1 {
		previousPositions = append(previousPositions, symbols.NewPosition(position.Line, position.Character-2))
	}

	probePositions := append(nextPositions, previousPositions...)

	for _, lookupPosition := range probePositions {
		retryParams := search_params.BuildSearchBySymbolUnderCursor(doc, unitModules, lookupPosition)
		retryResult := s.findClosestSymbolDeclaration(retryParams, state, debugger)
		if retryResult.IsSome() {
			s.declCache.set(cacheKey, retryResult.result)
			return retryResult.result
		}

		if builtinFallback := resolveBuiltinFaultFallback(retryParams.Symbol(), state); builtinFallback.IsSome() {
			s.declCache.set(cacheKey, builtinFallback)
			return builtinFallback
		}
		if qualifiedFaultFallback := s.resolveQualifiedFaultFallback(retryParams, state); qualifiedFaultFallback.IsSome() {
			s.declCache.set(cacheKey, qualifiedFaultFallback)
			return qualifiedFaultFallback
		}
	}

	result := option.None[symbols.Indexable]()
	s.declCache.set(cacheKey, result)
	return result
}

func resolveBuiltinFaultFallback(symbolName string, state *l.ProjectState) option.Option[symbols.Indexable] {
	if state == nil {
		return option.None[symbols.Indexable]()
	}

	symbolName = normalizeIdentifierSymbol(symbolName)
	if symbolName == "" || strings.Contains(symbolName, "::") {
		return option.None[symbols.Indexable]()
	}

	for _, r := range symbolName {
		if !utils.IsAZ09_(r) && r != '_' {
			return option.None[symbols.Indexable]()
		}
	}

	builtinFQN := "std::core::builtin::" + symbolName
	for _, symbol := range state.SearchByFQN(builtinFQN) {
		if isFaultLikeSymbol(symbol) {
			return option.Some[symbols.Indexable](symbol)
		}
	}

	var found symbols.Indexable
	state.ForEachModuleUntil(func(module *symbols.Module) bool {
		if module.GetName() != "std::core::builtin" {
			return false
		}
		for _, fault := range module.FaultDefs {
			if fault == nil {
				continue
			}
			for _, constant := range fault.GetConstants() {
				if constant != nil && constant.GetName() == symbolName {
					found = constant
					return true
				}
			}
		}
		return false
	})
	if found != nil {
		return option.Some[symbols.Indexable](found)
	}

	return option.None[symbols.Indexable]()
}

func normalizeIdentifierSymbol(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	start := -1
	end := -1
	for i, r := range raw {
		if start < 0 {
			if utils.IsAZ09_(r) || r == '_' {
				start = i
				end = i
			}
			continue
		}

		if utils.IsAZ09_(r) || r == '_' {
			end = i
			continue
		}

		break
	}

	if start < 0 || end < start {
		return ""
	}

	return raw[start : end+1]
}

func isFaultLikeSymbol(symbol symbols.Indexable) bool {
	if symbol == nil {
		return false
	}
	if _, ok := symbol.(*symbols.FaultConstant); ok {
		return true
	}
	if _, ok := symbol.(*symbols.FaultDef); ok {
		return true
	}
	return false
}

func (s *Search) resolveQualifiedFaultFallback(searchParams search_params.SearchParams, state *l.ProjectState) option.Option[symbols.Indexable] {
	if state == nil || !searchParams.HasModuleSpecified() {
		return option.None[symbols.Indexable]()
	}

	faultName := normalizeIdentifierSymbol(searchParams.Symbol())
	if faultName == "" {
		return option.None[symbols.Indexable]()
	}

	moduleOpt := searchParams.LimitToModule()
	if moduleOpt.IsNone() {
		return option.None[symbols.Indexable]()
	}

	snapshot := state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}

	moduleCandidates := []symbols.ModulePath{moduleOpt.Get()}
	if resolved := s.resolveShortModulePath(moduleOpt.Get(), searchParams, state); resolved.IsSome() {
		if resolved.Get().GetName() != moduleOpt.Get().GetName() {
			moduleCandidates = append(moduleCandidates, resolved.Get())
		}
	}

	seenModules := map[string]struct{}{}
	for _, modulePath := range moduleCandidates {
		moduleName := modulePath.GetName()
		if moduleName == "" {
			continue
		}
		if _, ok := seenModules[moduleName]; ok {
			continue
		}
		seenModules[moduleName] = struct{}{}

		for _, module := range snapshot.ModulesByName(moduleName) {
			for _, fault := range module.FaultDefs {
				if fault == nil {
					continue
				}
				for _, constant := range fault.GetConstants() {
					if constant != nil && constant.GetName() == faultName {
						return option.Some[symbols.Indexable](constant)
					}
				}
			}
		}
	}

	return option.None[symbols.Indexable]()
}

func shouldRetryLookupAtPreviousPosition(symbol string) bool {
	if symbol == "" || c3.IsLanguageKeyword(symbol) {
		return false
	}

	for _, r := range symbol {
		if !utils.IsAZ09_(r) && r != '@' && r != '$' {
			return true
		}
	}

	return false
}

func (s *Search) FindHoverInformation(docURI string, params *protocol.HoverParams, state *l.ProjectState) option.Option[protocol.Hover] {
	doc := state.GetDocument(docURI)
	if doc == nil {
		return option.None[protocol.Hover]()
	}

	unitModules := state.GetUnitModulesByDoc(doc.URI)
	if unitModules == nil {
		return option.None[protocol.Hover]()
	}

	search := search_params.BuildSearchBySymbolUnderCursor(
		doc,
		*unitModules,
		symbols.NewPositionFromLSPPosition(params.Position),
	)

	if c3.IsLanguageKeyword(search.Symbol()) {
		return option.None[protocol.Hover]()
	}

	snapshot := state.Snapshot()
	if snapshot == nil {
		return option.None[protocol.Hover]()
	}

	foundSymbolOption := s.findClosestSymbolDeclaration(search, state, FindDebugger{depth: 0})
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

// Finds the closest selectedSymbol based on current scope.
// If not present in current Scope:
// - Search in files of same module
// - Search in imported files
// - SearchParams in global symbols in workspace
func (s *Search) findClosestSymbolDeclaration(searchParams search_params.SearchParams, state *l.ProjectState, debugger FindDebugger) SearchResult {
	searchResult := NewSearchResult(searchParams.TrackTraversedModules())
	if debugger.depth > s.searchMaxDepth {
		return searchResult
	}
	if state == nil {
		return searchResult
	}

	snapshot := state.Snapshot()
	if snapshot == nil {
		return searchResult
	}
	if debugger.timedOut() {
		return searchResult
	}
	if c3.IsLanguageKeyword(searchParams.Symbol()) {
		s.debug("Ignore because C3 keyword", debugger)
		return NewSearchResultEmpty(searchParams.TrackTraversedModules())
	}

	s.debug(fmt.Sprintf("findClosestSymbolDeclaration on doc %s: %s: %s", searchParams.DocId(), searchParams.ModuleInCursor(), searchParams.Symbol()), debugger)

	// Check if there's parent contextual information in searchParams
	if searchParams.HasAccessPath() {
		cacheKey := buildParentResolutionCacheKey(searchParams, state.Revision())
		if cached, ok := s.parentCache.get(cacheKey); ok {
			result := NewSearchResult(searchParams.TrackTraversedModules())
			if cached.IsSome() {
				result.Set(cached.Get())
			}
			return result
		}

		// Going from here, search should not limit to root search
		symbolResult := s.findInParentSymbols(searchParams, state, debugger)
		s.parentCache.set(cacheKey, symbolResult.result)
		//if symbolResult.IsSome() {
		return symbolResult
		//}
	}

	if searchParams.LimitsSearchInModule() {
		/*searchInSpecificModule := search_params.NewSearchParamsBuilder().
		WithSymbolWord(searchParams.SymbolW()).
		LimitedToModule(searchParams.LimitToModule()).
		WithoutDoc().
		Build()
		*/
		//searchResult := l.findSymbolDeclarationInModule(searchParams, searchParams.LimitToModule().Get(), debugger.goIn())
		moduleToSearch := searchParams.LimitToModule().Get()
		searchResult := s.strictFindSymbolDeclarationInModule(searchParams, moduleToSearch, state, debugger.goIn())
		if searchResult.IsSome() {
			return searchResult
		}

		resolvedModule := s.resolveShortModulePath(moduleToSearch, searchParams, state)
		if resolvedModule.IsSome() && resolvedModule.Get().GetName() != moduleToSearch.GetName() {
			searchResult = s.strictFindSymbolDeclarationInModule(searchParams, resolvedModule.Get(), state, debugger.goIn())
			if searchResult.IsSome() {
				return searchResult
			}
		}

		return searchResult
	}

	// Important, when depth ==0 we really need to transmit to only search into root from here, even if we go multiple levels deep.

	//docIdOption := searchParams.DocId()
	var collectionParsedModules []symbols_table.UnitModules
	allModulesByDoc := snapshot.AllUnitModulesView()
	if searchParams.LimitSearchToDoc() {
		docIdOption := searchParams.DocId()
		docId := docIdOption.Get()
		parsedModules, ok := snapshot.UnitModulesByDocValue(docId)
		if !ok {
			return searchResult
		}

		collectionParsedModules = append(collectionParsedModules, parsedModules)

		for oDocId, parsedModules := range allModulesByDoc {
			if docId == oDocId {
				continue
			}

			for _, scope := range parsedModules.Modules() {
				if scope.GetModule().GetName() == searchParams.ModulePathInCursor().GetName() {
					collectionParsedModules = append(collectionParsedModules, parsedModules)
				} else if scope.GetModule().IsImplicitlyImported(searchParams.ModulePathInCursor()) {
					collectionParsedModules = append(collectionParsedModules, parsedModules)
				}
			}
		}
	} else {
		searchInModule := searchParams.ModulePathInCursor()
		if searchParams.LimitsSearchInModule() {
			searchInModule = searchParams.LimitToModule().Get()
		}

		// Doc id not specified, search by module. Collect scope belonging to same module as searchParams.module
		for docId, parsedModules := range allModulesByDoc {
			if searchParams.ShouldExcludeDocId(docId) {
				continue
			}

			for _, scope := range parsedModules.Modules() {
				if scope.GetModule().IsImplicitlyImported(searchInModule) {
					collectionParsedModules = append(collectionParsedModules, parsedModules)
					break
				}
			}
		}
	}

	trackedModules := searchParams.TrackTraversedModules()
	contextModulePath := searchParams.ModulePathInCursor()
	for _, parsedModules := range collectionParsedModules {
		if debugger.timedOut() {
			return searchResult
		}

		for _, scopedTree := range parsedModules.GetLoadableModules(contextModulePath) {
			if debugger.timedOut() {
				return searchResult
			}

			scopeMode := searchParams.ScopeMode()
			if scopeMode == search_params.InScope {
				docOpt := searchParams.DocId()
				if docOpt.IsSome() {
					if docOpt.Get() != scopedTree.GetDocumentURI() {
						scopeMode = search_params.InModuleRoot
					}
				}
			}

			// Go through every element defined in scopedTree
			identifier, _ := findDeepFirst(
				searchParams.Symbol(),
				searchParams.SymbolPosition(),
				scopedTree,
				0,
				searchParams.IsLimitSearchInScope(),
				scopeMode,
			)

			if identifier != nil {
				searchResult.Set(identifier)
				return searchResult
			}

		}
	}
	/*
		if searchParams.ContinueOnModules() {
			// SEARCH ON SAME ContextModulePath, but other files
			// This search should not go through imported modules in those other files
			sb := search_params.NewSearchParamsBuilder().
				WithText(searchParams.SymbolW().Text(), searchParams.SymbolW().TextRange()).
				WithContextModule(searchParams.ContextModulePath()).
				WithoutDoc().
				WithExcludedDocs(searchParams.DocId()).
				WithoutContinueOnModules().
				WithScopeMode(search_params.InModuleRoot) // Document this
			searchInSameModule := sb.Build()

			found := l.findClosestSymbolDeclaration(searchInSameModule, debugger.goIn())
			if found.IsSome() {
				return found
			}
		}*/

	// SEARCH IN IMPORTED MODULES
	if searchParams.LimitSearchToDoc() {
		for _, parsedModules := range collectionParsedModules {
			if debugger.timedOut() {
				return searchResult
			}

			for _, mod := range parsedModules.Modules() {
				if debugger.timedOut() {
					return searchResult
				}

				importCandidates := s.expandImportedModuleCandidates(mod, snapshot)
				for i := 0; i < len(importCandidates); i++ {
					if debugger.timedOut() {
						return searchResult
					}

					module := importCandidates[i]
					searchResult.TrackTraversedModule(module)
					if !searchParams.TrackTraversedModule(module) {
						continue
					}
					sp := search_params.NewSearchParamsBuilder().
						WithSymbolWord(searchParams.SymbolW()).
						LimitedToModulePath(symbols.NewModulePathFromString(module)).
						WithTrackedModules(trackedModules).
						WithScopeMode(search_params.InModuleRoot). // Document this
						Build()

					s.debug(fmt.Sprintf("findClosestSymbolDeclaration: search in imported module \"%s\": %s", module, searchParams.Symbol()), debugger)
					symbol := s.findSymbolDeclarationInModule(sp, symbols.NewModulePathFromString(module), state, debugger.goIn())
					if symbol.IsSome() {
						return symbol
					}
				}
			}
		}
	}

	// Last resort, check if any loadable module is compatible with the string being searched
	if /*debugger.depth == 0 &&*/ searchResult.IsNone() {
		moduleMatches := s.findModuleNameInTraversedModules(searchParams, searchResult.traversedModules, state)

		if len(moduleMatches) > 0 {
			searchResult.Set(moduleMatches[0])
		}
	}

	// Not found...
	return searchResult
}

// Search symbols inside a given module
func (l *Search) findSymbolDeclarationInModule(searchParams search_params.SearchParams, moduleToSearch symbols.ModulePath, projState *l.ProjectState, debugger FindDebugger) SearchResult {
	searchResult := NewSearchResult(searchParams.TrackTraversedModules())
	if projState == nil {
		return searchResult
	}

	snapshot := projState.Snapshot()
	if snapshot == nil {
		return searchResult
	}

	for docId, modulesByDoc := range snapshot.AllUnitModulesView() {
		if debugger.timedOut() {
			return searchResult
		}

		for _, scope := range modulesByDoc.GetLoadableModules(moduleToSearch) {
			if debugger.timedOut() {
				return searchResult
			}

			searchResult.TrackTraversedModule(scope.GetModuleString())

			if !searchParams.TrackTraversedModule(scope.GetModuleString()) {
				continue
			}
			l.debug(fmt.Sprintf("findSymbolDeclarationInModule: search symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)

			sp := search_params.NewSearchParamsBuilder().
				WithSymbolWord(searchParams.SymbolW()).
				WithDocId(docId).
				WithScopeMode(search_params.InModuleRoot).
				WithTrackedModules(searchParams.TrackedModules()).
				Build()

			symbolResult := l.findClosestSymbolDeclaration(
				sp, projState, FindDebugger{depth: debugger.depth + 1, deadline: debugger.deadline})
			l.debug(fmt.Sprintf("end searching symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)
			if symbolResult.IsSome() {
				return symbolResult
			}
		}
	}

	return searchResult
}

func (l *Search) strictFindSymbolDeclarationInModule(searchParams search_params.SearchParams, moduleToSearch symbols.ModulePath, projState *l.ProjectState, debugger FindDebugger) SearchResult {
	searchResult := NewSearchResult(searchParams.TrackTraversedModules())
	if projState == nil {
		return searchResult
	}

	snapshot := projState.Snapshot()
	if snapshot == nil {
		return searchResult
	}
	results := []struct {
		identifier symbols.Indexable
		score      int
	}{}

	for docId, modulesByDoc := range snapshot.AllUnitModulesView() {
		if debugger.timedOut() {
			return searchResult
		}

		for _, scope := range modulesByDoc.GetLoadableModules(moduleToSearch) {
			if debugger.timedOut() {
				return searchResult
			}

			searchResult.TrackTraversedModule(scope.GetModuleString())
			if !searchParams.TrackTraversedModule(scope.GetModuleString()) {
				continue
			}
			l.debug(fmt.Sprintf("strictFindSymbolDeclarationInModule: search symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)

			identifier, _ := findDeepFirst(
				searchParams.Symbol(),
				searchParams.SymbolPosition(),
				scope,
				0,
				searchParams.IsLimitSearchInScope(),
				search_params.InModuleRoot,
			)

			if identifier != nil {
				score := 2
				if moduleToSearch.GetName() == scope.GetModule().GetName() {
					score = 100
				}

				results = append(results, struct {
					identifier symbols.Indexable
					score      int
				}{identifier: identifier, score: score})
				//searchResult.Set(identifier)
				//return searchResult
			}
		}
	}

	if len(results) == 0 {
		if searchResult.IsNone() {
			moduleMatches := l.findModuleNameInTraversedModules(searchParams, searchResult.traversedModules, projState)

			if len(moduleMatches) > 0 {
				searchResult.Set(moduleMatches[0])
			}
		}

		return searchResult
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	searchResult.Set(results[0].identifier)
	return searchResult
}

func (l *Search) findModuleNameInTraversedModules(searchParams search_params.SearchParams, traversedModules map[string]bool, projState *l.ProjectState) []*symbols.Module {
	matches := []*symbols.Module{}

	// full module name
	if searchParams.HasAccessPath() {
		return matches
	}

	moduleName := searchParams.GetFullQualifiedName()

	if projState == nil {
		return matches
	}
	snapshot := projState.Snapshot()
	if snapshot == nil {
		return matches
	}

	if _, ok := traversedModules[moduleName]; !ok {
		return matches
	}

	matches = append(matches, snapshot.ModulesByName(moduleName)...)

	return matches
}

func (s *Search) resolveShortModulePath(moduleToSearch symbols.ModulePath, searchParams search_params.SearchParams, state *l.ProjectState) option.Option[symbols.ModulePath] {
	shortName := moduleToSearch.GetName()
	if shortName == "" || strings.Contains(shortName, "::") {
		return option.None[symbols.ModulePath]()
	}
	if state == nil {
		return option.None[symbols.ModulePath]()
	}
	snapshot := state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.ModulePath]()
	}

	imports := map[string]bool{}
	recursiveImports := map[string]bool{}
	docIdOpt := searchParams.DocId()
	if docIdOpt.IsSome() {
		docId := docIdOpt.Get()
		if modulesByDoc, ok := snapshot.UnitModulesByDocValue(docId); ok {
			contextModule := searchParams.ModulePathInCursor().GetName()
			for _, mod := range modulesByDoc.Modules() {
				if mod.GetName() != contextModule {
					continue
				}
				for _, imp := range mod.Imports {
					imports[imp] = true
					if !mod.IsImportNoRecurse(imp) {
						recursiveImports[imp] = true
					}
				}
			}
		}
	}

	type candidate struct {
		path  symbols.ModulePath
		score int
	}

	candidates := snapshot.ModuleNamesByShort(shortName)
	if len(candidates) == 0 {
		return option.None[symbols.ModulePath]()
	}

	best := option.None[candidate]()
	for _, name := range candidates {
		score := 0
		switch {
		case name == shortName:
			score = 1000
		case strings.HasSuffix(name, "::"+shortName):
			score = 100
		default:
			continue
		}

		if name == "std::core::"+shortName {
			score += 500
		}

		if imports[name] {
			score += 300
		}
		for imp := range recursiveImports {
			if strings.HasPrefix(name, imp+"::") {
				score += 200
			}
		}

		cand := candidate{path: symbols.NewModulePathFromString(name), score: score}
		if best.IsNone() {
			best = option.Some(cand)
			continue
		}

		current := best.Get()
		if cand.score > current.score || (cand.score == current.score && cand.path.GetName() < current.path.GetName()) {
			best = option.Some(cand)
		}
	}

	if best.IsNone() {
		return option.None[symbols.ModulePath]()
	}

	return option.Some(best.Get().path)
}

func (s *Search) expandImportedModuleCandidates(module *symbols.Module, snapshot *l.ProjectSnapshot) []string {
	if module == nil {
		return nil
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(module.Imports))

	for _, imported := range module.Imports {
		if imported == "" {
			continue
		}
		if _, ok := seen[imported]; !ok {
			seen[imported] = struct{}{}
			result = append(result, imported)
		}

		if module.IsImportNoRecurse(imported) || snapshot == nil {
			continue
		}

		prefix := imported + "::"
		snapshot.ForEachModule(func(scope *symbols.Module) {
			name := scope.GetName()
			if !strings.HasPrefix(name, prefix) {
				return
			}
			if _, ok := seen[name]; ok {
				return
			}
			seen[name] = struct{}{}
			result = append(result, name)
		})
	}

	return result
}

func (s *Search) implicitImportedParsedModules(
	state *l.ProjectState,
	acceptedModulePaths []symbols.ModulePath,
	excludeDocId option.Option[string],

) []*symbols.Module {
	var collectionModules []*symbols.Module
	if state == nil {
		return collectionModules
	}

	snapshot := state.Snapshot()
	if snapshot == nil {
		return collectionModules
	}

	scopeIdx := snapshot.ScopeIndex()

	// Fast path: use the pre-computed index when available.
	// For each accepted module, the index gives us exactly which module names
	// are reachable — no O(D×M) document scan required.
	if scopeIdx != nil {
		seen := make(map[*symbols.Module]struct{})
		indexMiss := false

		for _, acceptedModule := range acceptedModulePaths {
			names := scopeIdx.ModuleNames(acceptedModule.GetName())
			if names == nil {
				// Module not in index (e.g. snapshot is stale) — use slow path.
				indexMiss = true
				break
			}
			for _, name := range names {
				for _, mod := range snapshot.ModulesByName(name) {
					if excludeDocId.IsSome() && excludeDocId.Get() == mod.GetDocumentURI() {
						continue
					}
					if _, dup := seen[mod]; !dup {
						seen[mod] = struct{}{}
						collectionModules = append(collectionModules, mod)
					}
				}
			}
		}

		if !indexMiss {
			return collectionModules
		}

		// Reset and fall through to the linear scan.
		collectionModules = collectionModules[:0]
	}

	// Slow path: O(D×M×A) linear scan — used only when the index is absent.
	for docId, parsedModules := range snapshot.AllUnitModulesView() {
		if excludeDocId.IsSome() && excludeDocId.Get() == docId {
			continue
		}

		for _, scope := range parsedModules.Modules() {
			for _, acceptedModule := range acceptedModulePaths {
				if scope.GetModule().IsImplicitlyImported(acceptedModule) {
					collectionModules = append(collectionModules, scope)
					break
				}
			}
		}
	}

	return collectionModules
}
