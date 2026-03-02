package server

import (
	"fmt"
	"os"
	"time"

	"github.com/bep/debounce"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_v2"
	"github.com/pherrymason/c3-lsp/pkg/option"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
)

type Server struct {
	server  *glspserv.Server
	options ServerOpts
	version string

	state  *l.ProjectState
	parser *p.Parser
	search search.SearchInterface

	diagnosticDebounced func(func())
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
	var logpath *string
	if opts.LogFilepath.IsSome() {
		v := opts.LogFilepath.Get()
		logpath = &v
	}

	commonlog.Configure(2, logpath) // This increases logging verbosity (optional)

	logger := commonlog.GetLogger(appName)
	logger.Infof(fmt.Sprintf("%s version %s", appName, version))

	if executable, err := os.Executable(); err == nil {
		logger.Infof(fmt.Sprintf("Server executable: %s", executable))
	}

	if opts.SendCrashReports {
		logger.Debug("Crash reports enabled")
	} else {
		logger.Debug("Crash reports disabled")
	}

	if opts.C3.Version.IsSome() {
		logger.Infof(fmt.Sprintf("C3 Language version specified: %s", opts.C3.Version.Get()))
	}

	handler := protocol.Handler{}
	glspServer := glspserv.NewServer(&handler, appName, true)

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

		diagnosticDebounced: debounce.New(opts.Diagnostics.Delay * time.Millisecond),
	}

	handler.Initialized = func(context *glsp.Context, params *protocol.InitializedParams) error {
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
	handler.Shutdown = shutdown
	handler.SetTrace = setTrace

	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
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
	handler.TextDocumentHover = server.TextDocumentHover
	handler.TextDocumentDeclaration = server.TextDocumentDeclaration
	handler.TextDocumentDefinition = server.TextDocumentDefinition
	handler.TextDocumentCompletion = server.TextDocumentCompletion
	handler.TextDocumentSignatureHelp = server.TextDocumentSignatureHelp
	handler.WorkspaceDidChangeWatchedFiles = server.WorkspaceDidChangeWatchedFiles
	handler.WorkspaceDidDeleteFiles = server.WorkspaceDidDeleteFiles
	handler.WorkspaceDidRenameFiles = server.WorkspaceDidRenameFiles

	handler.CompletionItemResolve = func(context *glsp.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
		return params, nil
	}

	handler.WorkspaceDidChangeWorkspaceFolders = func(context *glsp.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {

		return nil
	}

	return server
}

// Run starts the Language Server in stdio mode.
func (s *Server) Run() error {
	return errors.Wrap(s.server.RunStdio(), "lsp")
}

func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func checkRequestedLanguageVersion(logger commonlog.Logger, version option.Option[string]) string {
	// Default to supported version if not specified
	if version.IsNone() {
		logger.Infof("Using default C3 version: %s", project_state.SupportedC3Version)
		return project_state.SupportedC3Version
	}

	requestedVersion := version.Get()

	// Warn if requested version doesn't match officially supported version
	if requestedVersion != project_state.SupportedC3Version {
		logger.Warningf("Requested C3 version %s differs from officially supported version %s",
			requestedVersion, project_state.SupportedC3Version)
		logger.Warningf("The LSP will attempt to load stdlib for version %s", requestedVersion)
		logger.Warning("Correct behavior is not guaranteed for unsupported versions.")
	}

	return requestedVersion
}
