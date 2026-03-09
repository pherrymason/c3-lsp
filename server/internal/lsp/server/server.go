package server

import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bep/debounce"
	"github.com/pherrymason/c3-lsp/internal/c3c"
	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_v2"
	"github.com/pherrymason/c3-lsp/pkg/option"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	"github.com/tliron/commonlog/simple"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Server struct {
	server  *glspserv.Server
	options ServerOpts
	version string

	state  *l.ProjectState
	parser *p.Parser
	search search.SearchInterface

	clientCapabilities protocol.ClientCapabilities

	diag                     diagnosticsCoordinator
	workspaceC3Options       c3c.C3Opts
	activeConfigRoot         string
	idx                      indexingCoordinator
	workspaceDependencyDirs  []string
	renameWarningRoots       map[string]bool
	rootCache                projectRootCacheState
	gate                     requestGate
	workspaceIndexer         func(ctx context.Context, path string)
	diagnosticsCommand       func(ctx context.Context, c3Options c3c.C3Opts, projectPath string) (bytes.Buffer, bytes.Buffer, error)
	completionCacheMu        sync.Mutex
	completionCache          map[completionCacheKey][]completionItemWithLabelDetails
	completionCacheOrder     []completionCacheKey
	normalizedDocIDCacheMu   sync.Mutex
	normalizedDocIDCache     map[string]string
	workDoneProgressMu       sync.Mutex
	workDoneProgressActive   map[string]struct{}
	workDoneProgressCanceled map[string]struct{}
	importPreloadMu          sync.Mutex
	importPreloadDone        map[string]struct{}
	cliDebug                 bool // records the original CLI --debug flag to prevent JSON from overwriting it
	initialized              atomic.Bool
	shutdownRequested        atomic.Bool
	exitRequested            atomic.Bool
}

// ServerOpts holds the options to create a new Server.
/*type ServerOpts struct {
	C3Version   option.Option[string]
	C3CPath     option.Option[string]
	LogFilepath option.Option[string]

	DiagnosticsDelay   time.Duration
	DiagnosticsEnabled bool

	SendCrashReports bool
	Debug            bool
}*/

func NewServer(opts ServerOpts, appName string, version string) *Server {
	// Read all C3LSP_* env vars once at startup.
	opts.Runtime = loadRuntimeOpts()

	// Remember whether the CLI set --debug so JSON can't overwrite it.
	cliDebug := opts.Debug

	// Apply rotation defaults for any field that was left at zero value.
	defaults := DefaultLogRotationOpts()
	if opts.LogRotation.MaxSizeMB <= 0 {
		opts.LogRotation.MaxSizeMB = defaults.MaxSizeMB
	}
	if opts.LogRotation.MaxBackups <= 0 {
		opts.LogRotation.MaxBackups = defaults.MaxBackups
	}
	// MaxAgeDays == 0 is a valid "no age limit" value; leave it as-is.

	// Set up initial logging.  A temporary Server stub is used so that
	// reconfigureLoggerOutput can read opts.Debug and opts.LogRotation.
	stub := &Server{options: opts}
	stub.reconfigureLoggerOutput(opts.LogFilepath)

	logger := commonlog.GetLogger(appName)
	logger.Info("server starting", "app", appName, "version", version)

	if executable, err := os.Executable(); err == nil {
		logger.Info("server executable", "path", executable)
	}

	if opts.SendCrashReports {
		logger.Debug("crash reports enabled")
	} else {
		logger.Debug("crash reports disabled")
	}

	if opts.C3.Version.IsSome() {
		logger.Info("C3 language version specified", "version", opts.C3.Version.Get())
	}

	handler := &protocol.Handler{}
	extendedHandler := &protocolHandlerWithExtensions{base: handler}
	glspServer := glspserv.NewServer(extendedHandler, appName, true)

	requestedLanguageVersion := checkRequestedLanguageVersion(logger, opts.C3.Version)

	state := l.NewProjectState(logger, option.Some(requestedLanguageVersion), opts.Debug)
	parser := p.NewParser(logger)

	// Instantiate search implementation based on feature flag
	var searchImpl search.SearchInterface
	if utils.IsFeatureEnabled("USE_SEARCH_V2") {
		logger.Info("Using SearchV2 implementation (new architecture)")
		searchImpl = search_v2.NewSearchV2(logger, opts.Debug)
	} else {
		logger.Info("Using original Search implementation")
		searchV1 := search.NewSearch(logger, opts.Debug)
		searchImpl = &searchV1
	}

	server := &Server{
		server:  glspServer,
		options: opts,
		version: version,

		state:  &state,
		parser: &parser,
		search: searchImpl,

		workspaceC3Options: cloneC3Opts(opts.C3),
		idx: indexingCoordinator{
			indexed:     make(map[string]bool),
			indexing:    make(map[string]bool),
			cancels:     make(map[string]context.CancelFunc),
			rootStates:  make(map[string]workspaceIndexState),
			generations: make(map[string]uint64),
		},
		rootCache: projectRootCacheState{
			cache: make(map[string]string),
		},
		renameWarningRoots:   make(map[string]bool),
		completionCache:      make(map[completionCacheKey][]completionItemWithLabelDetails, completionCacheMaxEntries),
		completionCacheOrder: make([]completionCacheKey, 0, completionCacheMaxEntries),
		diag: diagnosticsCoordinator{
			workers:         make(map[string]*diagnosticsWorkerState),
			saveDocVersions: make(map[string]int32),
			workerIdleTTL:   2 * time.Minute,
		},
		normalizedDocIDCache:     make(map[string]string),
		diagnosticsCommand:       c3c.CheckC3ErrorsCommandContext,
		workDoneProgressActive:   make(map[string]struct{}),
		workDoneProgressCanceled: make(map[string]struct{}),
		importPreloadDone:        make(map[string]struct{}),
		cliDebug:                 cliDebug,
	}
	if limit := opts.Runtime.RequestMaxInflight; limit > 0 {
		server.gate.slots = make(chan struct{}, limit)
	}
	server.resetDiagnosticsSchedulers()

	handler.Initialized = func(context *glsp.Context, params *protocol.InitializedParams) error {
		server.initialized.Store(true)
		/*
			context.Notify(protocol.ServerWorkspaceWorkspaceFolders, protocol.PublishDiagnosticsParams{
				URI:         doc.URI,
				Diagnostics: diagnostics,
			})*/
		/*sendCrashStatus := "disabled"
		if opts.SendCrashReports {
			sendCrashStatus = "enabled"
		}

		context.Notify(protocol.ServerWindowShowMessage, protocol.ShowMessageParams{
			Type:    protocol.MessageTypeInfo,
			Message: fmt.Sprintf("SendCrash: %s", sendCrashStatus),
		})
		*/
		return nil
	}
	handler.Shutdown = server.shutdown
	handler.Exit = server.exit
	handler.SetTrace = setTrace
	handler.WindowWorkDoneProgressCancel = server.WindowWorkDoneProgressCancel

	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		if params.Trace != nil {
			protocol.SetTraceValue(*params.Trace)
			notifyLogTrace(context, fmt.Sprintf("Initial trace set to %s", protocol.GetTraceValue()), nil)
		}

		capabilities := handler.CreateServerCapabilities()
		return server.Initialize(
			appName,
			server.version,
			capabilities,
			context,
			params,
		)
	}

	handler.TextDocumentDidOpen = server.TextDocumentDidOpen
	handler.TextDocumentDidChange = server.TextDocumentDidChange
	handler.TextDocumentDidClose = server.TextDocumentDidClose
	handler.TextDocumentDidSave = server.TextDocumentDidSave
	handler.TextDocumentWillSave = server.TextDocumentWillSave
	handler.TextDocumentWillSaveWaitUntil = func(glspContext *glsp.Context, params *protocol.WillSaveTextDocumentParams) (result []protocol.TextEdit, err error) {
		if params == nil {
			return []protocol.TextEdit{}, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s", params.TextDocument.URI))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentWillSaveWaitUntil, details, []protocol.TextEdit{}, func(_ context.Context) ([]protocol.TextEdit, error) {
			return server.TextDocumentWillSaveWaitUntil(glspContext, params)
		})
	}
	handler.TextDocumentHover = func(glspContext *glsp.Context, params *protocol.HoverParams) (result *protocol.Hover, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentHover, details, (*protocol.Hover)(nil), func(requestCtx context.Context) (*protocol.Hover, error) {
			return server.textDocumentHoverWithTrace(glspContext, params, details, requestCtx)
		})
	}
	handler.TextDocumentDeclaration = func(glspContext *glsp.Context, params *protocol.DeclarationParams) (result any, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentDeclaration, details, any(nil), func(_ context.Context) (any, error) {
			return server.TextDocumentDeclaration(glspContext, params)
		})
	}
	handler.TextDocumentDefinition = func(glspContext *glsp.Context, params *protocol.DefinitionParams) (result any, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentDefinition, details, any(nil), func(requestCtx context.Context) (any, error) {
			return server.textDocumentDefinitionWithTrace(glspContext, params, details, requestCtx)
		})
	}
	handler.TextDocumentTypeDefinition = func(glspContext *glsp.Context, params *protocol.TypeDefinitionParams) (result any, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentTypeDefinition, details, any(nil), func(_ context.Context) (any, error) {
			return server.TextDocumentTypeDefinition(glspContext, params)
		})
	}
	handler.TextDocumentImplementation = func(glspContext *glsp.Context, params *protocol.ImplementationParams) (result any, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentImplementation, details, any(nil), func(_ context.Context) (any, error) {
			return server.TextDocumentImplementation(glspContext, params)
		})
	}
	handler.TextDocumentReferences = func(glspContext *glsp.Context, params *protocol.ReferenceParams) (result []protocol.Location, err error) {
		if params == nil {
			return []protocol.Location{}, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentReferences, details, []protocol.Location{}, func(_ context.Context) ([]protocol.Location, error) {
			return server.TextDocumentReferences(glspContext, params)
		})
	}
	handler.TextDocumentDocumentHighlight = func(glspContext *glsp.Context, params *protocol.DocumentHighlightParams) (result []protocol.DocumentHighlight, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentDocumentHighlight, details, []protocol.DocumentHighlight{}, func(_ context.Context) ([]protocol.DocumentHighlight, error) {
			return server.TextDocumentDocumentHighlight(glspContext, params)
		})
	}
	handler.TextDocumentDocumentSymbol = func(glspContext *glsp.Context, params *protocol.DocumentSymbolParams) (result any, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s", params.TextDocument.URI))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentDocumentSymbol, details, any(nil), func(_ context.Context) (any, error) {
			return server.TextDocumentDocumentSymbol(glspContext, params)
		})
	}
	handler.TextDocumentCodeAction = func(glspContext *glsp.Context, params *protocol.CodeActionParams) (result any, err error) {
		if params == nil {
			return []protocol.CodeAction{}, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s", params.TextDocument.URI))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentCodeAction, details, any([]protocol.CodeAction{}), func(_ context.Context) (any, error) {
			return server.TextDocumentCodeAction(glspContext, params)
		})
	}
	handler.CodeActionResolve = func(glspContext *glsp.Context, params *protocol.CodeAction) (result *protocol.CodeAction, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID("codeAction/resolve")
		return runGuardedRequest(server, glspContext, protocol.MethodCodeActionResolve, details, (*protocol.CodeAction)(nil), func(_ context.Context) (*protocol.CodeAction, error) {
			return server.CodeActionResolve(glspContext, params)
		})
	}
	handler.TextDocumentFoldingRange = func(glspContext *glsp.Context, params *protocol.FoldingRangeParams) (result []protocol.FoldingRange, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s", params.TextDocument.URI))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentFoldingRange, details, []protocol.FoldingRange{}, func(_ context.Context) ([]protocol.FoldingRange, error) {
			return server.TextDocumentFoldingRange(glspContext, params)
		})
	}
	handler.TextDocumentSelectionRange = func(glspContext *glsp.Context, params *protocol.SelectionRangeParams) (result []protocol.SelectionRange, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s positions=%d", params.TextDocument.URI, len(params.Positions)))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentSelectionRange, details, []protocol.SelectionRange{}, func(_ context.Context) ([]protocol.SelectionRange, error) {
			return server.TextDocumentSelectionRange(glspContext, params)
		})
	}
	handler.TextDocumentLinkedEditingRange = func(glspContext *glsp.Context, params *protocol.LinkedEditingRangeParams) (result *protocol.LinkedEditingRanges, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentLinkedEditingRange, details, (*protocol.LinkedEditingRanges)(nil), func(_ context.Context) (*protocol.LinkedEditingRanges, error) {
			return server.TextDocumentLinkedEditingRange(glspContext, params)
		})
	}
	handler.TextDocumentMoniker = func(glspContext *glsp.Context, params *protocol.MonikerParams) (result []protocol.Moniker, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentMoniker, details, []protocol.Moniker{}, func(_ context.Context) ([]protocol.Moniker, error) {
			return server.TextDocumentMoniker(glspContext, params)
		})
	}
	handler.TextDocumentPrepareCallHierarchy = func(glspContext *glsp.Context, params *protocol.CallHierarchyPrepareParams) (result []protocol.CallHierarchyItem, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentPrepareCallHierarchy, details, []protocol.CallHierarchyItem{}, func(_ context.Context) ([]protocol.CallHierarchyItem, error) {
			return server.TextDocumentPrepareCallHierarchy(glspContext, params)
		})
	}
	handler.CallHierarchyIncomingCalls = func(glspContext *glsp.Context, params *protocol.CallHierarchyIncomingCallsParams) (result []protocol.CallHierarchyIncomingCall, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("item=%s", params.Item.Name))
		return runGuardedRequest(server, glspContext, protocol.MethodCallHierarchyIncomingCalls, details, []protocol.CallHierarchyIncomingCall{}, func(_ context.Context) ([]protocol.CallHierarchyIncomingCall, error) {
			return server.CallHierarchyIncomingCalls(glspContext, params)
		})
	}
	handler.CallHierarchyOutgoingCalls = func(glspContext *glsp.Context, params *protocol.CallHierarchyOutgoingCallsParams) (result []protocol.CallHierarchyOutgoingCall, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("item=%s", params.Item.Name))
		return runGuardedRequest(server, glspContext, protocol.MethodCallHierarchyOutgoingCalls, details, []protocol.CallHierarchyOutgoingCall{}, func(_ context.Context) ([]protocol.CallHierarchyOutgoingCall, error) {
			return server.CallHierarchyOutgoingCalls(glspContext, params)
		})
	}
	handler.TextDocumentPrepareRename = func(glspContext *glsp.Context, params *protocol.PrepareRenameParams) (result any, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentPrepareRename, details, any(nil), func(requestCtx context.Context) (any, error) {
			return server.textDocumentPrepareRenameWithTrace(glspContext, params, details, requestCtx)
		})
	}
	handler.TextDocumentRename = func(glspContext *glsp.Context, params *protocol.RenameParams) (result *protocol.WorkspaceEdit, err error) {
		if params == nil {
			return emptyWorkspaceEdit(), nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d newName=%s", params.TextDocument.URI, params.Position.Line, params.Position.Character, params.NewName))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentRename, details, emptyWorkspaceEdit(), func(_ context.Context) (*protocol.WorkspaceEdit, error) {
			return server.TextDocumentRename(glspContext, params)
		})
	}
	handler.TextDocumentCompletion = func(glspContext *glsp.Context, params *protocol.CompletionParams) (result any, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentCompletion, details, any(nil), func(requestCtx context.Context) (any, error) {
			return server.textDocumentCompletionWithTrace(glspContext, params, details, requestCtx)
		})
	}
	handler.TextDocumentSignatureHelp = func(glspContext *glsp.Context, params *protocol.SignatureHelpParams) (result *protocol.SignatureHelp, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s line=%d char=%d", params.TextDocument.URI, params.Position.Line, params.Position.Character))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentSignatureHelp, details, (*protocol.SignatureHelp)(nil), func(_ context.Context) (*protocol.SignatureHelp, error) {
			return server.TextDocumentSignatureHelp(glspContext, params)
		})
	}
	handler.TextDocumentFormatting = func(glspContext *glsp.Context, params *protocol.DocumentFormattingParams) (result []protocol.TextEdit, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s", params.TextDocument.URI))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentFormatting, details, []protocol.TextEdit{}, func(_ context.Context) ([]protocol.TextEdit, error) {
			return server.TextDocumentFormatting(glspContext, params)
		})
	}
	handler.TextDocumentRangeFormatting = func(glspContext *glsp.Context, params *protocol.DocumentRangeFormattingParams) (result []protocol.TextEdit, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s", params.TextDocument.URI))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentRangeFormatting, details, []protocol.TextEdit{}, func(_ context.Context) ([]protocol.TextEdit, error) {
			return server.TextDocumentRangeFormatting(glspContext, params)
		})
	}
	handler.TextDocumentOnTypeFormatting = func(glspContext *glsp.Context, params *protocol.DocumentOnTypeFormattingParams) (result []protocol.TextEdit, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s", params.TextDocument.URI))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentOnTypeFormatting, details, []protocol.TextEdit{}, func(_ context.Context) ([]protocol.TextEdit, error) {
			return server.TextDocumentOnTypeFormatting(glspContext, params)
		})
	}
	handler.TextDocumentDocumentLink = func(glspContext *glsp.Context, params *protocol.DocumentLinkParams) (result []protocol.DocumentLink, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("uri=%s", params.TextDocument.URI))
		return runGuardedRequest(server, glspContext, protocol.MethodTextDocumentDocumentLink, details, []protocol.DocumentLink{}, func(_ context.Context) ([]protocol.DocumentLink, error) {
			return server.TextDocumentDocumentLink(glspContext, params)
		})
	}
	handler.DocumentLinkResolve = func(glspContext *glsp.Context, params *protocol.DocumentLink) (result *protocol.DocumentLink, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID("documentLink/resolve")
		return runGuardedRequest(server, glspContext, protocol.MethodDocumentLinkResolve, details, (*protocol.DocumentLink)(nil), func(_ context.Context) (*protocol.DocumentLink, error) {
			return server.DocumentLinkResolve(glspContext, params)
		})
	}
	handler.WorkspaceDidChangeWatchedFiles = server.WorkspaceDidChangeWatchedFiles
	handler.WorkspaceWillCreateFiles = func(glspContext *glsp.Context, params *protocol.CreateFilesParams) (result *protocol.WorkspaceEdit, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("create_files=%d", len(params.Files)))
		return runGuardedRequest(server, glspContext, protocol.MethodWorkspaceWillCreateFiles, details, (*protocol.WorkspaceEdit)(nil), func(_ context.Context) (*protocol.WorkspaceEdit, error) {
			return server.WorkspaceWillCreateFiles(glspContext, params)
		})
	}
	handler.WorkspaceDidCreateFiles = server.WorkspaceDidCreateFiles
	handler.WorkspaceWillRenameFiles = func(glspContext *glsp.Context, params *protocol.RenameFilesParams) (result *protocol.WorkspaceEdit, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("rename_files=%d", len(params.Files)))
		return runGuardedRequest(server, glspContext, protocol.MethodWorkspaceWillRenameFiles, details, (*protocol.WorkspaceEdit)(nil), func(_ context.Context) (*protocol.WorkspaceEdit, error) {
			return server.WorkspaceWillRenameFiles(glspContext, params)
		})
	}
	handler.WorkspaceDidChangeConfiguration = server.WorkspaceDidChangeConfiguration
	handler.WorkspaceWillDeleteFiles = func(glspContext *glsp.Context, params *protocol.DeleteFilesParams) (result *protocol.WorkspaceEdit, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("delete_files=%d", len(params.Files)))
		return runGuardedRequest(server, glspContext, protocol.MethodWorkspaceWillDeleteFiles, details, (*protocol.WorkspaceEdit)(nil), func(_ context.Context) (*protocol.WorkspaceEdit, error) {
			return server.WorkspaceWillDeleteFiles(glspContext, params)
		})
	}
	handler.WorkspaceDidDeleteFiles = server.WorkspaceDidDeleteFiles
	handler.WorkspaceDidRenameFiles = server.WorkspaceDidRenameFiles
	handler.WorkspaceSymbol = func(glspContext *glsp.Context, params *protocol.WorkspaceSymbolParams) (result []protocol.SymbolInformation, err error) {
		if params == nil {
			return []protocol.SymbolInformation{}, nil
		}

		details := server.withRequestID(fmt.Sprintf("query=%s", params.Query))
		return runGuardedRequest(server, glspContext, protocol.MethodWorkspaceSymbol, details, []protocol.SymbolInformation{}, func(_ context.Context) ([]protocol.SymbolInformation, error) {
			return server.WorkspaceSymbol(glspContext, params)
		})
	}
	handler.WorkspaceExecuteCommand = func(glspContext *glsp.Context, params *protocol.ExecuteCommandParams) (result any, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID(fmt.Sprintf("command=%s args=%d", params.Command, len(params.Arguments)))
		return runGuardedRequest(server, glspContext, protocol.MethodWorkspaceExecuteCommand, details, any(nil), func(_ context.Context) (any, error) {
			return server.WorkspaceExecuteCommand(glspContext, params)
		})
	}

	handler.CompletionItemResolve = func(glspContext *glsp.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID("completionItem/resolve")
		return runGuardedRequest(server, glspContext, protocol.MethodCompletionItemResolve, details, (*protocol.CompletionItem)(nil), func(_ context.Context) (*protocol.CompletionItem, error) {
			return params, nil
		})
	}

	handler.WorkspaceDidChangeWorkspaceFolders = server.WorkspaceDidChangeWorkspaceFolders
	extendedHandler.workspaceSymbolResolve = func(glspContext *glsp.Context, params *protocol.SymbolInformation) (result *protocol.SymbolInformation, err error) {
		if params == nil {
			return nil, nil
		}

		details := server.withRequestID("workspaceSymbol/resolve")
		return runGuardedRequest(server, glspContext, methodWorkspaceSymbolResolve, details, (*protocol.SymbolInformation)(nil), func(_ context.Context) (*protocol.SymbolInformation, error) {
			return server.WorkspaceSymbolResolve(glspContext, params)
		})
	}

	return server
}

func (s *Server) resetDiagnosticsSchedulers() {
	quickDelay := s.options.Diagnostics.Delay
	if quickDelay <= 0 {
		quickDelay = 200
	}
	saveFullIdle := s.options.Diagnostics.SaveFullIdle
	if saveFullIdle <= 0 {
		saveFullIdle = 10000
	}
	saveFullIdle = clampDiagnosticsMillisOption(s, "Diagnostics.save-full-idle-ms", saveFullIdle, 500, 120000)
	s.options.Diagnostics.SaveFullIdle = saveFullIdle

	fullMinInterval := s.options.Diagnostics.FullMinInterval
	if fullMinInterval <= 0 {
		fullMinInterval = 30000
	}
	fullMinInterval = clampDiagnosticsMillisOption(s, "Diagnostics.full-min-interval-ms", fullMinInterval, 1000, 600000)
	s.options.Diagnostics.FullMinInterval = fullMinInterval

	quickDebounce := quickDelay * time.Millisecond
	fullDebounce := quickDebounce * 4
	if fullDebounce < 800*time.Millisecond {
		fullDebounce = 800 * time.Millisecond
	}
	saveFullDebounce := saveFullIdle * time.Millisecond

	s.diag.quickDebounced = debounce.New(quickDebounce)
	s.diag.fullDebounced = debounce.New(fullDebounce)
	s.diag.saveFullDebounced = debounce.New(saveFullDebounce)
}

func clampDiagnosticsMillisOption(s *Server, label string, value time.Duration, minDuration time.Duration, maxDuration time.Duration) time.Duration {
	clamped := value
	if clamped < minDuration {
		clamped = minDuration
	}
	if clamped > maxDuration {
		clamped = maxDuration
	}

	if clamped != value && s != nil && s.server != nil && s.server.Log != nil {
		s.server.Log.Warning("config value out of range, clamped", "field", label, "value_ms", value, "clamped_ms", clamped)
	}

	return clamped
}

func (s *Server) nextDiagnosticsGeneration() uint64 {
	return s.diag.generation.Add(1)
}

// contextKeyRequestID is a typed key for storing request IDs in context.Context.
type contextKeyRequestID struct{}

// nextRequestID returns the next monotonically increasing request ID.
func (s *Server) nextRequestID() uint64 {
	return s.gate.sequence.Add(1)
}

func (s *Server) withRequestID(details string) string {
	requestID := s.nextRequestID()
	if details == "" {
		return fmt.Sprintf("request_id=%d", requestID)
	}

	return fmt.Sprintf("request_id=%d %s", requestID, details)
}

func (s *Server) isCurrentDiagnosticsGeneration(generation uint64) bool {
	return s.diag.generation.Load() == generation
}

func cloneC3Opts(input c3c.C3Opts) c3c.C3Opts {
	clone := input
	if input.CompileArgs != nil {
		clone.CompileArgs = append([]string(nil), input.CompileArgs...)
	}

	return clone
}

// reconfigureLoggerOutput rewires the commonlog backend to write to the given
// path (with log rotation) or to stderr when no path is set.
//
// Log verbosity is set to Info normally, or Debug when opts.Debug is true.
func (s *Server) reconfigureLoggerOutput(logPath option.Option[string]) {
	verbosity := 1 // Info
	if s.options.Debug {
		verbosity = 2 // Debug
	}

	if logPath.IsSome() {
		rot := s.options.LogRotation
		// Build a fresh backend and wire lumberjack as its writer so that the
		// log file is rotated automatically when it reaches MaxSizeMB.
		backend := simple.NewBackend()
		backend.Writer = &lumberjack.Logger{
			Filename:   logPath.Get(),
			MaxSize:    rot.MaxSizeMB,  // MB before rotation; 0 → lumberjack default (100 MB)
			MaxBackups: rot.MaxBackups, // number of rotated files to keep
			MaxAge:     rot.MaxAgeDays, // days to keep old files; 0 → unlimited
			Compress:   rot.Compress,   // gzip old files
		}
		// Set the verbosity level without calling Configure (which would
		// overwrite our Writer with stderr).
		backend.SetMaxLevel(commonlog.VerbosityToMaxLevel(verbosity))
		commonlog.SetBackend(backend)
	} else {
		// No path — let commonlog route output to stderr.
		commonlog.Configure(verbosity, nil)
	}

	if s != nil && s.server != nil && s.server.Log != nil {
		if logPath.IsSome() {
			rot := s.options.LogRotation
			s.server.Log.Info("logger output reconfigured", "path", logPath.Get(), "max_size_mb", rot.MaxSizeMB, "max_backups", rot.MaxBackups, "max_age_days", rot.MaxAgeDays)
		} else {
			s.server.Log.Info("logger output reconfigured", "dest", "stderr")
		}
	}
}

// Run starts the Language Server in stdio mode.
func (s *Server) Run() error {
	if err := s.server.RunStdio(); err != nil {
		return errors.Wrap(err, "lsp")
	}

	if s != nil && s.exitRequested.Load() && !s.shutdownRequested.Load() {
		return errors.New("lsp exited without shutdown request")
	}

	return nil
}

func (s *Server) shutdown(context *glsp.Context) error {
	if s != nil {
		s.shutdownRequested.Store(true)
	}
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func (s *Server) exit(context *glsp.Context) error {
	if s != nil {
		s.exitRequested.Store(true)
		if !s.shutdownRequested.Load() {
			return errExitBeforeShutdown
		}
	}

	return nil
}

func (s *Server) isReadyForRequests() bool {
	if s == nil {
		return false
	}

	return s.initialized.Load() && !s.shutdownRequested.Load()
}

func (s *Server) shouldProcessNotification(method string) bool {
	if s.isReadyForRequests() {
		return true
	}

	if s != nil && s.server != nil && s.server.Log != nil {
		s.server.Log.Warning("ignoring request before initialize or after shutdown", "method", method)
	}

	return false
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	previous := protocol.GetTraceValue()
	protocol.SetTraceValue(params.Value)

	verbose := fmt.Sprintf("Previous trace value: %s", previous)
	notifyLogTrace(context, fmt.Sprintf("Trace set to %s", protocol.GetTraceValue()), &verbose)

	return nil
}

func notifyLogTrace(context *glsp.Context, message string, verbose *string) {
	if context == nil || protocol.GetTraceValue() == protocol.TraceValueOff {
		return
	}

	params := protocol.LogTraceParams{Message: message}
	if verbose != nil && protocol.GetTraceValue() == protocol.TraceValueVerbose {
		params.Verbose = verbose
	}

	go context.Notify(protocol.MethodLogTrace, params)
}

func (s *Server) recoverRequestPanic(glspCtx *glsp.Context, method string, errp *error) {
	if recovered := recover(); recovered != nil {
		msg := fmt.Sprintf("panic recovered in %s: %v", method, recovered)
		stack := string(debug.Stack())
		if s != nil && s.server != nil && s.server.Log != nil {
			s.server.Log.Error("panic recovered", "method", method, "panic", recovered, "stack", stack)
		} else {
			log.Printf("%s\n%s", msg, stack)
		}
		s.notifyWindowLogMessage(glspCtx, protocol.MessageTypeError, msg)
		*errp = stderrors.New(msg)
	}
}

func checkRequestedLanguageVersion(logger commonlog.Logger, version option.Option[string]) string {
	// Default to supported version if not specified
	if version.IsNone() {
		logger.Info("using default C3 version", "version", l.SupportedC3Version)
		return l.SupportedC3Version
	}

	requestedVersion := version.Get()

	// Warn if requested version doesn't match officially supported version
	if requestedVersion != l.SupportedC3Version {
		logger.Warning("requested C3 version differs from officially supported version",
			"requested", requestedVersion, "supported", l.SupportedC3Version)
		logger.Warning("correct behavior is not guaranteed for unsupported versions", "version", requestedVersion)
	}

	return requestedVersion
}
