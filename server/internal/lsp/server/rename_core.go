package server

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var moduleNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(::[A-Za-z_][A-Za-z0-9_]*)*$`)
var identifierNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// renameTarget describes the symbol targeted for rename.
type renameTarget struct {
	name         string
	renameRange  protocol.Range
	declaration  symbols.Indexable
	sourceDocURI string

	moduleFullName     string
	moduleSegmentIndex int
}

// symbolIdentity uniquely identifies a symbol declaration.
type symbolIdentity struct {
	docURI  string
	kind    protocol.CompletionItemKind
	idRange symbols.Range
}

// renameExecutionCache caches declaration look-ups performed during a single
// rename execution to avoid redundant work.
type renameExecutionCache struct {
	declarationByCandidate map[string]cachedDeclarationResult
	structOwnerByMember    map[string]cachedStructOwnerResult

	declarationLookups int
	declarationHits    int
	structOwnerLookups int
	structOwnerHits    int
}

type cachedDeclarationResult struct {
	decl  symbols.Indexable
	found bool
}

type cachedStructOwnerResult struct {
	owner string
	found bool
}

func newRenameExecutionCache() *renameExecutionCache {
	return &renameExecutionCache{
		declarationByCandidate: map[string]cachedDeclarationResult{},
		structOwnerByMember:    map[string]cachedStructOwnerResult{},
	}
}

// TextDocumentPrepareRename handles the textDocument/prepareRename LSP request.
func (h *Server) TextDocumentPrepareRename(glspContext *glsp.Context, params *protocol.PrepareRenameParams) (any, error) {
	return h.textDocumentPrepareRenameWithTrace(glspContext, params, "", context.Background())
}

func (h *Server) textDocumentPrepareRenameWithTrace(glspContext *glsp.Context, params *protocol.PrepareRenameParams, trace string, requestCtx context.Context) (any, error) {
	if requestCtx == nil {
		requestCtx = context.Background()
	}

	prepareStartedAt := time.Now()
	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}

	loadStartedAt := time.Now()
	doc, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	loadDuration := time.Since(loadStartedAt)
	targetDuration := time.Duration(0)
	foundTarget := false
	defer func() {
		if h.server == nil {
			return
		}

		if perfEnabled() {
			perfLogf(
				h.server.Log,
				"textDocument/prepareRename",
				prepareStartedAt,
				"phase=total %s uri=%s line=%d char=%d found=%t loadPhase=%s targetPhase=%s",
				trace,
				params.TextDocument.URI,
				params.Position.Line,
				params.Position.Character,
				foundTarget,
				loadDuration,
				targetDuration,
			)
		}
	}()

	if doc == nil {
		return nil, nil
	}

	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}

	targetStartedAt := time.Now()
	target, ok := moduleRenameTarget(doc.SourceCode.Text, params.Position, unitModules)
	if !ok {
		target, ok = quickParameterRenameTargetAtDelimiter(doc.SourceCode.Text, params.Position, unitModules)
	}
	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}
	if !ok {
		target, ok = h.symbolRenameTargetWithTimeout(doc.URI, doc.SourceCode.Text, params.Position, unitModules)
	}
	targetDuration = time.Since(targetStartedAt)
	foundTarget = ok
	if !ok {
		return nil, nil
	}
	placeholder := renamePlaceholder(doc.SourceCode.Text, target)
	if placeholder == "" {
		return nil, nil
	}

	return protocol.RangeWithPlaceholder{
		Range:       target.renameRange,
		Placeholder: placeholder,
	}, nil
}

func renamePlaceholder(source string, target renameTarget) string {
	if target.name != "" && !strings.Contains(target.name, "::") {
		return target.name
	}

	start := symbols.NewPositionFromLSPPosition(target.renameRange.Start).IndexIn(source)
	end := symbols.NewPositionFromLSPPosition(target.renameRange.End).IndexIn(source)
	if start < 0 || end <= start || end > len(source) {
		if target.name != "" {
			parts := strings.Split(target.name, "::")
			return parts[len(parts)-1]
		}
		return ""
	}

	segment := source[start:end]
	if !strings.Contains(segment, "::") {
		return strings.TrimSpace(segment)
	}

	parts := strings.Split(segment, "::")
	last := strings.TrimSpace(parts[len(parts)-1])
	if last != "" {
		return last
	}

	if target.name != "" {
		nameParts := strings.Split(target.name, "::")
		return nameParts[len(nameParts)-1]
	}

	return ""
}

// TextDocumentRename handles the textDocument/rename LSP request.
func (h *Server) TextDocumentRename(context *glsp.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	renameStartedAt := time.Now()
	targetStartedAt := time.Now()
	doc, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if doc == nil {
		return emptyWorkspaceEdit(), nil
	}

	h.maybeWarnPartialWorkspaceIndexForRename(context, params.TextDocument.URI)

	if moduleTarget, ok := moduleRenameTarget(doc.SourceCode.Text, params.Position, unitModules); ok {
		if h.server != nil {
			renameDebugf(h.server.Log, "module rename target=%s new=%s uri=%s", moduleTarget.name, params.NewName, params.TextDocument.URI)
		}
		oldModuleFullName := moduleTarget.moduleFullName
		if oldModuleFullName == "" {
			oldModuleFullName = moduleTarget.name
		}

		newModuleFullName := ""
		if strings.Contains(params.NewName, "::") {
			if !moduleNamePattern.MatchString(params.NewName) {
				return nil, fmt.Errorf("invalid module name: %s", params.NewName)
			}
			newModuleFullName = params.NewName
		} else {
			if !identifierNamePattern.MatchString(params.NewName) {
				return nil, fmt.Errorf("invalid module segment name: %s", params.NewName)
			}
			if moduleTarget.moduleSegmentIndex < 0 {
				return nil, fmt.Errorf("invalid module rename target")
			}

			renamed, ok := replaceModulePathSegment(oldModuleFullName, moduleTarget.moduleSegmentIndex, params.NewName)
			if !ok {
				return nil, fmt.Errorf("invalid module rename target")
			}
			newModuleFullName = renamed
		}

		if oldModuleFullName == newModuleFullName {
			return emptyWorkspaceEdit(), nil
		}

		changes := map[protocol.DocumentUri][]protocol.TextEdit{}
		for docID := range h.state.GetAllUnitModules() {
			otherDoc := h.state.GetDocument(string(docID))
			if otherDoc == nil {
				continue
			}

			edits := moduleRenameEdits(otherDoc.SourceCode.Text, oldModuleFullName, newModuleFullName)
			if len(edits) == 0 {
				continue
			}

			changes[toWorkspaceEditURI(otherDoc.URI, h.options.C3.StdlibPath)] = edits
		}

		return h.workspaceEditFromChanges(changes), nil
	}

	target, ok := h.symbolRenameTarget(doc.URI, doc.SourceCode.Text, params.Position, unitModules)
	if !ok {
		if h.server != nil {
			renameDebugf(h.server.Log, "no rename target found uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character)
		}
		return emptyWorkspaceEdit(), nil
	}
	targetDuration := time.Since(targetStartedAt)

	normalizedName, err := validateSymbolRenameNewName(target.declaration, params.NewName)
	if err != nil {
		return nil, err
	}

	if target.name == normalizedName {
		return emptyWorkspaceEdit(), nil
	}

	if err := h.validateRenameNoConflict(target, normalizedName); err != nil {
		return nil, err
	}

	cache := newRenameExecutionCache()
	editsStartedAt := time.Now()
	changes := h.semanticRenameChangesFromReferences(target, normalizedName)
	if len(changes) == 0 {
		if h.server != nil {
			renameDebugf(h.server.Log, "references-backed rename returned no edits; falling back target=%s uri=%s", target.name, params.TextDocument.URI)
		}
		changes = h.semanticRenameChanges(target, normalizedName, cache)
	} else {
		if h.server != nil {
			renameDebugf(h.server.Log, "references-backed rename used target=%s docs=%d", target.name, len(changes))
		}
	}
	changes = h.appendParameterDocContractRenameEdits(changes, target, normalizedName)
	editsDuration := time.Since(editsStartedAt)

	if perfEnabled() {
		totalEdits := 0
		for _, docEdits := range changes {
			totalEdits += len(docEdits)
		}
		perfLogf(
			h.server.Log,
			"textDocument/rename",
			renameStartedAt,
			"uri=%s target=%s docs=%d edits=%d targetPhase=%s editPhase=%s declCache=%d/%d ownerCache=%d/%d",
			params.TextDocument.URI,
			target.name,
			len(changes),
			totalEdits,
			targetDuration,
			editsDuration,
			cache.declarationHits,
			cache.declarationLookups,
			cache.structOwnerHits,
			cache.structOwnerLookups,
		)
	}

	return h.workspaceEditFromChanges(changes), nil
}

// getOrLoadDocumentForRename loads a document for rename operations, parsing it
// fresh if it is not yet indexed.
func (h *Server) getOrLoadDocumentForRename(uri protocol.DocumentUri) (*document.Document, *symbols_table.UnitModules) {
	docURI := utils.NormalizePath(uri)
	doc := h.state.GetDocument(docURI)
	if doc == nil {
		if !h.loadAndIndexFile(docURI) {
			return nil, nil
		}
		doc = h.state.GetDocument(docURI)
	}
	if doc == nil {
		return nil, nil
	}

	unitModules := h.state.GetUnitModulesByDoc(doc.URI)
	if unitModules == nil {
		h.state.RefreshDocumentIdentifiers(doc, h.parser)
		unitModules = h.state.GetUnitModulesByDoc(doc.URI)
	}

	return doc, unitModules
}
