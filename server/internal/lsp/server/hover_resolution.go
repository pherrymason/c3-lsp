package server

import (
	stdctx "context"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/c3"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) resolveImportedMethodHoverFallback(requestCtx stdctx.Context, docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}

	unit := h.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return option.None[symbols.Indexable]()
	}

	searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
	if !searchParams.HasAccessPath() {
		return option.None[symbols.Indexable]()
	}

	methodName := searchParams.Symbol()
	if methodName == "" {
		return option.None[symbols.Indexable]()
	}

	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}

	receiverType := h.hoverReceiverTypeHint(requestCtx, docID, searchParams)
	if receiverType == "" {
		receiverType = h.inferLambdaReceiverTypeFromContext(doc, pos, searchParams, snapshot, *unit)
	}

	candidateModules := h.collectImportedCandidateModules(snapshot, *unit, searchParams.ModulePathInCursor().GetName())

	candidates := []*symbols.Function{}
	fallbackCandidates := []*symbols.Function{}
	for moduleName := range candidateModules {
		for _, module := range snapshot.ModulesByName(moduleName) {
			for _, fn := range module.ChildrenFunctions {
				if fn == nil || fn.GetMethodName() != methodName {
					continue
				}
				fallbackCandidates = append(fallbackCandidates, fn)
				if receiverType != "" && fn.GetTypeIdentifier() == receiverType {
					candidates = append(candidates, fn)
				}
			}
		}
	}

	if len(candidates) == 0 {
		candidates = fallbackCandidates
	}
	if len(candidates) == 0 && receiverType != "" {
		snapshot.ForEachModule(func(module *symbols.Module) {
			for _, fn := range module.ChildrenFunctions {
				if fn == nil || fn.GetMethodName() != methodName {
					continue
				}
				if fn.GetTypeIdentifier() == receiverType {
					candidates = append(candidates, fn)
				}
			}
		})
	}

	if len(candidates) == 0 {
		return option.None[symbols.Indexable]()
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].GetFQN() < candidates[j].GetFQN()
	})

	return option.Some[symbols.Indexable](candidates[0])
}

func (h *Server) resolveQualifiedSymbolHoverFallback(docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}

	unit := h.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return option.None[symbols.Indexable]()
	}

	searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
	if !searchParams.LimitsSearchInModule() {
		return option.None[symbols.Indexable]()
	}

	symbolName := searchParams.Symbol()
	if symbolName == "" {
		return option.None[symbols.Indexable]()
	}

	modulePath := searchParams.LimitToModule().Get().GetName()
	if modulePath == "" {
		return option.None[symbols.Indexable]()
	}

	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}

	cursorModule := searchParams.ModulePathInCursor().GetName()
	resolvedModules := h.resolveModuleCandidatesWithFallback(snapshot, *unit, cursorModule, modulePath, docID, pos)

	if result := collectFirstMatchInModules(snapshot, resolvedModules, symbolName); result.IsSome() {
		return result
	}

	// Retry after attempting to load likely module files.
	h.tryLoadLikelyModuleFiles(docID, pos, modulePath)
	snapshot = h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}
	resolvedModules = h.resolveModuleCandidatesWithFallback(snapshot, *unit, cursorModule, modulePath, docID, pos)
	return collectFirstMatchInModules(snapshot, resolvedModules, symbolName)
}

func (h *Server) resolveModuleSeparatorSymbolHoverFallback(requestCtx stdctx.Context, docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}

	idx := pos.IndexIn(doc.SourceCode.Text)
	if idx < 0 || idx >= len(doc.SourceCode.Text) {
		return option.None[symbols.Indexable]()
	}
	s := doc.SourceCode.Text
	if !isCursorOnModuleSeparator(s, idx) {
		return option.None[symbols.Indexable]()
	}

	lineContent, lineStart, ok := lineBoundsAt(s, idx)
	if !ok {
		return option.None[symbols.Indexable]()
	}

	local := idx - lineStart
	sep := strings.Index(lineContent[local:], "::")
	if sep != 0 {
		// Cursor may be on second ':'
		if local > 0 && strings.HasPrefix(lineContent[local-1:], "::") {
			local--
		} else {
			return option.None[symbols.Indexable]()
		}
	}

	searchStart := local + 2
	for searchStart < len(lineContent) && !isIdentByte(lineContent[searchStart]) {
		searchStart++
	}
	if searchStart >= len(lineContent) {
		return option.None[symbols.Indexable]()
	}

	candidate := symbols.NewPosition(pos.Line, uint(searchStart))
	resolved := h.findSymbolDeclarationWithContext(requestCtx, docID, candidate)
	if resolved.IsSome() {
		return resolved
	}

	if fallback := h.resolveQualifiedCallHoverFallback(docID, candidate); fallback.IsSome() {
		return fallback
	}
	if fallback := h.resolveQualifiedSymbolHoverFallback(docID, candidate); fallback.IsSome() {
		return fallback
	}

	return option.None[symbols.Indexable]()
}

func (h *Server) resolveQualifiedCallHoverFallback(docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}

	modulePath, symbolName, ok := extractQualifiedSymbolAt(doc.SourceCode.Text, pos.IndexIn(doc.SourceCode.Text))
	if !ok || modulePath == "" || symbolName == "" {
		return option.None[symbols.Indexable]()
	}

	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}
	unit := h.state.GetUnitModulesByDoc(docID)
	var resolvedModules []string
	if unit != nil {
		searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
		cursorModule := searchParams.ModulePathInCursor().GetName()
		resolvedModules = h.resolveModuleCandidatesWithFallback(snapshot, *unit, cursorModule, modulePath, docID, pos)
	} else {
		resolvedModules = h.resolveModuleCandidatesFromSnapshotOnly(snapshot, doc.SourceCode.Text, modulePath, docID, pos)
	}

	if result := collectFirstMatchInModules(snapshot, resolvedModules, symbolName); result.IsSome() {
		return result
	}

	// Retry after preloading imported roots and likely module files.
	uri := h.documentURIFromDocID(docID)
	h.preloadImportedRootModulesForURIForce(uri)
	for _, importRoot := range importRootsFromSource(doc.SourceCode.Text) {
		h.indexImportRootCandidates(uri, importRoot)
	}
	h.tryLoadLikelyModuleFiles(docID, pos, modulePath)

	snapshot = h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}
	if unit != nil {
		searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
		cursorModule := searchParams.ModulePathInCursor().GetName()
		resolvedModules = h.resolveModuleCandidatesWithFallback(snapshot, *unit, cursorModule, modulePath, docID, pos)
	} else {
		resolvedModules = h.resolveModuleCandidatesFromSnapshotOnly(snapshot, doc.SourceCode.Text, modulePath, docID, pos)
	}
	return collectFirstMatchInModules(snapshot, resolvedModules, symbolName)
}

func (h *Server) resolveModuleTokenHoverFallback(docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}

	cursorIndex := pos.IndexIn(doc.SourceCode.Text)
	token, ok := extractModuleTokenAt(doc.SourceCode.Text, cursorIndex)
	if !ok || token == "" {
		return option.None[symbols.Indexable]()
	}

	unit := h.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return option.None[symbols.Indexable]()
	}
	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}

	searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
	candidateModules := h.collectImportedCandidateModules(snapshot, *unit, searchParams.ModulePathInCursor().GetName())
	allModules := allModuleNames(snapshot)
	if len(candidateModules) == 0 {
		candidateModules = allModules
	}

	type moduleMatch struct {
		module *symbols.Module
		score  int
	}
	matches := []moduleMatch{}
	for name := range candidateModules {
		score := 0
		switch {
		case name == token:
			score = 300
		case strings.HasSuffix(name, "::"+token):
			score = 200
		case strings.HasPrefix(name, token+"::"):
			score = 150
		default:
			continue
		}
		for _, module := range snapshot.ModulesByName(name) {
			if module != nil {
				matches = append(matches, moduleMatch{module: module, score: score})
			}
		}
	}

	if len(matches) == 0 {
		if h.indexingHover(protocol.DocumentUri(docID)) != nil {
			return option.None[symbols.Indexable]()
		}
		uri := h.documentURIFromDocID(docID)
		h.preloadImportedRootModulesForURIForce(uri)
		for _, importRoot := range importRootsFromSource(doc.SourceCode.Text) {
			h.indexImportRootCandidates(uri, importRoot)
		}
		snapshot = h.state.Snapshot()
		if snapshot == nil {
			return option.None[symbols.Indexable]()
		}
		allModules = allModuleNames(snapshot)
		candidateModules = h.collectImportedCandidateModules(snapshot, *unit, searchParams.ModulePathInCursor().GetName())
		if len(candidateModules) == 0 {
			candidateModules = allModules
		}
		for name := range candidateModules {
			score := 0
			switch {
			case name == token:
				score = 300
			case strings.HasSuffix(name, "::"+token):
				score = 200
			case strings.HasPrefix(name, token+"::"):
				score = 150
			default:
				continue
			}
			for _, module := range snapshot.ModulesByName(name) {
				if module != nil {
					matches = append(matches, moduleMatch{module: module, score: score})
				}
			}
		}
	}

	if len(matches) == 0 {
		h.tryLoadLikelyModuleFiles(docID, pos, token)
		snapshot = h.state.Snapshot()
		if snapshot == nil {
			return option.None[symbols.Indexable]()
		}
		allModules = allModuleNames(snapshot)
		candidateModules = h.collectImportedCandidateModules(snapshot, *unit, searchParams.ModulePathInCursor().GetName())
		if len(candidateModules) == 0 {
			candidateModules = allModules
		}
		for name := range candidateModules {
			score := 0
			switch {
			case name == token:
				score = 300
			case strings.HasSuffix(name, "::"+token):
				score = 200
			case strings.HasPrefix(name, token+"::"):
				score = 150
			default:
				continue
			}
			for _, module := range snapshot.ModulesByName(name) {
				if module != nil {
					matches = append(matches, moduleMatch{module: module, score: score})
				}
			}
		}
	}

	if len(matches) == 0 {
		for _, candidate := range h.likelyModulePathCandidates(docID, pos, token) {
			for _, module := range snapshot.ModulesByName(candidate) {
				if module != nil {
					matches = append(matches, moduleMatch{module: module, score: 300})
				}
			}
		}
	}

	if len(matches) == 0 {
		for name := range allModules {
			score := 0
			switch {
			case name == token:
				score = 300
			case strings.HasSuffix(name, "::"+token):
				score = 200
			case strings.HasPrefix(name, token+"::"):
				score = 150
			default:
				continue
			}
			for _, module := range snapshot.ModulesByName(name) {
				if module != nil {
					matches = append(matches, moduleMatch{module: module, score: score})
				}
			}
		}
	}

	if len(matches) == 0 {
		return option.None[symbols.Indexable]()
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].module.GetName() < matches[j].module.GetName()
	})

	return option.Some[symbols.Indexable](matches[0].module)
}

func (h *Server) resolveQualifiedModuleCandidates(snapshot *project_state.ProjectSnapshot, unit symbols_table.UnitModules, contextModuleName string, modulePath string) []string {
	if snapshot == nil || modulePath == "" {
		return nil
	}
	if len(snapshot.ModulesByName(modulePath)) > 0 {
		return []string{modulePath}
	}

	candidates := h.collectImportedCandidateModules(snapshot, unit, contextModuleName)
	allModules := allModuleNames(snapshot)
	if len(candidates) == 0 {
		candidates = allModules
	}

	resolved := collectMatchingModuleNames(modulePath, candidates)
	if len(resolved) == 0 {
		resolved = collectMatchingModuleNames(modulePath, allModules)
	}

	return resolved
}

func (h *Server) likelyModulePathCandidates(docID string, pos symbols.Position, modulePath string) []string {
	if modulePath == "" {
		return nil
	}
	if strings.Contains(modulePath, "::") {
		return []string{modulePath}
	}

	seen := map[string]struct{}{}
	out := []string{}
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}

	doc := h.state.GetDocument(docID)
	unit := h.state.GetUnitModulesByDoc(docID)
	if doc != nil && unit != nil {
		sp := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
		ctxModuleName := sp.ModulePathInCursor().GetName()
		if ctxModuleName != "" {
			parts := strings.Split(ctxModuleName, "::")
			start := len(parts) - 1
			if start < 1 {
				start = 1
			}
			for i := start; i >= 1; i-- {
				add(strings.Join(parts[:i], "::") + "::" + modulePath)
			}
		}

		for _, module := range unit.Modules() {
			if module == nil {
				continue
			}
			if ctxModuleName != "" && module.GetName() != ctxModuleName {
				continue
			}
			for _, imp := range module.Imports {
				if imp == "" {
					continue
				}
				if imp == modulePath || strings.HasSuffix(imp, "::"+modulePath) {
					add(imp)
				}
				add(imp + "::" + modulePath)
			}
		}
	}

	add(modulePath)
	return out
}

func (h *Server) tryLoadLikelyModuleFiles(docID string, pos symbols.Position, modulePath string) {
	if modulePath == "" {
		return
	}

	docURI := h.documentURIFromDocID(docID)
	root := h.resolveProjectRootForURI(&docURI)
	root = fs.GetCanonicalPath(root)
	if root == "" {
		return
	}

	searchRoots := []string{root}
	searchRoots = append(searchRoots, h.workspaceDependencyDirs...)

	short := modulePath
	if strings.Contains(short, "::") {
		parts := strings.Split(short, "::")
		short = parts[len(parts)-1]
	}

	moduleCandidates := h.likelyModulePathCandidates(docID, pos, modulePath)
	for _, moduleCandidate := range moduleCandidates {
		if strings.Contains(moduleCandidate, "::") {
			parts := strings.Split(moduleCandidate, "::")
			short = parts[len(parts)-1]
			break
		}
	}

	paths := []string{}
	for _, sr := range searchRoots {
		if sr == "" {
			continue
		}
		paths = append(paths,
			filepath.Join(sr, "src", short+".c3"),
			filepath.Join(sr, "src", short+".c3i"),
			filepath.Join(sr, short+".c3l"),
			filepath.Join(sr, short+".c3"),
			filepath.Join(sr, short+".c3i"),
		)
	}

	loadedAny := false
	for _, p := range paths {
		canonical := fs.GetCanonicalPath(p)
		if canonical == "" {
			continue
		}
		if h.state.GetDocument(canonical) != nil {
			continue
		}
		if h.loadAndIndexFile(canonical) {
			loadedAny = true
		}
	}

	if loadedAny {
		return
	}

	acceptModule := func(moduleName string) bool {
		return moduleName == modulePath || strings.HasSuffix(moduleName, "::"+short)
	}
	seen := map[string]struct{}{}
	for _, sr := range searchRoots {
		if sr == "" {
			continue
		}
		files, _, err := fs.ScanForC3WithOptions(sr, fs.ScanOptions{IgnoreDirs: fs.DefaultC3ScanIgnoreDirs()})
		if err != nil {
			continue
		}
		for _, file := range files {
			canonical := fs.GetCanonicalPath(file)
			if canonical == "" {
				continue
			}
			if _, ok := seen[canonical]; ok {
				continue
			}
			seen[canonical] = struct{}{}
			h.loadFilterAndIndex(canonical, acceptModule)
		}
	}
}

func (h *Server) documentURIFromDocID(docID string) protocol.DocumentUri {
	if docID == "" {
		return ""
	}
	if doc := h.state.GetDocument(docID); doc != nil && strings.TrimSpace(doc.URI) != "" {
		return protocol.DocumentUri(doc.URI)
	}
	return protocol.DocumentUri(fs.ConvertPathToURI(docID, option.None[string]()))
}

func collectMatchingModuleNames(modulePath string, candidateModules map[string]struct{}) []string {
	if modulePath == "" {
		return nil
	}

	matches := []string{}
	for name := range candidateModules {
		if name == modulePath || strings.HasSuffix(name, "::"+modulePath) {
			matches = append(matches, name)
		}
	}

	sort.Strings(matches)
	return matches
}

func allModuleNames(snapshot *project_state.ProjectSnapshot) map[string]struct{} {
	out := map[string]struct{}{}
	if snapshot == nil {
		return out
	}
	snapshot.ForEachModule(func(module *symbols.Module) {
		out[module.GetName()] = struct{}{}
	})
	return out
}

func (h *Server) resolveImportedSymbolHoverFallback(docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}

	unit := h.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return option.None[symbols.Indexable]()
	}

	searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
	if searchParams.Symbol() == "" {
		return option.None[symbols.Indexable]()
	}

	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}

	candidateModules := h.collectImportedCandidateModules(snapshot, *unit, searchParams.ModulePathInCursor().GetName())
	if len(candidateModules) == 0 {
		return option.None[symbols.Indexable]()
	}

	name := searchParams.Symbol()
	if searchParams.HasAccessPath() {
		access := searchParams.GetFullAccessPath()
		if len(access) == 0 {
			return option.None[symbols.Indexable]()
		}
		root := access[0].Text()
		if root == "" {
			return option.None[symbols.Indexable]()
		}
		r := rune(root[0])
		if !unicode.IsUpper(r) {
			return option.None[symbols.Indexable]()
		}
		name = root
	}
	var matches []symbols.Indexable
	for moduleName := range candidateModules {
		for _, module := range snapshot.ModulesByName(moduleName) {
			matches = append(matches, collectDirectMatchesInModule(module, name)...)
			for _, fault := range module.FaultDefs {
				if fault == nil {
					continue
				}
				for _, c := range fault.GetConstants() {
					if c != nil && c.GetName() == name {
						matches = append(matches, c)
					}
				}
			}
		}
	}

	if len(matches) == 0 {
		return option.None[symbols.Indexable]()
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].GetFQN() < matches[j].GetFQN()
	})

	return option.Some(matches[0])
}

func (h *Server) resolveIdentifierTokenHoverFallback(docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}

	idx := pos.IndexIn(doc.SourceCode.Text)
	ident, ok := extractIdentifierTokenAt(doc.SourceCode.Text, idx)
	if !ok || ident == "" {
		return option.None[symbols.Indexable]()
	}
	if c3.IsLanguageKeyword(ident) {
		return option.None[symbols.Indexable]()
	}

	unit := h.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return option.None[symbols.Indexable]()
	}
	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}

	searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
	candidateModules := h.collectImportedCandidateModules(snapshot, *unit, searchParams.ModulePathInCursor().GetName())
	if len(candidateModules) == 0 {
		candidateModules = allModuleNames(snapshot)
	}

	matches := []symbols.Indexable{}
	for moduleName := range candidateModules {
		for _, module := range snapshot.ModulesByName(moduleName) {
			matches = append(matches, collectDirectMatchesInModule(module, ident)...)
		}
	}
	if len(matches) == 0 {
		return option.None[symbols.Indexable]()
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].GetFQN() < matches[j].GetFQN()
	})

	return option.Some(matches[0])
}

func (h *Server) collectImportedCandidateModules(snapshot *project_state.ProjectSnapshot, modulesByDoc symbols_table.UnitModules, contextModuleName string) map[string]struct{} {
	candidateModules := map[string]struct{}{}
	h.addContextModuleCandidates(candidateModules, contextModuleName)
	for _, module := range modulesByDoc.Modules() {
		if module == nil {
			continue
		}
		if contextModuleName != "" && module.GetName() != contextModuleName {
			continue
		}
		h.addContextModuleCandidates(candidateModules, module.GetName())
		for _, imported := range module.Imports {
			if imported == "" {
				continue
			}
			candidateModules[imported] = struct{}{}
			if module.IsImportNoRecurse(imported) {
				continue
			}
			prefix := imported + "::"
			snapshot.ForEachModule(func(scope *symbols.Module) {
				name := scope.GetName()
				if strings.HasPrefix(name, prefix) {
					candidateModules[name] = struct{}{}
				}
			})
		}
	}

	if len(candidateModules) == 0 {
		h.addContextModuleCandidates(candidateModules, contextModuleName)
		for _, module := range modulesByDoc.Modules() {
			if module == nil {
				continue
			}
			h.addContextModuleCandidates(candidateModules, module.GetName())
			for _, imported := range module.Imports {
				candidateModules[imported] = struct{}{}
			}
		}
	}

	return candidateModules
}

func collectDirectMatchesInModule(module *symbols.Module, symbolName string) []symbols.Indexable {
	if module == nil || symbolName == "" {
		return nil
	}

	matches := []symbols.Indexable{}
	if v, ok := module.Variables[symbolName]; ok {
		matches = append(matches, v)
	}
	if st, ok := module.Structs[symbolName]; ok {
		matches = append(matches, st)
	}
	if en, ok := module.Enums[symbolName]; ok {
		matches = append(matches, en)
	}
	for _, enum := range module.Enums {
		if enum == nil {
			continue
		}
		for _, enumerator := range enum.GetEnumerators() {
			if enumerator != nil && enumerator.GetName() == symbolName {
				matches = append(matches, enumerator)
			}
		}
	}
	if d, ok := module.Aliases[symbolName]; ok {
		matches = append(matches, d)
	}
	if di, ok := module.TypeDefs[symbolName]; ok {
		matches = append(matches, di)
	}
	if bit, ok := module.Bitstructs[symbolName]; ok {
		matches = append(matches, bit)
	}
	for _, fn := range module.ChildrenFunctions {
		if fn != nil && fn.GetName() == symbolName {
			matches = append(matches, fn)
		}
	}

	return matches
}

func (h *Server) addContextModuleCandidates(candidateModules map[string]struct{}, moduleName string) {
	if moduleName == "" {
		return
	}
	parts := strings.Split(moduleName, "::")
	for i := len(parts); i >= 1; i-- {
		candidateModules[strings.Join(parts[:i], "::")] = struct{}{}
	}
}

func isCursorOnModuleSeparator(source string, idx int) bool {
	if idx < 0 || idx >= len(source) {
		return false
	}
	if source[idx] == ':' {
		if idx+1 < len(source) && source[idx+1] == ':' {
			return true
		}
		if idx > 0 && source[idx-1] == ':' {
			return true
		}
	}
	return false
}

func lineBoundsAt(source string, idx int) (string, int, bool) {
	if idx < 0 || idx >= len(source) {
		return "", 0, false
	}
	start := idx
	for start > 0 && source[start-1] != '\n' {
		start--
	}
	end := idx
	for end < len(source) && source[end] != '\n' {
		end++
	}
	return source[start:end], start, true
}

// resolveSymbolCommonFallbacks runs the three fallback resolvers that are
// shared between the hover and definition handlers:
//  1. resolveQualifiedSymbolHoverFallback — handles "Module::Symbol" tokens
//  2. resolveImportedMethodHoverFallback  — handles method calls on imported types
//  3. resolveImportedSymbolHoverFallback  — handles symbols from imported modules
//
// It returns the first non-None result, or None if all fallbacks fail.
// resolveModuleCandidatesWithFallback tries to resolve the target module using
// qualified module resolution, then falls back to likely module-path heuristics,
// and finally uses the raw modulePath as-is.
func (h *Server) resolveModuleCandidatesWithFallback(snapshot *project_state.ProjectSnapshot, unit symbols_table.UnitModules, cursorModule string, modulePath string, docID string, pos symbols.Position) []string {
	resolved := h.resolveQualifiedModuleCandidates(snapshot, unit, cursorModule, modulePath)
	if len(resolved) > 0 {
		return resolved
	}
	for _, candidate := range h.likelyModulePathCandidates(docID, pos, modulePath) {
		if len(snapshot.ModulesByName(candidate)) > 0 {
			resolved = append(resolved, candidate)
		}
	}
	if len(resolved) > 0 {
		return resolved
	}
	return []string{modulePath}
}

func (h *Server) resolveModuleCandidatesFromSnapshotOnly(snapshot *project_state.ProjectSnapshot, source string, modulePath string, docID string, pos symbols.Position) []string {
	if snapshot == nil || modulePath == "" {
		return nil
	}

	resolved := collectMatchingModuleNames(modulePath, allModuleNames(snapshot))
	if len(resolved) > 0 {
		return resolved
	}

	for _, importRoot := range importRootsFromSource(source) {
		candidate := importRoot + "::" + modulePath
		if len(snapshot.ModulesByName(candidate)) > 0 {
			resolved = append(resolved, candidate)
		}
	}
	if len(resolved) > 0 {
		return resolved
	}

	for _, candidate := range h.likelyModulePathCandidates(docID, pos, modulePath) {
		if len(snapshot.ModulesByName(candidate)) > 0 {
			resolved = append(resolved, candidate)
		}
	}
	if len(resolved) > 0 {
		return resolved
	}

	return []string{modulePath}
}

// collectFirstMatchInModules searches for symbolName across all modules named
// in resolvedModules, sorts matches by FQN, and returns the first match.
func collectFirstMatchInModules(snapshot *project_state.ProjectSnapshot, resolvedModules []string, symbolName string) option.Option[symbols.Indexable] {
	var matches []symbols.Indexable
	for _, resolvedModule := range resolvedModules {
		for _, module := range snapshot.ModulesByName(resolvedModule) {
			matches = append(matches, collectDirectMatchesInModule(module, symbolName)...)
		}
	}
	if len(matches) == 0 {
		return option.None[symbols.Indexable]()
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].GetFQN() < matches[j].GetFQN()
	})
	return option.Some(matches[0])
}

func (h *Server) resolveSymbolCommonFallbacks(requestCtx stdctx.Context, docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	if fallback := h.resolveQualifiedSymbolHoverFallback(docID, pos); fallback.IsSome() {
		return fallback
	}
	if fallback := h.resolveImportedMethodHoverFallback(requestCtx, docID, pos); fallback.IsSome() {
		return fallback
	}
	if fallback := h.resolveImportedSymbolHoverFallback(docID, pos); fallback.IsSome() {
		return fallback
	}
	return option.None[symbols.Indexable]()
}
