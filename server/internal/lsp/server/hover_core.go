package server

import (
	stdctx "context"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	ctx "github.com/pherrymason/c3-lsp/internal/lsp/context"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const (
	hoverIndexSettleTimeout      = 150 * time.Millisecond
	hoverIndexSettlePollInterval = 10 * time.Millisecond
	hoverIndexingMessage         = "Indexing project symbols...\n\nTry hover again in a moment."
)

var importRootPattern = regexp.MustCompile(`(?m)^\s*import\s+([A-Za-z_][A-Za-z0-9_:]*)`)

// Support "Hover"
func (h *Server) TextDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	return h.textDocumentHoverWithTrace(context, params, "", stdctx.Background())
}

func (h *Server) textDocumentHoverWithTrace(context *glsp.Context, params *protocol.HoverParams, trace string, requestCtx stdctx.Context) (*protocol.Hover, error) {
	if requestCtx == nil {
		requestCtx = stdctx.Background()
	}

	start := time.Now()
	ensureDuration := time.Duration(0)
	contextDuration := time.Duration(0)
	resolveDuration := time.Duration(0)
	defer func() {
		if h.server != nil {
			perfLogf(
				h.server.Log,
				"textDocument/hover",
				start,
				"phase=total %s uri=%s line=%d char=%d ensure=%s build_context=%s resolve=%s",
				trace,
				params.TextDocument.URI,
				params.Position.Line,
				params.Position.Character,
				ensureDuration,
				contextDuration,
				resolveDuration,
			)
		}
	}()

	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}

	ensureStart := time.Now()
	h.ensureDocumentIndexedWithProgress(context, params.TextDocument.URI)
	ensureDuration = time.Since(ensureStart)

	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}
	h.indexImportedRootsForHover(params.TextDocument.URI)

	ctxStart := time.Now()
	cursorContext := ctx.BuildFromDocumentPosition(params.Position, params.TextDocument.URI, h.state)
	contextDuration = time.Since(ctxStart)
	if cursorContext.IsLiteral {
		return nil, nil
	}

	resolveStart := time.Now()
	pos := symbols.NewPositionFromLSPPosition(params.Position)
	docId := utils.NormalizePath(params.TextDocument.URI)
	docForCursor := h.state.GetDocument(docId)
	moduleTokenCursor := false
	if docForCursor != nil {
		_, moduleTokenCursor = extractModuleTokenAt(docForCursor.SourceCode.Text, pos.IndexIn(docForCursor.SourceCode.Text))
	}
	if moduleTokenCursor {
		if module := h.resolveModuleTokenHoverFallback(docId, pos); module.IsSome() {
			found := module.Get()
			h.enrichConstantSymbolFromSource(found)
			h.enrichUnwrapBindingVariableTypeFromSource(requestCtx, docId, found)
			h.enrichInferredLambdaParamTypeFromContext(docId, pos, found)
			docCommentData := h.hoverDocCommentForSymbol(found)
			docComment := ""
			if docCommentData != nil {
				docComment = "\n\n" + docCommentData.DisplayBodyWithContracts()
			}
			hoverInfo := found.GetHoverInfo()
			if _, isModule := found.(*symbols.Module); isModule && !strings.HasPrefix(strings.TrimSpace(hoverInfo), "module ") {
				hoverInfo = "module " + hoverInfo
			}
			hover := protocol.Hover{Contents: protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: "```c3\n" + hoverInfo + "\n```" + docComment}}
			resolveDuration = time.Since(resolveStart)
			return &hover, nil
		}
		if indexingHover := h.indexingHover(params.TextDocument.URI); indexingHover != nil {
			resolveDuration = time.Since(resolveStart)
			return indexingHover, nil
		}

		// For module-token positions (`foo::bar` on `foo`), only module resolution is valid.
		// Do not fall back to generic symbol lookup (e.g. unrelated `@foo` macros).
		resolveDuration = time.Since(resolveStart)
		return nil, nil
	}
	foundSymbolOption := h.findSymbolDeclarationWithContext(requestCtx, docId, pos)
	if foundSymbolOption.IsNone() {
		for _, candidatePos := range hoverTokenBoundaryPositions(h.state, docId, pos) {
			candidate := h.findSymbolDeclarationWithContext(requestCtx, docId, candidatePos)
			if candidate.IsSome() {
				foundSymbolOption = candidate
				break
			}
		}
	}
	if foundSymbolOption.IsNone() && shouldRetryHoverNeighborLookup(h.state, docId, pos) {
		candidatePositions := []symbols.Position{}
		if pos.Character > 0 {
			candidatePositions = append(candidatePositions, symbols.NewPosition(pos.Line, pos.Character-1))
		}
		if pos.Character > 1 {
			candidatePositions = append(candidatePositions, symbols.NewPosition(pos.Line, pos.Character-2))
		}
		candidatePositions = append(candidatePositions, symbols.NewPosition(pos.Line, pos.Character+1))
		candidatePositions = append(candidatePositions, symbols.NewPosition(pos.Line, pos.Character+2))

		for _, candidatePos := range candidatePositions {
			candidate := h.findSymbolDeclarationWithContext(requestCtx, docId, candidatePos)
			if candidate.IsSome() {
				foundSymbolOption = candidate
				break
			}
		}
	}
	if foundSymbolOption.IsNone() && h.waitForHoverIndexing(requestCtx, params.TextDocument.URI) {
		foundSymbolOption = h.findSymbolDeclarationWithContext(requestCtx, docId, pos)
		if foundSymbolOption.IsNone() {
			for _, candidatePos := range hoverTokenBoundaryPositions(h.state, docId, pos) {
				candidate := h.findSymbolDeclarationWithContext(requestCtx, docId, candidatePos)
				if candidate.IsSome() {
					foundSymbolOption = candidate
					break
				}
			}
		}
		if foundSymbolOption.IsNone() && shouldRetryHoverNeighborLookup(h.state, docId, pos) {
			candidatePositions := []symbols.Position{}
			if pos.Character > 0 {
				candidatePositions = append(candidatePositions, symbols.NewPosition(pos.Line, pos.Character-1))
			}
			if pos.Character > 1 {
				candidatePositions = append(candidatePositions, symbols.NewPosition(pos.Line, pos.Character-2))
			}
			candidatePositions = append(candidatePositions, symbols.NewPosition(pos.Line, pos.Character+1))
			candidatePositions = append(candidatePositions, symbols.NewPosition(pos.Line, pos.Character+2))

			for _, candidatePos := range candidatePositions {
				candidate := h.findSymbolDeclarationWithContext(requestCtx, docId, candidatePos)
				if candidate.IsSome() {
					foundSymbolOption = candidate
					break
				}
			}
		}
	}
	if designator := h.resolveDesignatedStructMemberHoverFallback(docId, pos); designator.IsSome() {
		foundSymbolOption = designator
	}
	if moduleToken := h.resolveModuleTokenHoverFallback(docId, pos); moduleToken.IsSome() {
		foundSymbolOption = moduleToken
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveModuleSeparatorSymbolHoverFallback(requestCtx, docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveQualifiedSymbolHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveQualifiedCallHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveLambdaFnSymbolHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveImportedMethodHoverFallback(requestCtx, docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveDesignatedStructMemberHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveImportedSymbolHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveIdentifierTokenHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.retryImportedIdentifierHoverResolution(requestCtx, params.TextDocument.URI, docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveQualifiedSymbolHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if fallback := h.resolveQualifiedSymbolFromImportsFallback(docId, pos); fallback.IsSome() {
			foundSymbolOption = fallback
		}
	}
	if foundSymbolOption.IsNone() {
		if syntheticHover := h.syntheticCollectionLenHover(docId, pos); syntheticHover != nil {
			resolveDuration = time.Since(resolveStart)
			return syntheticHover, nil
		}
		if syntheticHover := h.syntheticLambdaFnHover(docId, pos); syntheticHover != nil {
			resolveDuration = time.Since(resolveStart)
			return syntheticHover, nil
		}
		if indexingHover := h.indexingHover(params.TextDocument.URI); indexingHover != nil {
			resolveDuration = time.Since(resolveStart)
			return indexingHover, nil
		}
		if syntheticHover := h.syntheticIdentifierHover(docId, pos); syntheticHover != nil {
			resolveDuration = time.Since(resolveStart)
			return syntheticHover, nil
		}

		resolveDuration = time.Since(resolveStart)
		return nil, nil
	}

	foundSymbol := foundSymbolOption.Get()
	if isNilIndexable(foundSymbol) {
		if fallback := h.resolveModuleSeparatorSymbolHoverFallback(requestCtx, docId, pos); fallback.IsSome() {
			foundSymbol = fallback.Get()
		} else if fallback := h.resolveModuleTokenHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbol = fallback.Get()
		} else if fallback := h.resolveQualifiedSymbolHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbol = fallback.Get()
		} else if fallback := h.resolveQualifiedCallHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbol = fallback.Get()
		} else if fallback := h.resolveImportedMethodHoverFallback(requestCtx, docId, pos); fallback.IsSome() {
			foundSymbol = fallback.Get()
		} else if fallback := h.resolveDesignatedStructMemberHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbol = fallback.Get()
		} else if fallback := h.resolveImportedSymbolHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbol = fallback.Get()
		} else if fallback := h.resolveIdentifierTokenHoverFallback(docId, pos); fallback.IsSome() {
			foundSymbol = fallback.Get()
		} else {
			return nil, nil
		}
	}
	if isNilIndexable(foundSymbol) {
		return nil, nil
	}
	h.enrichConstantSymbolFromSource(foundSymbol)
	h.enrichUnwrapBindingVariableTypeFromSource(requestCtx, docId, foundSymbol)
	h.enrichInferredLambdaParamTypeFromContext(docId, pos, foundSymbol)
	doc := h.state.GetDocument(docId)

	// expected behaviour:
	// hovering on variables: display variable type + any description
	// hovering on functions: display function signature + docs
	// hovering on members: same as variable

	docCommentData := h.hoverDocCommentForSymbol(foundSymbol)
	docComment := ""
	if docCommentData != nil {
		docComment = "\n\n" + docCommentData.DisplayBodyWithContracts()
	}

	faultsSection := h.buildFunctionFaultsHoverSection(requestCtx, docId, foundSymbol)

	extraLine := ""

	_, isModule := foundSymbol.(*symbols.Module)
	if !isModule {
		extraLine += "\n\nIn module **[" + h.hoverModuleDisplayName(foundSymbol) + "]**"
	}

	moduleGenericConstraints := ""
	if !isModule {
		if module := findModuleByName(h.state, foundSymbol.GetModuleString()); module != nil {
			constraints := symbols.ModuleGenericConstraintMarkdown(module)
			if constraints != "" {
				moduleGenericConstraints = "\n\n" + constraints
			}
		}
	}

	sizeInfo := ""
	if utils.IsFeatureEnabled("SIZE_ON_HOVER") {
		if hasSize(foundSymbol) {
			sizeInfo = "// size = " + calculateSize(foundSymbol) + ", align = " + calculateAlignment(foundSymbol) + "\n"
		}
	}

	hoverInfo := foundSymbol.GetHoverInfo()
	if isModule && !strings.HasPrefix(strings.TrimSpace(hoverInfo), "module ") {
		hoverInfo = "module " + hoverInfo
	}
	if doc != nil {
		if genericSuffix, ok := genericTypeSuffixAtPosition(doc.SourceCode.Text, pos); ok {
			hoverInfo = appendGenericSuffixToHoverInfo(foundSymbol, hoverInfo, genericSuffix)
		}
	}

	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind: protocol.MarkupKindMarkdown,
			Value: "```c3" + "\n" +
				sizeInfo +
				hoverInfo + "\n```" +
				extraLine +
				faultsSection +
				docComment +
				moduleGenericConstraints,
		},
	}

	resolveDuration = time.Since(resolveStart)

	return &hover, nil
}

func shouldRetryHoverNeighborLookup(state *project_state.ProjectState, docID string, pos symbols.Position) bool {
	if state == nil {
		return false
	}

	doc := state.GetDocument(docID)
	if doc == nil {
		return false
	}

	unitModules := state.GetUnitModulesByDoc(doc.URI)
	if unitModules == nil {
		return false
	}

	symbol := doc.SourceCode.SymbolInPosition(pos, unitModules).Text()
	if symbol == "" {
		return true
	}

	for _, r := range symbol {
		if !utils.IsAZ09_(r) && r != '@' && r != '$' && r != '#' {
			return true
		}
	}

	return false
}

func hoverTokenBoundaryPositions(state *project_state.ProjectState, docID string, pos symbols.Position) []symbols.Position {
	if state == nil {
		return nil
	}

	doc := state.GetDocument(docID)
	if doc == nil {
		return nil
	}

	unitModules := state.GetUnitModulesByDoc(doc.URI)
	if unitModules == nil {
		return nil
	}

	word := doc.SourceCode.SymbolInPosition(pos, unitModules)
	if word.Text() == "" {
		return nil
	}

	rng := word.TextRange()
	if rng.Start.Line != pos.Line || rng.End.Line != pos.Line {
		return nil
	}

	positions := []symbols.Position{}
	if rng.Start.Character != pos.Character {
		positions = append(positions, symbols.NewPosition(pos.Line, rng.Start.Character))
	}
	if rng.End.Character > rng.Start.Character {
		endChar := rng.End.Character - 1
		if endChar != pos.Character {
			positions = append(positions, symbols.NewPosition(pos.Line, endChar))
		}
	}

	return positions
}

func hoverNeighborPositions(pos symbols.Position) []symbols.Position {
	positions := []symbols.Position{}
	if pos.Character > 0 {
		positions = append(positions, symbols.NewPosition(pos.Line, pos.Character-1))
	}
	if pos.Character > 1 {
		positions = append(positions, symbols.NewPosition(pos.Line, pos.Character-2))
	}
	positions = append(positions, symbols.NewPosition(pos.Line, pos.Character+1))
	positions = append(positions, symbols.NewPosition(pos.Line, pos.Character+2))
	return positions
}

func (h *Server) shouldDelaySyntheticIdentifierHover(docID string, pos symbols.Position) bool {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return false
	}

	ident, ok := extractIdentifierTokenAt(doc.SourceCode.Text, pos.IndexIn(doc.SourceCode.Text))
	if !ok || ident == "" {
		return false
	}

	first := rune(ident[0])
	if first < 'A' || first > 'Z' {
		return false
	}

	unit := h.state.GetUnitModulesByDoc(docID)
	if unit != nil {
		for _, module := range unit.Modules() {
			if module != nil && len(module.Imports) > 0 {
				return true
			}
		}
	}

	return len(importRootsFromSource(doc.SourceCode.Text)) > 0
}

func (h *Server) retryImportedIdentifierHoverResolution(requestCtx stdctx.Context, uri protocol.DocumentUri, docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	if !h.shouldDelaySyntheticIdentifierHover(docID, pos) {
		return option.None[symbols.Indexable]()
	}

	h.preloadImportedRootModulesForURIForce(uri)
	importRoots := importRootsFromSource(h.state.GetDocument(docID).SourceCode.Text)
	for _, importRoot := range importRoots {
		h.indexImportRootCandidates(uri, importRoot)
	}
	if unit := h.state.GetUnitModulesByDoc(docID); unit != nil {
		seenImports := map[string]struct{}{}
		for _, module := range unit.Modules() {
			if module == nil {
				continue
			}
			for _, imported := range module.Imports {
				if imported == "" {
					continue
				}
				if _, ok := seenImports[imported]; ok {
					continue
				}
				seenImports[imported] = struct{}{}
				h.tryLoadLikelyModuleFiles(docID, pos, imported)
			}
		}
		for _, importRoot := range importRoots {
			if _, ok := seenImports[importRoot]; ok {
				continue
			}
			seenImports[importRoot] = struct{}{}
			h.tryLoadLikelyModuleFiles(docID, pos, importRoot)
		}
	} else {
		for _, importRoot := range importRoots {
			h.tryLoadLikelyModuleFiles(docID, pos, importRoot)
		}
	}

	positions := []symbols.Position{pos}
	positions = append(positions, hoverTokenBoundaryPositions(h.state, docID, pos)...)
	positions = append(positions, hoverNeighborPositions(pos)...)

	for _, candidatePos := range positions {
		if found := h.findSymbolDeclarationWithContext(requestCtx, docID, candidatePos); found.IsSome() {
			return found
		}
		if fallback := h.resolveSymbolCommonFallbacks(requestCtx, docID, candidatePos); fallback.IsSome() {
			return fallback
		}
	}

	return option.None[symbols.Indexable]()
}

func (h *Server) resolveQualifiedSymbolFromImportsFallback(docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}

	modulePath, symbolName, ok := extractQualifiedSymbolAt(doc.SourceCode.Text, pos.IndexIn(doc.SourceCode.Text))
	if !ok || modulePath == "" || symbolName == "" {
		return option.None[symbols.Indexable]()
	}

	uri := h.documentURIFromDocID(docID)
	for _, importRoot := range importRootsFromSource(doc.SourceCode.Text) {
		h.indexImportRootCandidates(uri, importRoot)
	}

	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}

	resolvedModules := h.resolveModuleCandidatesFromSnapshotOnly(snapshot, doc.SourceCode.Text, modulePath, docID, pos)
	return collectFirstMatchInModules(snapshot, resolvedModules, symbolName)
}

func (h *Server) indexImportedRootsForHover(uri protocol.DocumentUri) {
	docID := string(uri)
	if normalized := utils.NormalizePath(docID); normalized != "" {
		docID = normalized
	}
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return
	}
	for _, importRoot := range importRootsFromSource(doc.SourceCode.Text) {
		h.indexImportRootCandidates(uri, importRoot)
	}
}

func (h *Server) indexImportRootCandidates(uri protocol.DocumentUri, importRoot string) {
	if importRoot == "" {
		return
	}

	root := h.resolveProjectRootForURI(&uri)
	root = fs.GetCanonicalPath(root)
	if root == "" {
		return
	}

	searchRoots := []string{root}
	searchRoots = append(searchRoots, h.workspaceDependencyDirs...)
	seen := map[string]struct{}{}
	for _, searchRoot := range searchRoots {
		for _, candidate := range []string{
			filepath.Join(searchRoot, importRoot+".c3l"),
			filepath.Join(searchRoot, importRoot+".c3i"),
			filepath.Join(searchRoot, importRoot+".c3"),
			filepath.Join(searchRoot, "src", importRoot+".c3i"),
			filepath.Join(searchRoot, "src", importRoot+".c3"),
		} {
			candidate = fs.GetCanonicalPath(candidate)
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}

			docs, err := loadSourceDocuments(candidate)
			if err != nil {
				continue
			}
			for _, doc := range docs {
				if doc.readErr != nil || h.state.GetDocument(doc.path) != nil {
					continue
				}
				h.indexFileWithContent(doc.path, []byte(doc.content))
			}
		}
	}
}

func importRootsFromSource(source string) []string {
	matches := importRootPattern.FindAllStringSubmatch(source, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	roots := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		root := match[1]
		if sep := strings.Index(root, "::"); sep >= 0 {
			root = root[:sep]
		}
		if root == "" {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		roots = append(roots, root)
	}

	return roots
}

func (h *Server) waitForHoverIndexing(requestCtx stdctx.Context, uri protocol.DocumentUri) bool {
	root, state, ok := h.hoverRootState(uri)
	if !ok || state != rootStateIndexing {
		return false
	}

	deadline := time.Now().Add(hoverIndexSettleTimeout)
	for {
		if requestCtx != nil {
			select {
			case <-requestCtx.Done():
				return false
			default:
			}
		}

		if h.rootState(root) == rootStateIndexed {
			return true
		}
		if h.rootState(root) != rootStateIndexing {
			return false
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return false
		}
		sleepFor := hoverIndexSettlePollInterval
		if remaining < sleepFor {
			sleepFor = remaining
		}
		time.Sleep(sleepFor)
	}
}

func (h *Server) indexingHover(uri protocol.DocumentUri) *protocol.Hover {
	_, state, ok := h.hoverRootState(uri)
	if !ok || state != rootStateIndexing {
		return nil
	}

	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: hoverIndexingMessage,
		},
	}

	return &hover
}

func (h *Server) hoverRootState(uri protocol.DocumentUri) (string, workspaceIndexState, bool) {
	root := h.resolveProjectRootForURI(&uri)
	if !isBuildableProjectRoot(root) {
		return "", rootStateNotIndexed, false
	}

	return root, h.rootState(root), true
}
