package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func initializeWorkspaceURI(params *protocol.InitializeParams) *protocol.DocumentUri {
	if params.RootURI != nil {
		return params.RootURI
	}

	if len(params.WorkspaceFolders) > 0 {
		uri := params.WorkspaceFolders[0].URI
		return &uri
	}

	if params.RootPath != nil {
		uri := protocol.DocumentUri(fs.ConvertPathToURI(*params.RootPath, option.None[string]()))
		return &uri
	}

	return nil
}

func supportsWorkspaceFoldersRequest(capabilities protocol.ClientCapabilities) bool {
	if capabilities.Workspace == nil || capabilities.Workspace.WorkspaceFolders == nil {
		return false
	}

	return *capabilities.Workspace.WorkspaceFolders
}

func resolveWorkspaceURI(context *glsp.Context, params *protocol.InitializeParams) *protocol.DocumentUri {
	uri := initializeWorkspaceURI(params)
	if uri != nil {
		return uri
	}

	if context == nil || !supportsWorkspaceFoldersRequest(params.Capabilities) {
		return nil
	}

	workspaceFolders := []protocol.WorkspaceFolder{}
	context.Call(protocol.ServerWorkspaceWorkspaceFolders, nil, &workspaceFolders)
	if len(workspaceFolders) == 0 {
		return nil
	}

	folderURI := workspaceFolders[0].URI
	return &folderURI
}

// Support "Hover"
func (s *Server) Initialize(serverName string, serverVersion string, capabilities protocol.ServerCapabilities, context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	start := time.Now()
	defer func() {
		if s.server != nil {
			perfLogf(s.server.Log, "initialize", start, "server=%s version=%s", serverName, serverVersion)
		}
	}()

	s.clientCapabilities = params.Capabilities
	//capabilities := handler.CreateServerCapabilities()

	change := protocol.TextDocumentSyncKindIncremental
	capabilities.TextDocumentSync = protocol.TextDocumentSyncOptions{
		OpenClose:         cast.ToPtr(true),
		Change:            &change,
		WillSave:          cast.ToPtr(true),
		WillSaveWaitUntil: cast.ToPtr(true),
		Save:              cast.ToPtr(true),
	}
	capabilities.DeclarationProvider = true
	capabilities.DefinitionProvider = true
	capabilities.TypeDefinitionProvider = true
	capabilities.ImplementationProvider = true
	capabilities.ReferencesProvider = true
	capabilities.DocumentHighlightProvider = true
	capabilities.DocumentSymbolProvider = true
	capabilities.CodeActionProvider = &protocol.CodeActionOptions{
		CodeActionKinds: []protocol.CodeActionKind{
			protocol.CodeActionKindQuickFix,
			protocol.CodeActionKindRefactor,
		},
		ResolveProvider: cast.ToPtr(true),
	}
	capabilities.WorkspaceSymbolProvider = true
	capabilities.FoldingRangeProvider = true
	capabilities.SelectionRangeProvider = true
	capabilities.LinkedEditingRangeProvider = true
	capabilities.MonikerProvider = true
	capabilities.CallHierarchyProvider = true
	capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: cast.ToPtr(true)}
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{".", ":"},
	}
	capabilities.SignatureHelpProvider = &protocol.SignatureHelpOptions{
		TriggerCharacters:   []string{"(", ","},
		RetriggerCharacters: []string{")"},
	}
	capabilities.DocumentFormattingProvider = true
	capabilities.DocumentRangeFormattingProvider = true
	capabilities.DocumentOnTypeFormattingProvider = &protocol.DocumentOnTypeFormattingOptions{
		FirstTriggerCharacter: "}",
		MoreTriggerCharacter:  onTypeFormattingAdditionalTriggerCharacters(),
	}
	capabilities.DocumentLinkProvider = &protocol.DocumentLinkOptions{ResolveProvider: cast.ToPtr(true)}
	capabilities.ExecuteCommandProvider = &protocol.ExecuteCommandOptions{Commands: workspaceExecuteCommands()}
	capabilities.Workspace = &protocol.ServerCapabilitiesWorkspace{
		WorkspaceFolders: &protocol.WorkspaceFoldersServerCapabilities{
			Supported: cast.ToPtr(true),
			ChangeNotifications: &protocol.BoolOrString{
				Value: true,
			},
		},
		FileOperations: &protocol.ServerCapabilitiesWorkspaceFileOperations{
			WillCreate: &protocol.FileOperationRegistrationOptions{
				Filters: []protocol.FileOperationFilter{{
					Pattern: protocol.FileOperationPattern{
						Glob: "**/*.{c3,c3i,c3l}",
					},
				}},
			},
			DidCreate: &protocol.FileOperationRegistrationOptions{
				Filters: []protocol.FileOperationFilter{{
					Pattern: protocol.FileOperationPattern{
						Glob: "**/*.{c3,c3i,c3l}",
					},
				}},
			},
			DidDelete: &protocol.FileOperationRegistrationOptions{
				Filters: []protocol.FileOperationFilter{{
					Pattern: protocol.FileOperationPattern{
						Glob: "**/*.{c3,c3i,c3l}",
					},
				}},
			},
			WillRename: &protocol.FileOperationRegistrationOptions{
				Filters: []protocol.FileOperationFilter{{
					Pattern: protocol.FileOperationPattern{
						Glob: "**/*.{c3,c3i,c3l}",
					},
				}},
			},
			DidRename: &protocol.FileOperationRegistrationOptions{
				Filters: []protocol.FileOperationFilter{{
					Pattern: protocol.FileOperationPattern{
						Glob: "**/*.{c3,c3i,c3l}",
					},
				}},
			},
			WillDelete: &protocol.FileOperationRegistrationOptions{
				Filters: []protocol.FileOperationFilter{{
					Pattern: protocol.FileOperationPattern{
						Glob: "**/*.{c3,c3i,c3l}",
					},
				}},
			},
		},
	}

	workspaceURI := resolveWorkspaceURI(context, params)
	if workspaceURI != nil {
		s.state.SetProjectRootURI(utils.NormalizePath(*workspaceURI))
		path, _ := fs.UriToPath(*workspaceURI)
		canonicalPath := fs.GetCanonicalPath(path)
		buildableRoot := isBuildableProjectRoot(path)
		s.configureProjectForRootWithContext(path, context)
		s.loadClientRuntimeConfiguration(context, workspaceURI)
		s.notifyWindowLogMessage(context, protocol.MessageTypeInfo, fmt.Sprintf("C3-LSP loaded workspace: %s", path))
		if buildableRoot {
			s.indexWorkspaceAtWithLSPContext(path, context)
			s.RunDiagnosticsFull(s.state, context.Notify, false)
			s.markRootIndexed(canonicalPath)
		} else {
			s.notifyWindowLogMessage(context, protocol.MessageTypeInfo, "C3-LSP detected aggregate workspace root; deferring indexing to opened C3 project files")
			s.clearRootTracking(canonicalPath)
		}

		if !buildableRoot {
			s.notifyWindowLogMessage(context, protocol.MessageTypeInfo, "C3-LSP skipped initial diagnostics: workspace root is not a C3 project root")
		}
		s.notifyTelemetryEvent(context, "c3lsp.initialize.workspace", map[string]any{
			"root":      path,
			"buildable": buildableRoot,
		})
	}

	// Disable diagnostics only if the client does not support publishDiagnostics at all.
	if params.Capabilities.TextDocument == nil || params.Capabilities.TextDocument.PublishDiagnostics == nil {
		s.options.Diagnostics.Enabled = false
		s.notifyWindowShowMessage(context, protocol.MessageTypeWarning, "C3-LSP diagnostics disabled: client does not support publishDiagnostics")
	}

	s.notifyTelemetryEvent(context, "c3lsp.initialize", map[string]any{
		"server":  serverName,
		"version": serverVersion,
	})

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &serverVersion,
		},
	}, nil
}

func (h *Server) indexWorkspaceWithLSPContext(lspContext *glsp.Context) {
	path := h.state.GetProjectRootURI()
	h.indexWorkspaceAtWithContextAndProgress(context.Background(), path, lspContext)
}

func (h *Server) indexWorkspaceAt(path string) {
	h.indexWorkspaceAtWithLSPContext(path, nil)
}

func (h *Server) indexWorkspaceAtWithLSPContext(path string, lspContext *glsp.Context) {
	h.indexWorkspaceAtWithContextAndProgress(context.Background(), path, lspContext)
}

func (h *Server) indexWorkspaceAtWithContext(ctx context.Context, path string) bool {
	return h.indexWorkspaceAtWithContextAndProgress(ctx, path, nil)
}

func (h *Server) indexWorkspaceAtWithContextAndProgress(ctx context.Context, path string, lspContext *glsp.Context) bool {
	start := time.Now()
	canonicalPath := fs.GetCanonicalPath(path)
	loadedFiles := 0
	failedFiles := 0
	workerCount := h.indexReadWorkerCount()
	var scanStats fs.ScanStats
	scanRootsCount := 0
	dependencyRootsCount := 0
	defer func() {
		if h.server != nil {
			perfLogf(
				h.server.Log,
				"indexWorkspaceAt",
				start,
				"path=%s workers=%d roots=%d dep_roots=%d loaded=%d failed=%d matched=%d skipped_dirs=%d visited_dirs=%d",
				canonicalPath,
				workerCount,
				scanRootsCount,
				dependencyRootsCount,
				loadedFiles,
				failedFiles,
				scanStats.Matched,
				scanStats.SkippedDirs,
				scanStats.VisitedDirs,
			)
		}
	}()

	if path == "" {
		return false
	}

	token, hasProgress := h.beginWorkDoneProgress(lspContext, "Workspace indexing", "Scanning workspace files", false)
	endMessage := "Workspace indexed"
	if hasProgress {
		defer func() {
			h.endWorkDoneProgress(lspContext, token, endMessage)
		}()
	}
	reportProgress := func(message string, pct int) {
		if !hasProgress {
			return
		}
		percentage := protocol.UInteger(pct)
		h.reportWorkDoneProgress(lspContext, token, message, &percentage)
	}

	scanRoots := []string{canonicalPath}
	scanRoots = append(scanRoots, h.workspaceDependencyDirs...)
	scanRootsCount = len(scanRoots)
	dependencyRootsCount = len(h.workspaceDependencyDirs)
	reportProgress("Resolving workspace and dependency roots", 5)

	allFiles := make([]string, 0, 1024)
	seenFiles := make(map[string]struct{}, 2048)
	totalRoots := len(scanRoots)
	if totalRoots == 0 {
		totalRoots = 1
	}
	for i, scanRoot := range scanRoots {
		reportProgress(fmt.Sprintf("Scanning root %d/%d: %s", i+1, len(scanRoots), scanRoot), 10+(20*(i+1)/totalRoots))

		priorityDirs := []string{}
		if scanRoot == canonicalPath {
			priorityDirs = h.indexPriorityDirs(canonicalPath)
		}

		files, stats, _ := fs.ScanForC3WithOptions(scanRoot, fs.ScanOptions{
			IgnoreDirs:   fs.DefaultC3ScanIgnoreDirs(),
			PriorityDirs: priorityDirs,
		})
		scanStats.Matched += stats.Matched
		scanStats.SkippedDirs += stats.SkippedDirs
		scanStats.VisitedDirs += stats.VisitedDirs

		for _, file := range files {
			canonicalFile := fs.GetCanonicalPath(file)
			if canonicalFile == "" {
				continue
			}
			if _, ok := seenFiles[canonicalFile]; ok {
				continue
			}
			seenFiles[canonicalFile] = struct{}{}
			allFiles = append(allFiles, canonicalFile)
		}
	}

	reportProgress(fmt.Sprintf("Preparing %d discovered files", len(allFiles)), 35)
	loadedDocs := h.loadDocumentsForIndexing(ctx, allFiles, workerCount)
	reportProgress(fmt.Sprintf("Read %d source documents", len(loadedDocs)), 45)

	totalDocs := len(loadedDocs)
	if totalDocs == 0 {
		totalDocs = 1
	}
	for i, loaded := range loadedDocs {
		select {
		case <-ctx.Done():
			if hasProgress {
				endMessage = "Workspace indexing cancelled"
			}
			return false
		default:
		}

		if loaded.readErr != nil {
			failedFiles++
			continue
		}

		loadedFiles++
		h.indexFileWithContent(loaded.path, []byte(loaded.content))

		if hasProgress && (i == 0 || (i+1)%25 == 0 || i+1 == totalDocs) {
			pct := protocol.UInteger(45 + (50 * (i + 1) / totalDocs))
			h.reportWorkDoneProgress(lspContext, token, fmt.Sprintf("Indexed %d/%d documents (%d failed)", i+1, totalDocs, failedFiles), &pct)
		}
	}
	reportProgress(fmt.Sprintf("Finalized workspace snapshot (%d indexed, %d failed)", loadedFiles, failedFiles), 100)

	return true
}

type loadedDocument struct {
	path    string
	content string
	readErr error
}

func (h *Server) indexPriorityDirs(root string) []string {
	priority := []string{}
	seen := make(map[string]struct{})

	addDir := func(p string) {
		if p == "" {
			return
		}
		p = fs.GetCanonicalPath(p)
		if p == "" || (p != root && !strings.HasPrefix(p, root+string(os.PathSeparator))) {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		priority = append(priority, p)
	}

	for docURI := range h.state.GetAllUnitModules() {
		dir := filepath.Dir(string(docURI))
		addDir(dir)
	}

	for _, docURI := range h.state.GetDocumentsForModules(h.state.GetLastInvalidationScope().ImpactedModules) {
		dir := filepath.Dir(docURI)
		addDir(dir)
	}

	return priority
}

func (h *Server) indexReadWorkerCount() int {
	workers := runtime.NumCPU() / 2
	if workers < 2 {
		workers = 2
	}
	if workers > 6 {
		workers = 6
	}

	return workers
}

func (h *Server) loadDocumentsForIndexing(ctx context.Context, files []string, workers int) []loadedDocument {
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan string, workers*2)
	results := make(chan loadedDocument, workers*2)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				docs, err := loadSourceDocuments(filePath)
				if err != nil {
					results <- loadedDocument{path: filePath, readErr: err}
					continue
				}
				for _, doc := range docs {
					results <- doc
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, filePath := range files {
			select {
			case <-ctx.Done():
				return
			case jobs <- filePath:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	loaded := make([]loadedDocument, 0, len(files))
	for result := range results {
		loaded = append(loaded, result)
	}

	return loaded
}
